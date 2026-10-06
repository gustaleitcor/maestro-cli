package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"maestro-cli/internal/maestroapi"
)

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Commands for the machines containers run on",
}

var machineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the machines, and whether Maestro can reach them now",
	Long: `Lists the machines an administrator added on the Maestro page. Their
names are what maestro.toml refers to them by.`,
	Args: cobra.NoArgs,
	RunE: runMachineList,
}

func init() {
	rootCmd.AddCommand(machineCmd)
	machineCmd.AddCommand(machineListCmd)
}

func runMachineList(cmd *cobra.Command, args []string) error {
	machines, err := maestroapi.ListMachines(cmd.Context(), maestroKey)
	if err != nil {
		return err
	}
	if len(machines) == 0 {
		fmt.Println("No machines yet. An administrator can add them on the Maestro page.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tSLOTS\tPODMAN\tDESCRIPTION")
	for _, m := range machines {
		version := m.PodmanVersion
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n", m.Name, m.Status, m.Slots, version, m.Description)
	}
	return w.Flush()
}
