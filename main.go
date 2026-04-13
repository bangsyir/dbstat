package main

import (
	"container/list"
	_ "embed"
	"fmt"
	"image/color"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

/* ----------  tiny helper  ---------- */

func run(args ...string) string {
	out, _ := exec.Command(args[0], args[1:]...).Output()
	return strings.TrimSpace(string(out))
}

/* ----------  logger  ---------- */

type logger struct {
	text *container.Scroll
	list *list.List
}

var (
	logInfoColor  = &color.RGBA{R: 16, G: 185, B: 129, A: 255}
	logWarnColor  = &color.RGBA{R: 245, G: 158, B: 11, A: 255}
	logErrorColor = &color.RGBA{R: 239, G: 68, B: 68, A: 255}
)

func newLogger() *logger {
	l := &logger{}
	l.list = list.New()
	vbox := container.NewVBox()
	scroll := container.NewScroll(vbox)
	scroll.Direction = fyne.ScrollVerticalOnly
	l.text = scroll
	return l
}

func (l *logger) Add(level string, levelColor color.Color, format string, args ...interface{}) {
	if l == nil || l.text == nil {
		return
	}
	timestamp := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)
	entry := fmt.Sprintf("[%s] %s: %s", timestamp, level, msg)

	text := canvas.NewText(entry, levelColor)
	text.TextStyle = fyne.TextStyle{Monospace: true}
	text.TextSize = 12

	vbox := l.text.Content.(*fyne.Container)
	children := vbox.Objects
	children = append(children, text)
	vbox.Objects = children

	if l.list.Len() > 100 {
		l.list.Remove(l.list.Front())
	}

	if fyne.CurrentApp() != nil {
		l.text.Refresh()
	}
}

func (l *logger) Info(format string, args ...interface{}) {
	l.Add("INFO", logInfoColor, format, args...)
}

func (l *logger) Warn(format string, args ...interface{}) {
	l.Add("WARN", logWarnColor, format, args...)
}

func (l *logger) Error(format string, args ...interface{}) {
	l.Add("ERROR", logErrorColor, format, args...)
}

var appLogger *logger

/* ----------  discover systemd unit  ---------- */

func findUnit(prefixes []string) (string, bool) {
	for _, p := range prefixes {
		if out := run("systemctl", "list-unit-files", p+"*.service", "-q", "--no-legend"); out != "" {
			return strings.Fields(out)[0], true
		}
	}
	return "", false
}

/* ----------  discover PID from systemd service  ---------- */

func getServicePID(unit string) string {
	pid := run("systemctl", "show", unit, "--property=MainPID", "--value")
	if pid != "" && pid != "0" {
		return pid
	}

	status := run("systemctl", "status", unit)
	if status != "" {
		re := regexp.MustCompile(`Main PID:\s+(\d+)`)
		if matches := re.FindStringSubmatch(status); len(matches) == 2 {
			return matches[1]
		}
	}

	return ""
}

/* ----------  discover port using netstat with PID or service name  ---------- */

func listeningPort(pid string, serviceName string) int {
	serviceName = strings.TrimSuffix(serviceName, ".service")

	if pid != "" && pid != "0" {
		cmd := fmt.Sprintf("netstat -plnt | grep %s", pid)
		out := run("sh", "-c", cmd)

		if out != "" {
			port := extractPortFromNetstat(out)
			if port != 0 {
				return port
			}
		}
	}

	cmd := fmt.Sprintf("netstat -plnt | grep %s", serviceName)
	out := run("sh", "-c", cmd)

	if out != "" {
		port := extractPortFromNetstat(out)
		if port != 0 {
			return port
		}
	}

	switch {
	case strings.Contains(serviceName, "postgres"):
		return 5432
	case strings.Contains(serviceName, "mysql"), strings.Contains(serviceName, "mariadb"):
		return 3306
	case strings.Contains(serviceName, "redis"):
		return 6379
	}

	return 0
}

/* ----------  extract port from netstat output  ---------- */

func extractPortFromNetstat(netstatOutput string) int {
	lines := strings.Split(netstatOutput, "\n")
	for _, line := range lines {
		re := regexp.MustCompile(`:(\d+)\s`)
		if matches := re.FindStringSubmatch(line); len(matches) == 2 {
			if port, err := strconv.Atoi(matches[1]); err == nil {
				return port
			}
		}

		re = regexp.MustCompile(`0\.0\.0\.0:(\d+)`)
		if matches := re.FindStringSubmatch(line); len(matches) == 2 {
			if port, err := strconv.Atoi(matches[1]); err == nil {
				return port
			}
		}

		re = regexp.MustCompile(`127\.0\.0\.1:(\d+)`)
		if matches := re.FindStringSubmatch(line); len(matches) == 2 {
			if port, err := strconv.Atoi(matches[1]); err == nil {
				return port
			}
		}

		re = regexp.MustCompile(`::1:(\d+)`)
		if matches := re.FindStringSubmatch(line); len(matches) > 1 {
			if port, err := strconv.Atoi(matches[1]); err == nil {
				return port
			}
		}
		re = regexp.MustCompile(`\.(\d+)$`)
		if matches := re.FindStringSubmatch(line); len(matches) == 2 {
			if port, err := strconv.Atoi(matches[1]); err == nil {
				return port
			}
		}
	}
	return 0
}

/* ----------  engine descriptor  ---------- */

type engine struct {
	name       string
	unit       string
	prefixes   []string
	processPat string
	icon       fyne.Resource
	color      string
}

//go:embed assets/postgres.svg
var postgresSvg []byte

//go:embed assets/mysql.svg
var mysqlSvg []byte

//go:embed assets/redis.svg
var redisSvg []byte

var engines = []engine{
	{
		name:       "PostgreSQL",
		prefixes:   []string{"postgresql", "postgres"},
		processPat: "postgres",
		icon:       fyne.NewStaticResource("postgres.svg", postgresSvg),
		color:      "#336791",
	},
	{
		name:       "MySQL",
		prefixes:   []string{"mysql", "mariadb"},
		processPat: "mysqld",
		icon:       fyne.NewStaticResource("mysql.svg", mysqlSvg),
		color:      "#00758F",
	},
	{
		name:       "Redis",
		prefixes:   []string{"redis", "redis-server"},
		processPat: "redis-server",
		icon:       fyne.NewStaticResource("redis.svg", redisSvg),
		color:      "#DC382D",
	},
}

/* ----------  service row  ---------- */

type serviceRow struct {
	engine      engine
	unit        string
	pid         string
	port        int
	status      string
	icon        *widget.Icon
	nameLabel   *canvas.Text
	statusLabel *canvas.Text
	portLabel   *canvas.Text
	pidLabel    *canvas.Text
	actionBtn   *widget.Button
	row         *fyne.Container
}

func (s *serviceRow) buildUI() {
	s.icon = widget.NewIcon(s.engine.icon)
	s.icon.Resize(fyne.NewSize(20, 20))

	s.nameLabel = canvas.NewText(s.engine.name, &color.RGBA{R: 204, G: 204, B: 204, A: 255})
	s.nameLabel.TextStyle = fyne.TextStyle{Bold: true}

	s.statusLabel = canvas.NewText("...", &color.RGBA{R: 204, G: 204, B: 204, A: 255})
	s.statusLabel.Alignment = fyne.TextAlignCenter

	s.portLabel = canvas.NewText("Port: -", &color.RGBA{R: 204, G: 204, B: 204, A: 255})
	s.portLabel.Alignment = fyne.TextAlignCenter

	s.pidLabel = canvas.NewText("PID: -", &color.RGBA{R: 204, G: 204, B: 204, A: 255})
	s.pidLabel.Alignment = fyne.TextAlignCenter

	s.actionBtn = widget.NewButton("Start", nil)
	s.actionBtn.Importance = widget.HighImportance

	infoContainer := container.NewGridWithColumns(4,
		s.nameLabel,
		s.statusLabel,
		s.portLabel,
		s.pidLabel,
	)

	s.row = container.NewBorder(nil, nil, s.icon, s.actionBtn, infoContainer)

	s.updateUI()
}

func (s *serviceRow) refresh() {
	oldStatus := s.status
	s.status = run("systemctl", "is-active", s.unit)
	s.pid = getServicePID(s.unit)

	if s.status == "active" {
		s.port = listeningPort(s.pid, s.unit)
	} else {
		s.port = 0
		s.pid = ""
	}

	if oldStatus != s.status && oldStatus != "" {
		switch s.status {
		case "active":
			appLogger.Info("Service %s started", s.engine.name)
		case "inactive":
			appLogger.Info("Service %s stopped", s.engine.name)
		case "failed":
			appLogger.Error("Service %s failed", s.engine.name)
		}
	}

	s.updateUI()
}

func (s *serviceRow) updateUI() {
	var statusColor *color.RGBA
	switch s.status {
	case "active":
		s.statusLabel.Text = "Running"
		statusColor = &color.RGBA{R: 16, G: 185, B: 129, A: 255}
		s.actionBtn.SetText("Stop")
		s.actionBtn.Importance = widget.DangerImportance
		s.actionBtn.OnTapped = func() {
			appLogger.Info("Stopping %s...", s.engine.name)
			run("pkexec", "systemctl", "stop", s.unit)
			time.Sleep(500 * time.Millisecond)
			s.refresh()
		}
	case "inactive":
		s.statusLabel.Text = "Stopped"
		statusColor = &color.RGBA{R: 204, G: 204, B: 204, A: 255}
		s.actionBtn.SetText("Start")
		s.actionBtn.Importance = widget.HighImportance
		s.actionBtn.OnTapped = func() {
			appLogger.Info("Starting %s...", s.engine.name)
			run("pkexec", "systemctl", "start", s.unit)
			time.Sleep(500 * time.Millisecond)
			s.refresh()
		}
	case "failed":
		s.statusLabel.Text = "Failed"
		statusColor = &color.RGBA{R: 239, G: 68, B: 68, A: 255}
		s.actionBtn.SetText("Restart")
		s.actionBtn.Importance = widget.WarningImportance
		s.actionBtn.OnTapped = func() {
			appLogger.Info("Restarting %s...", s.engine.name)
			run("pkexec", "systemctl", "restart", s.unit)
			time.Sleep(500 * time.Millisecond)
			s.refresh()
		}
	default:
		s.statusLabel.Text = "Unknown"
		statusColor = &color.RGBA{R: 204, G: 204, B: 204, A: 255}
		s.actionBtn.SetText("Refresh")
		s.actionBtn.Importance = widget.MediumImportance
		s.actionBtn.OnTapped = func() {
			s.refresh()
		}
	}
	if statusColor != nil {
		s.statusLabel.Color = statusColor
		s.statusLabel.Refresh()
	}

	if s.port != 0 {
		s.portLabel.Text = fmt.Sprintf("Port: %d", s.port)
	} else {
		s.portLabel.Text = "Port: -"
	}
	s.portLabel.Refresh()

	if s.pid != "" {
		s.pidLabel.Text = fmt.Sprintf("PID: %s", s.pid)
	} else {
		s.pidLabel.Text = "PID: -"
	}
	s.pidLabel.Refresh()
}

/* ----------  main  ---------- */

func main() {
	a := app.New()
	appLogger = newLogger()
	appLogger.Info("DBStat starting...")

	w := a.NewWindow("DBStat - Database Services Manager")
	w.Resize(fyne.NewSize(500, 500))
	w.SetFixedSize(false)

	headerTitle := widget.NewLabel("Database Services Manager")
	headerTitle.TextStyle = fyne.TextStyle{Bold: true}
	headerTitle.Alignment = fyne.TextAlignCenter

	refreshBtn := widget.NewButtonWithIcon("Refresh All", theme.ViewRefreshIcon(), nil)

	header := container.NewBorder(nil, nil, headerTitle, refreshBtn)

	var rows []*serviceRow
	serviceList := container.NewVBox()

	for _, eng := range engines {
		unit, ok := findUnit(eng.prefixes)
		if !ok {
			continue
		}
		row := &serviceRow{engine: eng, unit: unit}
		row.buildUI()
		serviceList.Add(row.row)
		rows = append(rows, row)
		appLogger.Info("Detected %s service: %s", eng.name, unit)
	}

	if len(rows) == 0 {
		noServices := widget.NewLabel("No supported database services found")
		noServices.Alignment = fyne.TextAlignCenter
		noServices.TextStyle = fyne.TextStyle{Bold: true}
		serviceList.Add(container.NewCenter(noServices))
		appLogger.Warn("No database services detected")
	}

	scroll := container.NewScroll(serviceList)
	scroll.SetMinSize(fyne.NewSize(0, 200))

	logTitle := widget.NewLabel("Activity Log")
	logTitle.TextStyle = fyne.TextStyle{Bold: true}

	clearLogBtn := widget.NewButtonWithIcon("Clear", theme.DeleteIcon(), nil)
	clearLogBtn.Importance = widget.LowImportance

	logHeader := container.NewBorder(nil, nil, logTitle, clearLogBtn)

	logScrollBg := canvas.NewRectangle(fyne.CurrentApp().Settings().Theme().Color("background", 0))
	logScrollBg.SetMinSize(fyne.NewSize(0, 250))
	logScrollContainer := container.NewStack(
		logScrollBg,
		appLogger.text,
	)

	logContent := container.NewVBox(
		logHeader,
		logScrollContainer,
	)

	clearLogBtn.OnTapped = func() {
		if appLogger == nil || appLogger.text == nil {
			return
		}
		vbox := appLogger.text.Content.(*fyne.Container)
		vbox.Objects = nil
		appLogger.text.Refresh()
	}

	refreshBtn.OnTapped = func() {
		appLogger.Info("Refreshing all services...")
		for _, row := range rows {
			row.refresh()
		}
	}

	mainContent := container.NewVBox(
		header,
		widget.NewSeparator(),
		scroll,
		widget.NewSeparator(),
		logContent,
	)

	paddedContent := container.NewPadded(mainContent)
	w.SetContent(paddedContent)

	for _, row := range rows {
		row.refresh()
	}

	w.ShowAndRun()
}
