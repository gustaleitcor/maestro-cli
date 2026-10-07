package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"maestro-cli/internal/humanize"
	"maestro-cli/internal/maestroapi"
)

// Fetch reads the machines; only is one machine's name, or empty for all.
type Fetch func(ctx context.Context, only string) ([]maestroapi.MachineMetrics, error)

type topModel struct {
	ctx      context.Context
	fetch    Fetch
	interval time.Duration

	machines []maestroapi.MachineMetrics
	loaded   bool
	fetching bool
	err      error // of the last fetch; what was shown before stays
	updated  time.Time
	paused   bool

	// selected is a machine's name, so that the selection stays on it when
	// the list changes.
	selected string
	detail   bool
	scroll   int // lines of the detail scrolled off the top

	width, height int
}

type (
	topTickMsg time.Time
	topDataMsg struct {
		machines []maestroapi.MachineMetrics
		err      error
	}
)

// RunTop shows the machines like top does, refreshed every interval. With
// only set, it opens on that machine.
func RunTop(ctx context.Context, fetch Fetch, interval time.Duration, only string) error {
	m := topModel{ctx: ctx, fetch: fetch, interval: interval, selected: only, detail: only != ""}
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func (m topModel) Init() tea.Cmd {
	return tea.Batch(m.load(), m.tick())
}

func (m topModel) tick() tea.Cmd {
	return tea.Tick(m.interval, func(t time.Time) tea.Msg { return topTickMsg(t) })
}

// load reads what is wanted: every machine, unless a detail is open, which
// only needs its own.
func (m topModel) load() tea.Cmd {
	only := ""
	if m.detail {
		only = m.selected
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
		defer cancel()
		machines, err := m.fetch(ctx, only)
		return topDataMsg{machines: machines, err: err}
	}
}

func (m topModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case topTickMsg:
		if m.paused || m.fetching {
			return m, m.tick()
		}
		m.fetching = true
		return m, tea.Batch(m.load(), m.tick())

	case topDataMsg:
		m.fetching = false
		m.err = msg.err
		if msg.err != nil {
			return m, nil
		}
		m.loaded, m.updated = true, time.Now()
		if m.detail {
			// A detail reads one machine; keep the others as they were.
			m.machines = merge(m.machines, msg.machines)
		} else {
			m.machines = msg.machines
		}
		if m.find(m.selected) < 0 && len(m.machines) > 0 && !m.detail {
			m.selected = m.machines[0].Name
		}

	case tea.KeyMsg:
		return m.key(msg)
	}
	return m, nil
}

func (m topModel) key(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "left", "h", "backspace":
		if m.detail {
			m.detail, m.scroll = false, 0
			return m, m.load()
		}
		if msg.String() == "esc" {
			return m, tea.Quit
		}
	case "enter", "right", "l":
		if !m.detail && m.find(m.selected) >= 0 {
			m.detail, m.scroll = true, 0
			return m, m.load()
		}
	case "up", "k":
		if m.detail {
			m.scroll = max(0, m.scroll-1)
		} else {
			m = m.move(-1)
		}
	case "down", "j":
		if m.detail {
			m.scroll++
		} else {
			m = m.move(1)
		}
	case "pgup":
		m.scroll = max(0, m.scroll-m.height/2)
	case "pgdown":
		m.scroll += m.height / 2
	case "home", "g":
		m.scroll = 0
	case "p", " ":
		m.paused = !m.paused
	case "r":
		if !m.fetching {
			m.fetching = true
			return m, m.load()
		}
	}
	return m, nil
}

func (m topModel) find(name string) int {
	for i, machine := range m.machines {
		if machine.Name == name {
			return i
		}
	}
	return -1
}

func (m topModel) move(by int) topModel {
	if len(m.machines) == 0 {
		return m
	}
	i := m.find(m.selected)
	if i < 0 {
		i = 0
	} else {
		i = (i + by + len(m.machines)) % len(m.machines)
	}
	m.selected = m.machines[i].Name
	return m
}

func merge(old, fresh []maestroapi.MachineMetrics) []maestroapi.MachineMetrics {
	merged := append([]maestroapi.MachineMetrics(nil), old...)
	for _, f := range fresh {
		replaced := false
		for i := range merged {
			if merged[i].Name == f.Name {
				merged[i], replaced = f, true
			}
		}
		if !replaced {
			merged = append(merged, f)
		}
	}
	return merged
}

var (
	topTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1)
	topDim      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	topBold     = lipgloss.NewStyle().Bold(true)
	topSelected = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("236"))
	topLabel    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	topWarn     = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	topBad      = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	topGood     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
)

func level(percent float64) lipgloss.Color {
	switch {
	case percent >= 85:
		return lipgloss.Color("196")
	case percent >= 60:
		return lipgloss.Color("214")
	}
	return lipgloss.Color("42")
}

func gauge(width int, percent float64) string {
	width = max(width, 3)
	filled := int(percent/100*float64(width) + 0.5)
	filled = max(0, min(width, filled))
	return lipgloss.NewStyle().Foreground(level(percent)).Render(strings.Repeat("█", filled)) +
		topDim.Render(strings.Repeat("░", width-filled))
}

func meter(label string, width int, percent float64, figure string) string {
	return topLabel.Render(fmt.Sprintf("%-5s", label)) + gauge(width, percent) + " " + figure
}

func (m topModel) View() string {
	if m.width == 0 {
		return ""
	}
	var body []string
	switch {
	case m.err != nil && !m.loaded:
		body = []string{"", "  " + topBad.Render("Error: "+m.err.Error())}
	case !m.loaded:
		body = []string{"", "  Reading the machines..."}
	case m.detail:
		body = m.detailLines()
	default:
		body = m.overviewLines()
	}

	// The body fills what the header and the footer leave; a detail taller
	// than that scrolls.
	room := max(m.height-2, 1)
	if m.detail {
		m.scroll = min(m.scroll, max(0, len(body)-room))
		body = body[m.scroll:]
	}
	if len(body) > room {
		body = body[:room]
	}
	for len(body) < room {
		body = append(body, "")
	}
	for i, line := range body {
		body[i] = lipgloss.NewStyle().MaxWidth(m.width).Render(line)
	}
	return m.header() + "\n" + strings.Join(body, "\n") + "\n" + m.footer()
}

func (m topModel) header() string {
	title := "maestro top"
	if m.detail {
		title += " · " + m.selected
	}
	status := ""
	switch {
	case m.err != nil && m.loaded:
		status = topBad.Render("update failed: " + m.err.Error())
	case m.paused:
		status = topBad.Render("paused")
	case m.loaded:
		status = topDim.Render(fmt.Sprintf("every %s · updated %s", m.interval, m.updated.Format("15:04:05")))
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(topTitle.Render(title) + "  " + status)
}

func (m topModel) footer() string {
	keys := "↑/↓ choose · enter open · p pause · r refresh · q quit"
	if m.detail {
		keys = "↑/↓ scroll · esc back · p pause · r refresh · q quit"
	}
	return lipgloss.NewStyle().MaxWidth(m.width).Render(topDim.Render(keys))
}

// cell fits text into exactly width columns, cutting what doesn't fit.
func cell(text string, width int) string {
	if lipgloss.Width(text) > width {
		text = ansi.Truncate(text, width, "…")
	}
	return text + strings.Repeat(" ", max(0, width-lipgloss.Width(text)))
}

// The overview's columns that are as wide whatever the window: what each
// shows next to its bar, and those without one.
const (
	statusWidth = 11 // unreachable
	cpuWidth    = 4  // 100%
	memWidth    = 9  // 8.0G/128G
	gpuWidth    = 7  // 100% ×8
	loadWidth   = 6
	slotsWidth  = 9 // 12/16 +99
	ctrsWidth   = 4
	maxBarWidth = 20
)

// overviewLayout is how wide the overview's columns that depend on the
// window and on the machines are.
type overviewLayout struct {
	name int
	bar  int // 0: no room for bars, only figures
}

func machineLabel(machine maestroapi.MachineMetrics) string {
	if machine.Kind == "host" {
		return machine.Name + " (server)"
	}
	return machine.Name
}

func (m topModel) layout() overviewLayout {
	l := overviewLayout{name: len("NAME")}
	for _, machine := range m.machines {
		l.name = max(l.name, lipgloss.Width(machineLabel(machine)))
	}
	l.name = min(l.name, 24)
	// Three columns, the dot, and two spaces between each pair of the eight.
	fixed := 3 + l.name + statusWidth + cpuWidth + memWidth + gpuWidth + loadWidth + slotsWidth + ctrsWidth + 7*2
	// Each of the three bars takes its width and a space.
	if l.bar = min(maxBarWidth, (m.width-fixed)/3-1); l.bar < 4 {
		l.bar = 0
	}
	return l
}

// measure is a bar, when there is room for one, and a figure after it.
func (l overviewLayout) measure(percent float64, figure string, figureWidth int) string {
	figure = lipgloss.NewStyle().Foreground(level(percent)).Render(figure)
	if l.bar == 0 {
		return cell(figure, figureWidth)
	}
	return gauge(l.bar, percent) + " " + cell(figure, figureWidth)
}

func (l overviewLayout) measureWidth(figureWidth int) int {
	if l.bar == 0 {
		return figureWidth
	}
	return l.bar + 1 + figureWidth
}

func (l overviewLayout) row(dot, name, status, cpu, mem, gpu, load, slots, ctrs string) string {
	return strings.Join([]string{
		" " + dot + " " + cell(name, l.name), cell(status, statusWidth),
		cell(cpu, l.measureWidth(cpuWidth)), cell(mem, l.measureWidth(memWidth)), cell(gpu, l.measureWidth(gpuWidth)),
		cell(load, loadWidth), cell(slots, slotsWidth), ctrs,
	}, "  ")
}

// rowWithReason is a row for a machine with nothing to measure: why takes
// the place of the measures.
func (l overviewLayout) rowWithReason(dot, name, status, reason, slots, ctrs string) string {
	span := l.measureWidth(cpuWidth) + l.measureWidth(memWidth) + l.measureWidth(gpuWidth) + loadWidth + 3*2
	return strings.Join([]string{
		" " + dot + " " + cell(name, l.name), cell(status, statusWidth),
		cell(topDim.Render(strings.Join(strings.Fields(reason), " ")), span),
		cell(slots, slotsWidth), ctrs,
	}, "  ")
}

func (m topModel) overviewLines() []string {
	if len(m.machines) == 0 {
		return []string{"", "  No machines yet. An administrator can add them on the Maestro page."}
	}
	l := m.layout()
	lines := []string{"", topDim.Render(l.row(" ", "NAME", "STATUS", "CPU", "MEM", "GPU", "LOAD", "SLOTS", "CTRS"))}
	for _, machine := range m.machines {
		row := machineRow(machine, l)
		if machine.Name == m.selected {
			row = topSelected.Width(m.width).MaxWidth(m.width).Render(row)
		}
		lines = append(lines, row)
	}
	return lines
}

func machineRow(machine maestroapi.MachineMetrics, l overviewLayout) string {
	name := topBold.Render(machine.Name)
	if machine.Kind == "host" {
		name += topDim.Render(" (server)")
	}
	slots := SlotsInUse(machine)
	if machine.Status != "ready" {
		return l.rowWithReason(topBad.Render("●"), name, topBad.Render("unreachable"), machine.Error, slots, "-")
	}
	ctrs := fmt.Sprintf("%d", len(machine.Containers))
	sys := machine.System
	if sys == nil {
		return l.rowWithReason(topGood.Render("●"), name, "ready", "no system information: "+machine.SystemError, slots, ctrs)
	}

	memPercent := humanize.Fraction(sys.Memory.Used, sys.Memory.Total)
	gpu := "-"
	if len(sys.GPUs) > 0 {
		var busy float64
		for _, g := range sys.GPUs {
			busy += g.Percent
		}
		busy /= float64(len(sys.GPUs))
		gpu = l.measure(busy, fmt.Sprintf("%4s ×%d", humanize.Percent(busy), len(sys.GPUs)), gpuWidth)
	}
	return l.row(topGood.Render("●"), name, "ready",
		l.measure(sys.CPU.Percent, fmt.Sprintf("%4s", humanize.Percent(sys.CPU.Percent)), cpuWidth),
		l.measure(memPercent, humanize.Bytes(sys.Memory.Used)+"/"+humanize.Bytes(sys.Memory.Total), memWidth),
		gpu, fmt.Sprintf("%.2f", sys.CPU.Load[0]), slots, ctrs)
}

// SlotsInUse reads "2/4 +3": two of four slots taken and three lines
// waiting. A machine without slots, the host, has "-".
func SlotsInUse(m maestroapi.MachineMetrics) string {
	if m.Slots == 0 {
		return "-"
	}
	if m.Queued == 0 {
		return fmt.Sprintf("%d/%d", m.Busy, m.Slots)
	}
	return fmt.Sprintf("%d/%d +%d", m.Busy, m.Slots, m.Queued)
}

// owner is the name of whoever a container belongs to, else their email.
func owner(c maestroapi.ContainerMetrics) string {
	switch {
	case c.UserName != "":
		return c.UserName
	case c.UserEmail != "":
		return c.UserEmail
	}
	return fmt.Sprintf("user %d", c.UserID)
}

// source is the repo and ref a container was built from.
func source(c maestroapi.ContainerMetrics) string {
	if c.Repo == "" {
		return "-"
	}
	return c.Repo + "@" + c.Ref
}

// slotsLine says in words what the overview's SLOTS column says in figures.
func slotsLine(machine maestroapi.MachineMetrics) string {
	if machine.Slots == 0 {
		return ""
	}
	return fmt.Sprintf("%d of %d slot(s) in use · %d line(s) queued", machine.Busy, machine.Slots, machine.Queued)
}

func (m topModel) detailLines() []string {
	i := m.find(m.selected)
	if i < 0 {
		return []string{"", "  No machine " + m.selected}
	}
	machine := m.machines[i]
	var lines []string
	add := func(l ...string) { lines = append(lines, l...) }
	section := func(title string) { add("", topLabel.Render(" "+strings.ToUpper(title))) }

	if machine.Status != "ready" {
		add("", " "+topBad.Render("unreachable")+"  "+machine.Error)
		if slots := slotsLine(machine); slots != "" {
			add(" " + topDim.Render(slots))
		}
		return lines
	}
	sys := machine.System
	if sys == nil {
		add("", " "+topDim.Render("Maestro reaches this machine's Podman, but can't run commands on it, so there is nothing to show but its containers."), " "+topDim.Render(machine.SystemError))
		if slots := slotsLine(machine); slots != "" {
			add(" " + topDim.Render(slots))
		}
		return append(lines, m.containerLines(machine)...)
	}

	add("", " "+topBold.Render(firstNonEmpty(sys.Hostname, machine.Name))+topDim.Render(fmt.Sprintf("  %s · up %s", sys.Kernel, humanize.Uptime(sys.UptimeSeconds))))
	if sys.CPUModel != "" {
		add(" " + topDim.Render(fmt.Sprintf("%s · %d cores · load %.2f %.2f %.2f", sys.CPUModel, sys.CPU.Cores, sys.CPU.Load[0], sys.CPU.Load[1], sys.CPU.Load[2])))
	}
	if slots := slotsLine(machine); slots != "" {
		add(" " + topDim.Render(slots))
	}

	section("cpu")
	full := m.width - 18
	add(" " + meter("ALL", max(10, min(full, 60)), sys.CPU.Percent, humanize.Percent(sys.CPU.Percent)))
	columns := max(1, min(4, m.width/30))
	cell := m.width/columns - 1
	var row []string
	for core, percent := range sys.CPU.PerCore {
		row = append(row, lipgloss.NewStyle().Width(cell).Render(meter(fmt.Sprintf("%d", core), max(4, cell-12), percent, fmt.Sprintf("%4s", humanize.Percent(percent)))))
		if len(row) == columns {
			add(" " + strings.Join(row, " "))
			row = nil
		}
	}
	if len(row) > 0 {
		add(" " + strings.Join(row, " "))
	}

	section("memory")
	memory := sys.Memory
	used := humanize.Fraction(memory.Used, memory.Total)
	add(" " + meter("MEM", max(10, min(full, 60)), used, fmt.Sprintf("%s of %s (%s cached)", humanize.Bytes(memory.Used), humanize.Bytes(memory.Total), humanize.Bytes(memory.Cached))))
	if memory.SwapTotal > 0 {
		add(" " + meter("SWAP", max(10, min(full, 60)), humanize.Fraction(memory.SwapUsed, memory.SwapTotal), fmt.Sprintf("%s of %s", humanize.Bytes(memory.SwapUsed), humanize.Bytes(memory.SwapTotal))))
	}

	if len(sys.GPUs) > 0 {
		section("gpu")
		for _, g := range sys.GPUs {
			add(" " + topBold.Render(fmt.Sprintf("%d  %s", g.Index, g.Name)) + topDim.Render(fmt.Sprintf("  %.0f°C%s", g.TempC, power(g.PowerW, g.PowerLimitW))))
			add("   " + meter("GPU", max(10, min(full/2, 40)), g.Percent, humanize.Percent(g.Percent)) + "  " +
				meter("VRAM", max(10, min(full/2, 40)), humanize.Fraction(g.MemoryUsed, g.MemoryTotal), fmt.Sprintf("%s of %s", humanize.Bytes(g.MemoryUsed), humanize.Bytes(g.MemoryTotal))))
		}
	}

	if len(sys.Network) > 0 {
		section("network")
		for _, nic := range sys.Network {
			add(fmt.Sprintf(" %-14s ↓ %-10s ↑ %s", nic.Name, humanize.Rate(nic.RxRate), humanize.Rate(nic.TxRate)))
		}
	}

	if len(sys.Disks) > 0 {
		section("disks")
		for _, d := range sys.Disks {
			percent := humanize.Fraction(d.Used, d.Total)
			add(" " + meter("", max(10, min(full/2, 30)), percent, fmt.Sprintf("%4s %s of %s  %s", humanize.Percent(percent), humanize.Bytes(d.Used), humanize.Bytes(d.Total), d.Mount)))
		}
	}

	if len(sys.Temps) > 0 {
		var temps []string
		for _, t := range sys.Temps {
			temps = append(temps, fmt.Sprintf("%s %.0f°C", t.Name, t.C))
		}
		section("temperatures")
		add(" " + strings.Join(temps, " · "))
	}

	return append(lines, m.containerLines(machine)...)
}

func power(watts, limit float64) string {
	if watts == 0 {
		return ""
	}
	if limit == 0 {
		return fmt.Sprintf(" · %.0fW", watts)
	}
	return fmt.Sprintf(" · %.0f/%.0fW", watts, limit)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (m topModel) containerLines(machine maestroapi.MachineMetrics) []string {
	lines := []string{"", topLabel.Render(" CONTAINERS")}
	if len(machine.Containers) == 0 {
		return append(lines, " "+topDim.Render("none running"))
	}
	ownerWidth, sourceWidth := len("OWNER"), len("REPO@REF")
	for _, c := range machine.Containers {
		ownerWidth, sourceWidth = max(ownerWidth, lipgloss.Width(owner(c))), max(sourceWidth, lipgloss.Width(source(c)))
	}
	ownerWidth, sourceWidth = min(ownerWidth, 24), min(sourceWidth, 32)
	lines = append(lines, topDim.Render(fmt.Sprintf(" %-9s %s %8s %12s  %s  %s", "RUN/LINE", cell("OWNER", ownerWidth), "CPU", "MEMORY", cell("REPO@REF", sourceWidth), "NAME")))
	for _, c := range machine.Containers {
		memory := humanize.Bytes(c.MemUsed)
		if c.MemLimit > 0 {
			memory += "/" + humanize.Bytes(c.MemLimit)
		}
		cpu := lipgloss.NewStyle().Foreground(level(c.Percent)).Render(fmt.Sprintf("%7.0f%%", c.Percent))
		lines = append(lines, fmt.Sprintf(" %-9s %s %s %12s  %s  %s", fmt.Sprintf("%d/%d", c.Run, c.Line), cell(owner(c), ownerWidth), cpu, memory, cell(source(c), sourceWidth), c.Name))
	}
	return lines
}
