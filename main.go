package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type DBService struct {
	Name        string
	Type        string
	Unit        string
	Status      string
	PID         string
	Port        int
	DefaultPort int
}

type ActivityEntry struct {
	Time  string
	Level string
	Msg   string
	Color color.Color
}

var (
	serviceRows []serviceRowInfo
	activity    []ActivityEntry
	activityBox *fyne.Container
	serviceList *container.Scroll
	isDarkMode  = true
)

type serviceRowInfo struct {
	svc      *DBService
	row      *fyne.Container
	name     *canvas.Text
	status   *canvas.Text
	resource *canvas.Text
	btn      *widget.Button
}

func run(args ...string) string {
	cmd := exec.Command(args[0], args[1:]...)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func findUnit(prefixes []string) string {
	for _, p := range prefixes {
		// exact unit first: redis.service over redis-sentinel.service
		out := run("systemctl", "list-unit-files", p+".service", "-q", "--no-legend")
		if out != "" {
			return strings.Fields(out)[0]
		}
		out = run("systemctl", "list-unit-files", p+"*.service", "-q", "--no-legend")
		if out != "" {
			return strings.Fields(out)[0]
		}
	}
	return ""
}

func getServicePID(unit string) string {
	pid := run("systemctl", "show", unit, "--property=MainPID", "--value")
	if pid != "" && pid != "0" {
		return pid
	}
	status := run("systemctl", "status", unit)
	if status != "" {
		re := regexp.MustCompile(`Main PID:\s+(\d+)`)
		if m := re.FindStringSubmatch(status); len(m) == 2 {
			return m[1]
		}
	}
	return ""
}

func getPort(pid, unit string, defaultPort int) int {
	unit = strings.TrimSuffix(unit, ".service")

	if pid != "" && pid != "0" {
		out := run("sh", "-c", fmt.Sprintf("netstat -plnt 2>/dev/null | grep %s", pid))
		if p := extractPort(out); p > 0 {
			return p
		}
	}

	out := run("sh", "-c", fmt.Sprintf("netstat -plnt 2>/dev/null | grep %s", unit))
	if p := extractPort(out); p > 0 {
		return p
	}

	return defaultPort
}

func extractPort(out string) int {
	if out == "" {
		return 0
	}
	re := regexp.MustCompile(`:(\d+)`)
	matches := re.FindAllStringSubmatch(out, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		if p, err := strconv.Atoi(matches[i][1]); err == nil && p > 1024 {
			return p
		}
	}
	return 0
}

func getServiceStatus(unit string) string {
	return run("systemctl", "is-active", unit)
}

type ServiceConfig struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Prefixes []string `json:"prefixes"`
	Port     int      `json:"port"`
}

func loadServiceConfig() []ServiceConfig {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(configDir, "dbstat", "services.json"))
	if err != nil {
		return nil
	}
	var configs []ServiceConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil
	}
	return configs
}

func detectServices() []*DBService {
	configs := loadServiceConfig()
	if configs == nil {
		// ponytail: built-in defaults when no config file
		configs = []ServiceConfig{
			{"PostgreSQL", "postgres", []string{"postgresql", "postgres"}, 5432},
			{"MySQL", "mysql", []string{"mysql", "mariadb"}, 3306},
			{"Redis", "redis", []string{"redis", "redis-server"}, 6379},
		}
	}

	var result []*DBService
	for _, c := range configs {
		unit := findUnit(c.Prefixes)
		if unit == "" {
			continue
		}
		svc := &DBService{
			Name:        c.Name,
			Type:        c.Type,
			Unit:        unit,
			DefaultPort: c.Port,
		}
		result = append(result, svc)
	}
	return result
}

func refreshService(svc *DBService) {
	svc.Status = getServiceStatus(svc.Unit)
	svc.PID = getServicePID(svc.Unit)

	if svc.Status == "active" {
		svc.Port = getPort(svc.PID, svc.Unit, svc.DefaultPort)
	} else {
		svc.Port = 0
		svc.PID = ""
	}
}

func getResourceUsage(pid string) string {
	if pid == "" || pid == "0" {
		return ""
	}
	out := run("ps", "-p", pid, "-o", "%cpu,%mem", "--no-headers")
	fields := strings.Fields(out)
	if len(fields) >= 2 {
		return fmt.Sprintf("CPU:%s%% MEM:%s%%", fields[0], fields[1])
	}
	return ""
}

func parseColorHex(s string) color.Color {
	var r, g, b uint64
	fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b)
	return &color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func logActivity(msg, level string) {
	c := parseColorHex("#22c55e")
	switch level {
	case "error":
		c = parseColorHex("#ef4444")
	case "warn":
		c = parseColorHex("#f97316")
	}
	entry := ActivityEntry{
		Time:  time.Now().Format("15:04:05"),
		Level: strings.ToUpper(level),
		Msg:   msg,
		Color: c,
	}
	activity = append(activity, entry)
	if len(activity) > 50 {
		activity = activity[len(activity)-50:]
	}
	refreshActivityList()
}

func buildServiceRow(svc *DBService) *serviceRowInfo {
	refreshService(svc)
	nameLabel := canvas.NewText(svc.Name, theme.DefaultTheme().Color(theme.ColorNameForeground, theme.VariantDark))
	nameLabel.TextStyle = fyne.TextStyle{Bold: true}
	nameLabel.TextSize = 12

	statusLabel := canvas.NewText("", parseColorHex("#888888"))
	statusLabel.TextSize = 12

	switch svc.Status {
	case "active":
		statusLabel.Text = fmt.Sprintf("Running :%d", svc.Port)
		statusLabel.Color = parseColorHex("#22c55e")
	case "inactive":
		statusLabel.Text = "Stopped"
		statusLabel.Color = parseColorHex("#ef4444")
	case "failed":
		statusLabel.Text = "Failed"
		statusLabel.Color = parseColorHex("#f97316")
	default:
		statusLabel.Text = "Unknown"
		statusLabel.Color = parseColorHex("#888888")
	}

	// ponytail: resource usage text, blank when inactive
	resourceText := canvas.NewText("", parseColorHex("#888888"))
	resourceText.TextSize = 12
	resourceText.TextStyle = fyne.TextStyle{Monospace: true}

	labelList := container.NewHBox(nameLabel, statusLabel)
	labelPadding := container.New(layout.NewCustomPaddedLayout(0, 4, 10, 0), labelList)
	btnText := "Start"
	btnImp := widget.HighImportance
	if svc.Status == "active" {
		btnText = "Stop"
		btnImp = widget.DangerImportance
	}

	btn := widget.NewButton(btnText, nil)

	btn.Importance = btnImp
	btnContainer := container.NewGridWrap(fyne.NewSize(70, 30), btn)

	leftSide := container.NewHBox(
		labelPadding,
		resourceText,
	)
	row := container.NewBorder(nil, nil, nil, btnContainer, leftSide)

	return &serviceRowInfo{svc: svc, row: row, name: nameLabel, status: statusLabel, resource: resourceText, btn: btn}
}

func refreshServiceRows() {
	for _, info := range serviceRows {
		refreshService(info.svc)
		switch info.svc.Status {
		case "active":
			info.status.Text = fmt.Sprintf("Running :%d", info.svc.Port)
			info.status.Color = parseColorHex("#22c55e")
			info.btn.SetText("Stop")
			info.btn.Importance = widget.DangerImportance
		case "inactive":
			info.status.Text = "Stopped"
			info.status.Color = parseColorHex("#ef4444")
			info.btn.SetText("Start")
			info.btn.Importance = widget.HighImportance
		case "failed":
			info.status.Text = "Failed"
			info.status.Color = parseColorHex("#f97316")
		default:
			info.status.Text = "Unknown"
			info.status.Color = parseColorHex("#888888")
		}
		if info.svc.Status == "active" {
			info.resource.Text = getResourceUsage(info.svc.PID)
		} else {
			info.resource.Text = ""
		}
		info.resource.Refresh()
		info.status.Refresh()
		info.btn.Refresh()
	}
}

func toggleService(info *serviceRowInfo) {
	if info.svc.Status == "active" {
		logActivity("Stopping "+info.svc.Name+"...", "info")
		run("pkexec", "systemctl", "stop", info.svc.Unit)
	} else {
		logActivity("Starting "+info.svc.Name+"...", "info")
		run("pkexec", "systemctl", "start", info.svc.Unit)
	}
	time.Sleep(500 * time.Millisecond)
	refreshServiceRows()
	if info.svc.Status == "active" {
		logActivity(info.svc.Name+" started", "info")
	} else {
		logActivity(info.svc.Name+" stoped", "info")
	}
}

func refreshActivityList() {
	if activityBox == nil {
		return
	}
	activityBox.Objects = nil

	if len(activity) == 0 {
		empty := canvas.NewText("No activity yet", parseColorHex("#888888"))
		empty.TextSize = 14
		activityBox.Add(empty)
	} else {
		for _, entry := range activity {
			text := canvas.NewText(fmt.Sprintf("[%s] %s: %s", entry.Time, entry.Level, entry.Msg), entry.Color)
			text.TextSize = 12
			text.TextStyle = fyne.TextStyle{Monospace: true}
			activityBox.Add(text)
		}
	}
	activityBox.Refresh()
}

func main() {
	a := app.New()
	logActivity("DBStat starting...", "info")

	w := a.NewWindow("DBStat")
	w.Resize(fyne.NewSize(500, 480))

	title := widget.NewLabel("DBStat")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		refreshServiceRows()
		logActivity("Services refreshed", "info")
	})

	themeBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		s := a.Settings()
		if isDarkMode {
			s.SetTheme(theme.LightTheme())
			for _, info := range serviceRows {
				info.name.Color = theme.DefaultTheme().Color(theme.ColorNameForeground, theme.VariantLight)
				info.name.Refresh()
			}
		} else {
			s.SetTheme(theme.DarkTheme())
			for _, info := range serviceRows {
				info.name.Color = theme.DefaultTheme().Color(theme.ColorNameForeground, theme.VariantDark)
				info.name.Refresh()
			}
		}
		isDarkMode = !isDarkMode
	})

	header := container.NewBorder(nil, nil, title, container.NewHBox(refreshBtn, themeBtn))

	svcList := detectServices()
	logActivity(fmt.Sprintf("Found %d services", len(svcList)), "info")

	dbListBox := container.NewVBox()

	if len(svcList) == 0 {
		empty := widget.NewLabel("No database services found")
		empty.TextStyle = fyne.TextStyle{Bold: true}
		dbListBox.Add(container.NewCenter(empty))
		logActivity("No services detected", "warn")
	} else {
		for _, svc := range svcList {
			info := buildServiceRow(svc)
			info.btn.OnTapped = func() { toggleService(info) }
			serviceRows = append(serviceRows, *info)
			dbListBox.Add(info.row)
		}
	}

	serviceList = container.NewScroll(dbListBox)
	serviceList.SetMinSize(fyne.NewSize(0, 120))

	logTitle := widget.NewLabel("Activity Log")
	logTitle.TextStyle = fyne.TextStyle{Bold: true}

	btnClear := widget.NewButton("Clear", func() {
		activity = nil
		refreshActivityList()
		logActivity("Log cleared", "info")
	})

	logHeader := container.NewBorder(nil, nil, logTitle, btnClear)

	activityBox = container.NewVBox()
	activityList := container.NewScroll(activityBox)
	activityList.SetMinSize(fyne.NewSize(0, 150))
	refreshActivityList()

	mainContent := container.NewVBox(
		header,
		widget.NewSeparator(),
		serviceList,
		widget.NewSeparator(),
		logHeader,
		activityList,
	)

	padded := container.NewPadded(mainContent)
	w.SetContent(padded)

	// ponytail: goroutine dies with process, no cleanup needed
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			fyne.Do(refreshServiceRows)
		}
	}()

	w.ShowAndRun()
}
