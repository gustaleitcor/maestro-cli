package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"maestro-cli/internal/maestroapi"
)

func sample(t *testing.T) []maestroapi.MachineMetrics {
	t.Helper()
	const data = `[
	 {"name":"orq","kind":"host","status":"ready","system":{"hostname":"orq.example","kernel":"Linux 6.8.0","cpu_model":"AMD EPYC 7763","uptime_seconds":1209600,
	  "cpu":{"cores":4,"percent":12,"per_core":[10,20,5,13],"load":[0.4,0.3,0.2]},
	  "memory":{"total":17179869184,"used":4294967296,"available":12884901888,"cached":2147483648,"swap_total":0,"swap_used":0},
	  "disks":[{"device":"/dev/sda1","mount":"/","total":107374182400,"used":32212254720}],
	  "network":[{"name":"eth0","rx_rate":125000,"tx_rate":64000}],"gpus":[],"temperatures":[{"name":"x86_pkg_temp","c":54}]},"containers":[]},
	 {"name":"Q1","kind":"machine","status":"ready","system":{"hostname":"q1","kernel":"Linux 5.14.0","cpu_model":"Intel Xeon Gold 6130","uptime_seconds":273600,
	  "cpu":{"cores":8,"percent":91.5,"per_core":[100,98,97,95,90,88,80,84],"load":[7.5,6.9,5.1]},
	  "memory":{"total":68719476736,"used":57982058496,"available":10737418240,"cached":4294967296,"swap_total":8589934592,"swap_used":1073741824},
	  "disks":[{"device":"/dev/nvme0n1p2","mount":"/","total":1000000000000,"used":400000000000},{"device":"/dev/sdb1","mount":"/mnt/data","total":4000000000000,"used":3900000000000}],
	  "network":[{"name":"eth0","rx_rate":52428800,"tx_rate":1048576},{"name":"ib0","rx_rate":0,"tx_rate":0}],
	  "gpus":[{"index":0,"name":"NVIDIA A100-SXM4-40GB","percent":87,"memory_used":21474836480,"memory_total":42949672960,"temperature_c":61,"power_w":250.5,"power_limit_w":400},
	          {"index":1,"name":"NVIDIA A100-SXM4-40GB","percent":3,"memory_used":0,"memory_total":42949672960,"temperature_c":35,"power_w":55,"power_limit_w":400}],
	  "temperatures":[]},
	  "containers":[{"name":"maestro-run-31-line-1","user_id":7,"run":31,"line":1,"cpu_percent":640,"memory_used":8589934592,"memory_limit":17179869184},
	                {"name":"maestro-run-31-line-2","user_id":7,"run":31,"line":2,"cpu_percent":12.5,"memory_used":104857600,"memory_limit":0}]},
	 {"name":"Q2","kind":"machine","status":"unreachable","error":"connecting to 10.0.0.2:22: i/o timeout","containers":[]},
	 {"name":"Q3","kind":"machine","status":"ready","system_error":"opening a shell: ssh: rejected: administratively prohibited","containers":[{"name":"maestro-run-9-line-1","user_id":2,"run":9,"line":1,"cpu_percent":50,"memory_used":1048576,"memory_limit":0}]}
	]`
	var machines []maestroapi.MachineMetrics
	if err := json.Unmarshal([]byte(data), &machines); err != nil {
		t.Fatal(err)
	}
	return machines
}

func loadedTop(t *testing.T, width, height int) topModel {
	t.Helper()
	m := topModel{ctx: context.Background(), interval: 2 * time.Second, width: width, height: height}
	updated, _ := m.Update(topDataMsg{machines: sample(t)})
	return updated.(topModel)
}

func pressTop(m topModel, keys ...string) topModel {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		updated, _ := m.Update(msg)
		m = updated.(topModel)
	}
	return m
}

func TestTopOverview(t *testing.T) {
	m := loadedTop(t, 110, 30)
	view := ansi.Strip(m.View())
	t.Logf("\n%s", view)

	for _, want := range []string{"maestro top", "orq", "(the server)", "Q1", "Q2", "unreachable", "i/o timeout", "Q3", "no system information", "administratively prohibited",
		"CPU", "MEM", "GPU", "92%", "54G/64G", "×2", "2 container(s)", "Linux 5.14.0", "up 3d4h"} {
		if !strings.Contains(view, want) {
			t.Errorf("the overview lacks %q", want)
		}
	}
	lines := strings.Split(view, "\n")
	if len(lines) != 30 {
		t.Errorf("drew %d lines in a window of 30", len(lines))
	}
	for i, line := range lines {
		if w := ansi.StringWidth(line); w > 110 {
			t.Errorf("line %d is %d wide in a window of 110: %q", i, w, line)
		}
	}
	if m.selected != "orq" {
		t.Errorf("selected = %q, want the first machine", m.selected)
	}
}

func TestTopSelectionFollowsTheMachineNotThePosition(t *testing.T) {
	m := pressTop(loadedTop(t, 110, 30), "j", "j")
	if m.selected != "Q2" {
		t.Fatalf("selected = %q after two downs", m.selected)
	}
	// A refresh with the list in another order keeps the selection.
	reordered := sample(t)
	reordered[0], reordered[3] = reordered[3], reordered[0]
	updated, _ := m.Update(topDataMsg{machines: reordered})
	if updated.(topModel).selected != "Q2" {
		t.Errorf("a refresh moved the selection to %q", updated.(topModel).selected)
	}
	if got := pressTop(m, "k", "k", "k").selected; got != "Q3" {
		t.Errorf("going up from the first wraps to the last, got %q", got)
	}
}

func TestTopDetail(t *testing.T) {
	m := pressTop(loadedTop(t, 120, 60), "j", "enter") // orq -> Q1 is the second
	m = pressTop(loadedTop(t, 120, 60), "j")
	if m.selected != "Q1" {
		t.Fatalf("selected = %q", m.selected)
	}
	m = pressTop(m, "enter")
	if !m.detail {
		t.Fatal("enter didn't open the machine")
	}
	view := ansi.Strip(m.View())
	t.Logf("\n%s", view)

	for _, want := range []string{"maestro top · Q1", "Intel Xeon Gold 6130", "8 cores", "load 7.50 6.90 5.10", "CPU", "MEMORY", "SWAP", "GPU",
		"NVIDIA A100-SXM4-40GB", "61°C", "250/400W", "VRAM", "NETWORK", "eth0", "↓ 50M/s", "DISKS", "/mnt/data", "CONTAINERS", "31/1", "640%", "8.0G/16G", "maestro-run-31-line-2"} {
		if !strings.Contains(view, want) {
			t.Errorf("the detail lacks %q", want)
		}
	}
	if got := pressTop(m, "esc"); got.detail || got.selected != "Q1" {
		t.Errorf("esc: detail = %v, selected = %q", got.detail, got.selected)
	}
}

func TestTopDetailOfAMachineThatWontRunCommands(t *testing.T) {
	m := pressTop(loadedTop(t, 110, 30), "j", "j", "j", "enter")
	view := ansi.Strip(m.View())
	for _, want := range []string{"can't run commands", "administratively prohibited", "maestro-run-9-line-1"} {
		if !strings.Contains(view, want) {
			t.Errorf("lacks %q:\n%s", want, view)
		}
	}
	m = pressTop(loadedTop(t, 110, 30), "j", "j", "enter")
	if view := ansi.Strip(m.View()); !strings.Contains(view, "i/o timeout") {
		t.Errorf("an unreachable machine's detail doesn't say why:\n%s", view)
	}
}

func TestTopDetailScrolls(t *testing.T) {
	m := pressTop(loadedTop(t, 100, 12), "j", "enter")
	top := ansi.Strip(m.View())
	m = pressTop(m, "down", "down", "down", "down", "down")
	if scrolled := ansi.Strip(m.View()); scrolled == top {
		t.Error("scrolling changed nothing")
	}
	if len(strings.Split(ansi.Strip(m.View()), "\n")) != 12 {
		t.Error("a scrolled detail doesn't fill the window")
	}
	for i := 0; i < 200; i++ {
		m = pressTop(m, "down")
	}
	if strings.TrimSpace(ansi.Strip(m.View())) == "" {
		t.Error("scrolled past everything")
	}
}

func TestTopCopesWithNarrowAndTinyWindows(t *testing.T) {
	for _, size := range [][2]int{{20, 5}, {40, 10}, {60, 24}, {200, 80}, {1, 1}} {
		m := loadedTop(t, size[0], size[1])
		for _, view := range []string{m.View(), pressTop(pressTop(m, "j"), "enter").View()} {
			for i, line := range strings.Split(ansi.Strip(view), "\n") {
				if ansi.StringWidth(line) > size[0] {
					t.Errorf("%v: line %d is %d wide", size, i, ansi.StringWidth(line))
				}
			}
		}
	}
}

func TestTopKeepsWhatItShowedWhenAnUpdateFails(t *testing.T) {
	m := loadedTop(t, 110, 30)
	updated, _ := m.Update(topDataMsg{err: fmt.Errorf("maestro-orq is unreachable")})
	m = updated.(topModel)
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "update failed: maestro-orq is unreachable") || !strings.Contains(view, "Q1") {
		t.Errorf("an update that failed hid the machines:\n%s", view)
	}

	first := topModel{ctx: context.Background(), interval: time.Second, width: 80, height: 20}
	updated, _ = first.Update(topDataMsg{err: fmt.Errorf("no key")})
	if view := ansi.Strip(updated.(topModel).View()); !strings.Contains(view, "Error: no key") {
		t.Errorf("a first fetch that failed says nothing:\n%s", view)
	}
}

func TestTopPauseAndOneMachineDetailMergesIntoWhatIsKnown(t *testing.T) {
	m := pressTop(loadedTop(t, 110, 30), "p")
	if !m.paused {
		t.Fatal("p didn't pause")
	}
	updated, cmd := m.Update(topTickMsg(time.Now()))
	if updated.(topModel).fetching || cmd == nil {
		t.Error("a paused window fetched, or stopped ticking")
	}

	m = pressTop(loadedTop(t, 110, 30), "j", "enter")
	one := sample(t)[1]
	one.System.CPU.Percent = 5
	updated, _ = m.Update(topDataMsg{machines: []maestroapi.MachineMetrics{one}})
	m = updated.(topModel)
	if len(m.machines) != 4 || m.machines[1].System.CPU.Percent != 5 {
		t.Errorf("a one-machine update lost the others or didn't apply: %d machines", len(m.machines))
	}
}

func TestBarShowsLoad(t *testing.T) {
	if got := ansi.Strip(gauge(10, 50)); got != "█████░░░░░" {
		t.Errorf("half a bar = %q", got)
	}
	if got := ansi.Strip(gauge(10, 250)); got != "██████████" {
		t.Errorf("an overfull bar = %q", got)
	}
	if got := ansi.Strip(gauge(10, -5)); got != "░░░░░░░░░░" {
		t.Errorf("a negative bar = %q", got)
	}
}
