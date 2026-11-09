package cmd

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/humanize"
	"maestro-cli/internal/maestroapi"
)

var machinesAsJSON bool

// machines was `machine` once, which still works. On its own it lists.
var machinesCmd = &cobra.Command{
	Use:     "machines",
	Aliases: []string{"machine"},
	Short:   "Commands for the machines containers run on",
	Args:    cobra.NoArgs,
	RunE:    runMachinesList,
}

var machinesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the machines, and whether Maestro can reach them now",
	Long: `Lists the machines an administrator added on the Maestro page. Their
names are what maestro.toml refers to them by.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runMachinesList,
}

func init() {
	for _, c := range []*cobra.Command{machinesCmd, machinesListCmd} {
		c.Flags().BoolVar(&machinesAsJSON, "json", false, "print the machines as JSON")
	}
	rootCmd.AddCommand(machinesCmd)
	machinesCmd.AddCommand(machinesListCmd)
}

func runMachinesList(cmd *cobra.Command, args []string) error {
	machines, err := maestroapi.ListMachines(cmd.Context(), maestroKey)
	if err != nil {
		return err
	}
	if machinesAsJSON {
		return writeJSON(os.Stdout, machines)
	}
	return writeMachinesList(os.Stdout, machines)
}

func writeMachinesList(out io.Writer, machines []maestroapi.Machine) error {
	if len(machines) == 0 {
		_, err := fmt.Fprintln(out, "No machines yet. An administrator can add them on the Maestro page.")
		return err
	}
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tCPUS\tMEMORY\tPODMAN\tDESCRIPTION")
	for _, m := range machines {
		version := m.PodmanVersion
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", m.Name, m.Status,
			reserved(m.CPUs, m.ReserveCPUs, func(n int64) string { return fmt.Sprint(n) }),
			reserved(m.MemoryMB, m.ReserveMemoryMB, humanize.Megabytes),
			version, m.Description)
	}
	return w.Flush()
}

// reserved is what a machine has and what of it is kept back from lines,
// "16 (1 kept)", each written by unit; "-" while its size isn't known.
func reserved(total, reserve int64, unit func(int64) string) string {
	switch {
	case total == 0:
		return "-"
	case reserve == 0:
		return unit(total)
	}
	return fmt.Sprintf("%s (%s kept)", unit(total), unit(reserve))
}
