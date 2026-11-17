package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"maestro-cli/internal/humanize"
	"maestro-cli/internal/maestroapi"
	"maestro-cli/tui"
)

var (
	topInterval  time.Duration
	statsAsJSON  bool
	minTopPeriod = time.Second
)

var topCmd = &cobra.Command{
	Use:   "top [machine]",
	Short: "Watch how loaded the machines are, like top",
	Long: `Shows one row for each machine: its CPU, memory, GPUs and load. Under
CORES, 6/14 +3 means lines use 6 of the 14 cores the machine has for them
and 3 more lines wait. The rows refresh every 2 seconds, or as often as -n
says.

Press Enter on a machine to see its cores, memory, disks, network and GPUs,
and the containers on it with whose each one is. Name a machine to open it
straight away.

Anyone signed in sees the machines and their own containers. Administrators
also see every container, and the server Maestro runs on, as "orq".

However many windows are open, Maestro reads each machine at most once
every 2 seconds, and everyone watching shares that reading.`,
	Args:              cobra.MatchAll(cobra.MaximumNArgs(1), flagCheck(checkTopInterval)),
	ValidArgsFunction: completeMachines,
	RunE:              runTop,
}

func checkTopInterval() error {
	if topInterval < minTopPeriod {
		return fmt.Errorf("--interval must be at least %s, because Maestro doesn't read the machines more often than that", minTopPeriod)
	}
	return nil
}

var machinesStatsCmd = &cobra.Command{
	Use:   "stats [machine]",
	Short: "Print how loaded the machines are, once",
	Long: `Prints a table of every machine once: its CPU, memory, GPUs and load,
the cores and memory lines take of what it has for them, with the lines
waiting as +N, and its containers. With --json, it prints everything
Maestro knows, for scripts.

To watch the machines, use ` + "`maestro top`" + `.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeMachines,
	RunE:              runMachinesStats,
}

func init() {
	topCmd.Flags().DurationVarP(&topInterval, "interval", "n", 2*time.Second, "how often to redraw, at least 1s")
	machinesStatsCmd.Flags().BoolVar(&statsAsJSON, "json", false, "print everything as JSON")
	rootCmd.AddCommand(topCmd)
	machinesCmd.AddCommand(machinesStatsCmd)
}

func fetchMetrics(cmd *cobra.Command) tui.Fetch {
	return func(ctx context.Context, only string) ([]maestroapi.MachineMetrics, error) {
		return maestroapi.GetMetrics(ctx, maestroKey, only)
	}
}

func runTop(cmd *cobra.Command, args []string) error {
	only := ""
	if len(args) == 1 {
		only = args[0]
	}
	return tui.RunTop(cmd.Context(), fetchMetrics(cmd), topInterval, only)
}

func runMachinesStats(cmd *cobra.Command, args []string) error {
	only := ""
	if len(args) == 1 {
		only = args[0]
	}
	machines, err := maestroapi.GetMetrics(cmd.Context(), maestroKey, only)
	if err != nil {
		return err
	}

	if statsAsJSON {
		return writeJSON(os.Stdout, machines)
	}
	if len(machines) == 0 {
		fmt.Println("No machines yet. An administrator can add them on the Maestro page.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tCPU\tMEMORY\tGPU\tLOAD\tNET\tUP\tIN USE\tCONTAINERS")
	for _, m := range machines {
		name := m.Name
		if m.Kind == "host" {
			name += " (server)"
		}
		if m.Status != "ready" {
			fmt.Fprintf(w, "%s\t%s\t-\t-\t-\t-\t-\t-\t%s\t-\n", name, m.Status, tui.InUse(m))
			continue
		}
		sys := m.System
		if sys == nil {
			fmt.Fprintf(w, "%s\tready\t-\t-\t-\t-\t-\t-\t%s\t%d\n", name, tui.InUse(m), len(m.Containers))
			continue
		}
		fmt.Fprintf(w, "%s\tready\t%s\t%s/%s\t%s\t%.2f\t%s\t%s\t%s\t%d\n", name,
			humanize.Percent(sys.CPU.Percent),
			humanize.Bytes(sys.Memory.Used), humanize.Bytes(sys.Memory.Total),
			gpuSummary(sys), sys.CPU.Load[0], netSummary(sys),
			humanize.Uptime(sys.UptimeSeconds), tui.InUse(m), len(m.Containers))
	}
	return w.Flush()
}

func gpuSummary(sys *maestroapi.SystemMetrics) string {
	if len(sys.GPUs) == 0 {
		return "-"
	}
	var busy float64
	var used, total uint64
	for _, g := range sys.GPUs {
		busy += g.Percent
		used += g.MemoryUsed
		total += g.MemoryTotal
	}
	return fmt.Sprintf("%s %s/%s ×%d", humanize.Percent(busy/float64(len(sys.GPUs))), humanize.Bytes(used), humanize.Bytes(total), len(sys.GPUs))
}

func netSummary(sys *maestroapi.SystemMetrics) string {
	var rx, tx float64
	for _, n := range sys.Network {
		rx += n.RxRate
		tx += n.TxRate
	}
	return strings.TrimSpace(fmt.Sprintf("↓%s ↑%s", humanize.Rate(rx), humanize.Rate(tx)))
}
