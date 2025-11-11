package main

import (
	_ "embed"
	"fmt"
	"image/color"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
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

/* ----------  discover port from listening socket  ---------- */

func listeningPort(pid string) int {
	// line that contains the PID
	line := run("sh", "-c", "ss -lntup 2>/dev/null | grep -E '\\bpid='+pid+'\\b'")
	// pick the first *:PORT part
	re := regexp.MustCompile(`:(\d+)\s+.*\bpid=` + pid + `\b`)
	if m := re.FindStringSubmatch(line); len(m) == 2 {
		if p, e := strconv.Atoi(m[1]); e == nil {
			return p
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

//go:embed assets/icon.svg
var appIcon []byte

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
		icon:       fyne.NewStaticResource("postgres.svg", postgresSvg), // see embed section below
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

	logo  *widget.Icon
	title *widget.Label
	stat  *widget.Label
	btn   *widget.Button

	card *fyne.Container
}

func (r *row) buildUI() {
	bg := canvas.NewCircle(color.NRGBA{R: 35, G: 35, B: 35, A: 255})
	r.logo = widget.NewIcon(r.engine.icon)
	r.logo.Resize(fyne.NewSize(24, 24))
	r.title = widget.NewLabel("")
	r.stat = widget.NewLabel("")
	r.btn = widget.NewButton("", nil)

	sizedIcon := container.NewCenter(
		container.NewPadded(
			container.NewStack(bg, r.logo),
		),
	)

	r.card = container.NewGridWithColumns(4,
		sizedIcon,
		container.NewCenter(r.title),
		container.NewCenter(r.stat),
		container.NewCenter(r.btn),
	)
	r.card.Resize(fyne.NewSize(0, 32))
	r.updateUI()
}

func (r *row) refresh() {
	// systemd state
	r.status = run("systemctl", "is-active", r.unit)

	// PID
	r.pid = run("systemctl", "show", r.unit, "-p", "MainPID", "--value")
	if r.pid == "0" {
		r.pid = ""
	}

	// port
	if r.pid != "" {
		r.port = listeningPort(r.pid)
	} else {
		r.port = 0
	}

	// update UI
	r.updateUI()
}

func (r *row) updateUI() {
	status := r.status
	if status == "active" {
		status = "running"
	}
	full := r.engine.name
	if r.port != 0 {
		full = fmt.Sprintf("%s : %d", r.engine.name, r.port)
	}
	r.title.SetText(full)
	r.stat.SetText(status)

	switch r.status {
	case "active":
		r.btn.SetText("Stop")
		r.btn.OnTapped = func() {
			run("pkexec", "systemctl", "stop", r.unit)
			r.refresh()
		}
	case "inactive", "failed":
		r.btn.SetText("Start")
		r.btn.OnTapped = func() {
			run("pkexec", "systemctl", "start", r.unit)
			r.refresh()
		}
	default:
		r.btn.SetText("Restart")
		r.btn.OnTapped = func() {
			run("pkexec", "systemctl", "restart", r.unit)
			r.refresh()
		}
	}
}

/* ----------  main  ---------- */

func main() {
	a := app.New()
	w := a.NewWindow("DB stat")
	// Set application icon
	icon := fyne.NewStaticResource("icon.svg", appIcon)
	w.SetIcon(icon)
	w.Resize(fyne.NewSize(480, 140))
	w.SetFixedSize(true) // <--  disables resize
	w.CenterOnScreen()

	// Then in main after window creation:
	w.SetCloseIntercept(func() {
		// This sometimes helps with window management
		w.Hide()
	})

	grid := container.NewGridWithColumns(1)
	grid.Objects = nil
	var rows []*row

	for _, eng := range engines {
		unit, ok := findUnit(eng.prefixes)
		if !ok {
			continue
		}
		r := &row{engine: eng, unit: unit}
		r.buildUI() // real widgets first
		grid.Add(r.card)
		rows = append(rows, r)
	}

	if len(rows) == 0 {
		grid.Add(widget.NewLabel("No supported database engines found"))
	}

	// w.SetContent(container.NewScroll(grid))

	// initial population
	for _, r := range rows {
		r.refresh()
	}
	scroll := container.NewScroll(grid)
	scroll.SetMinSize(fyne.NewSize(480, 140)) // 140 logical-pixels high list
	w.SetContent(scroll)

	w.ShowAndRun()
}
