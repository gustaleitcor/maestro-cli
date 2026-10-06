package cmd

import (
	"context"
	"encoding/json"
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
	Long: `Shows every machine's CPU, memory and GPUs, refreshed as you watch. Press
enter on a machine for its cores, memory, disks, network, GPUs and the
containers on it, or name a machine to open it straight away.

Anyone signed in sees the machines and their own containers. Administrators
also see every container, and the server Maestro runs on, as "orq".

Many people watching share one reading: Maestro takes it at most every couple
of seconds, however many windows are open.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeMachines,
	RunE:              runTop,
}

var machineStatsCmd = &cobra.Command{
	Use:   "stats [machine]",
	Short: "Print how loaded the machines are, once",
	Long: `Prints a table of every machine's CPU, memory, GPUs, load and containers
once. With --json, prints everything Maestro knows, for scripts.

For something to watch, use ` + "`maestro top`" + `.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeMachines,
	RunE:              runMachineStats,
}

func init() {
	topCmd.Flags().DurationVarP(&topInterval, "interval", "n", 2*time.Second, "how often to refresh (at least 1s)")
	machineStatsCmd.Flags().BoolVar(&statsAsJSON, "json", false, "print everything as JSON")
	rootCmd.AddCommand(topCmd)
	machineCmd.AddCommand(machineStatsCmd)
}

func fetchMetrics(cmd *cobra.Command) tui.Fetch {
	return func(ctx context.Context, only string) ([]maestroapi.MachineMetrics, error) {
		return maestroapi.GetMetrics(ctx, maestroKey, only)
	}
}

func runTop(cmd *cobra.Command, args []string) error {
	if topInterval < minTopPeriod {
		return fmt.Errorf("--interval must be at least %s: Maestro doesn't read the machines more often than that", minTopPeriod)
	}
	only := ""
	if len(args) == 1 {
		only = args[0]
	}
	return tui.RunTop(cmd.Context(), fetchMetrics(cmd), topInterval, only)
}

func runMachineStats(cmd *cobra.Command, args []string) error {
	only := ""
	if len(args) == 1 {
		only = args[0]
	}
	machines, err := maestroapi.GetMetrics(cmd.Context(), maestroKey, only)
	if err != nil {
		return err
	}

	if statsAsJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(machines)
	}
	if len(machines) == 0 {
		fmt.Println("No machines yet. An administrator can add them on the Maestro page.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tCPU\tMEMORY\tGPU\tLOAD\tNET\tUP\tCONTAINERS")
	for _, m := range machines {
		name := m.Name
		if m.Kind == "host" {
			name += " (server)"
		}
		if m.Status != "ready" {
			fmt.Fprintf(w, "%s\t%s\t-\t-\t-\t-\t-\t-\t-\n", name, m.Status)
			continue
		}
		sys := m.System
		if sys == nil {
			fmt.Fprintf(w, "%s\tready\t-\t-\t-\t-\t-\t-\t%d\n", name, len(m.Containers))
			continue
		}
		fmt.Fprintf(w, "%s\tready\t%s\t%s/%s\t%s\t%.2f\t%s\t%s\t%d\n", name,
			humanize.Percent(sys.CPU.Percent),
			humanize.Bytes(sys.Memory.Used), humanize.Bytes(sys.Memory.Total),
			gpuSummary(sys), sys.CPU.Load[0], netSummary(sys),
			humanize.Uptime(sys.UptimeSeconds), len(m.Containers))
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
