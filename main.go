package main

import (
	_ "embed"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

/* ----------  tiny helper  ---------- */

func run(args ...string) string {
	out, _ := exec.Command(args[0], args[1:]...).Output()
	return strings.TrimSpace(string(out))
}

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
	// Method 1: Try systemctl show with different properties
	pid := run("systemctl", "show", unit, "--property=MainPID", "--value")
	if pid != "" && pid != "0" {
		return pid
	}

	// Method 2: Try systemctl status and parse output
	status := run("systemctl", "status", unit)
	if status != "" {
		// Look for "Main PID: 1234" in status output
		re := regexp.MustCompile(`Main PID:\s+(\d+)`)
		if matches := re.FindStringSubmatch(status); len(matches) == 2 {
			return matches[1]
		}
	}

	return ""
}

/* ----------  discover port using netstat with PID or service name  ---------- */

func listeningPort(pid string, serviceName string) int {
	// Remove .service suffix if present
	serviceName = strings.TrimSuffix(serviceName, ".service")

	// Method 1: Use netstat with PID (most accurate)
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

	// Method 2: Use netstat with service name (fallback)
	cmd := fmt.Sprintf("netstat -plnt | grep %s", serviceName)
	out := run("sh", "-c", cmd)

	if out != "" {
		port := extractPortFromNetstat(out)
		if port != 0 {
			return port
		}
	}

	// Method 3: Use default ports based on service name
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
		// Look for port pattern in format :5432
		re := regexp.MustCompile(`:(\d+)\s`)
		if matches := re.FindStringSubmatch(line); len(matches) == 2 {
			if port, err := strconv.Atoi(matches[1]); err == nil {
				return port
			}
		}

		// Alternative pattern for different netstat formats
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
	},
	{
		name:       "MySQL",
		prefixes:   []string{"mysql", "mariadb"},
		processPat: "mysqld",
		icon:       fyne.NewStaticResource("mysql.svg", mysqlSvg),
	},
	{
		name:       "Redis",
		prefixes:   []string{"redis", "redis-server"},
		processPat: "redis-server",
		icon:       fyne.NewStaticResource("redis.svg", redisSvg),
	},
}

/* ----------  live data for one row  ---------- */

type row struct {
	engine engine
	unit   string
	pid    string
	port   int
	status string

	logo      *widget.Icon
	title     *widget.Label
	stat      *widget.Label
	btn       *widget.Button
	portLabel *widget.Label

	card *fyne.Container
	bg   *canvas.Rectangle
}

func (r *row) buildUI() {
	// Create a light background for the card
	r.bg = canvas.NewRectangle(fyne.CurrentApp().Settings().Theme().Color("button", 0))
	r.bg.CornerRadius = 8

	r.logo = widget.NewIcon(r.engine.icon)
	r.logo.Resize(fyne.NewSize(32, 32))

	r.title = widget.NewLabel(r.engine.name)
	r.title.TextStyle = fyne.TextStyle{Bold: true}

	r.stat = widget.NewLabel("Checking...")
	r.stat.Alignment = fyne.TextAlignCenter

	r.portLabel = widget.NewLabel("")
	r.portLabel.Alignment = fyne.TextAlignCenter

	r.btn = widget.NewButton("Refresh", nil)
	r.btn.Importance = widget.MediumImportance

	// Content container
	content := container.NewHBox(
		container.NewPadded(container.NewCenter(r.logo)),
		container.NewVBox(
			r.title,
			container.NewHBox(
				widget.NewLabel("Port:"),
				r.portLabel,
			),
		),
		layout.NewSpacer(),
		container.NewVBox(
			container.NewHBox(
				widget.NewLabel("Status:"),
				r.stat,
			),
			r.btn,
		),
	)

	// Card container with background and padding
	r.card = container.NewStack(
		r.bg,
		container.NewPadded(content),
	)

	r.updateUI()
}

func (r *row) refresh() {
	// systemd state
	r.status = run("systemctl", "is-active", r.unit)

	// PID
	r.pid = getServicePID(r.unit)

	// port - using both PID and service name
	if r.status == "active" {
		r.port = listeningPort(r.pid, r.unit)
	} else {
		r.port = 0
	}

	// update UI
	r.updateUI()
}

func (r *row) updateUI() {
	status := r.status
	statusText := status
	var btnText string
	var importance widget.ButtonImportance

	switch r.status {
	case "active":
		statusText = "Running"
		btnText = "Stop"
		importance = widget.DangerImportance
		r.btn.OnTapped = func() {
			run("pkexec", "systemctl", "stop", r.unit)
			r.refresh()
		}
	case "inactive":
		statusText = "Stopped"
		btnText = "Start"
		importance = widget.HighImportance
		r.btn.OnTapped = func() {
			run("pkexec", "systemctl", "start", r.unit)
			r.refresh()
		}
	case "failed":
		statusText = "Failed"
		btnText = "Restart"
		importance = widget.WarningImportance
		r.btn.OnTapped = func() {
			run("pkexec", "systemctl", "restart", r.unit)
			r.refresh()
		}
	default:
		statusText = "Unknown"
		btnText = "Refresh"
		importance = widget.MediumImportance
		r.btn.OnTapped = func() {
			r.refresh()
		}
	}

	r.stat.SetText(statusText)
	r.btn.SetText(btnText)
	r.btn.Importance = importance

	// Update port display
	if r.port != 0 {
		r.portLabel.SetText(fmt.Sprintf("%d", r.port))
	} else {
		r.portLabel.SetText("-")
	}
}

/* ----------  main  ---------- */

func main() {
	a := app.New()
	w := a.NewWindow("Database Manager")
	w.Resize(fyne.NewSize(600, 400))
	w.SetFixedSize(false)

	// Create rows
	var rows []*row
	grid := container.NewGridWithColumns(1)

	// Create rows for each detected engine
	for _, eng := range engines {
		unit, ok := findUnit(eng.prefixes)
		if !ok {
			continue
		}
		r := &row{engine: eng, unit: unit}
		r.buildUI()
		grid.Add(r.card)
		rows = append(rows, r)
	}

	if len(rows) == 0 {
		noServices := widget.NewLabel("No supported database services found")
		noServices.Alignment = fyne.TextAlignCenter
		grid.Add(noServices)
	}

	// Clean header with just title and refresh button
	title := widget.NewLabel("Database Services Manager")
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.Alignment = fyne.TextAlignCenter

	refreshAllBtn := widget.NewButton("Refresh All", nil)

	header := container.NewHBox(
		container.NewCenter(title),
		layout.NewSpacer(),
		refreshAllBtn,
	)

	// Set up refresh all button
	refreshAllBtn.OnTapped = func() {
		for _, r := range rows {
			r.refresh()
		}
	}

	// Initial population
	for _, r := range rows {
		r.refresh()
	}

	// Scrollable content with proper spacing
	scroll := container.NewScroll(grid)
	scroll.SetMinSize(fyne.NewSize(580, 300))

	// Main container with padding and spacing
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		scroll,
	)

	paddedContent := container.NewPadded(content)
	w.SetContent(paddedContent)

	w.ShowAndRun()
}
