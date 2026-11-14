package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var genDocsCmd = &cobra.Command{
	Use:    "gen-docs <file>",
	Short:  "Write the book's CLI reference, as Markdown",
	Long:   `Writes every command, its flags and its help to one Markdown page, "-" for stdout. The book is built with it.`,
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   runGenDocs,
}

func init() {
	rootCmd.AddCommand(genDocsCmd)
}

func runGenDocs(cmd *cobra.Command, args []string) error {
	if args[0] == "-" {
		return writeCLIReference(cmd.OutOrStdout(), rootCmd)
	}
	f, err := os.Create(args[0])
	if err != nil {
		return err
	}
	w := bufio.NewWriter(f)
	if err := writeCLIReference(w, rootCmd); err != nil {
		f.Close()
		return err
	}
	if err := w.Flush(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// writeCLIReference writes one heading per command, in the order help lists
// them, each a link target named as commandAnchor says, for `maestro docs`.
func writeCLIReference(w io.Writer, root *cobra.Command) error {
	var b strings.Builder
	b.WriteString("# Every command\n\n")
	b.WriteString("Written by the CLI itself when the book is built, so it says what this\n")
	b.WriteString("version of `maestro` takes. `maestro <command> --help` says the same in a\n")
	b.WriteString("terminal.\n\n")
	b.WriteString("Every command also takes `-h, --help`, and these:\n\n")
	fmt.Fprintf(&b, "```text\n%s\n```\n\n", strings.TrimRight(root.PersistentFlags().FlagUsages(), "\n"))
	for _, c := range root.Commands() {
		writeCommandReference(&b, c)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func writeCommandReference(b *strings.Builder, c *cobra.Command) {
	// Not hidden or deprecated commands, and not help, which every one has.
	if !c.IsAvailableCommand() {
		return
	}
	fmt.Fprintf(b, "## %s\n\n", c.CommandPath())
	fmt.Fprintf(b, "%s.\n\n", escapeMarkdown(strings.TrimSuffix(c.Short, ".")))
	fmt.Fprintf(b, "```text\n%s\n```\n\n", c.UseLine())
	if c.Long != "" && c.Long != c.Short {
		fmt.Fprintf(b, "%s\n\n", helpText(c))
	}
	if len(c.Aliases) > 0 {
		fmt.Fprintf(b, "Also: %s.\n\n", "`"+strings.Join(c.Aliases, "`, `")+"`")
	}
	if c.Example != "" {
		var lines []string
		for _, line := range strings.Split(strings.TrimRight(c.Example, "\n"), "\n") {
			lines = append(lines, strings.TrimPrefix(line, "  "))
		}
		fmt.Fprintf(b, "Examples:\n\n```sh\n%s\n```\n\n", strings.Join(lines, "\n"))
	}
	if flags := strings.TrimRight(c.NonInheritedFlags().FlagUsages(), "\n"); flags != "" {
		fmt.Fprintf(b, "Flags:\n\n```text\n%s\n```\n\n", stripHelpFlag(flags))
	}
	var subs []string
	for _, sub := range c.Commands() {
		if sub.IsAvailableCommand() {
			subs = append(subs, fmt.Sprintf("- [%s](#%s): %s", sub.CommandPath(), commandAnchor(sub), escapeMarkdown(sub.Short)))
		}
	}
	if len(subs) > 0 {
		fmt.Fprintf(b, "Commands:\n\n%s\n\n", strings.Join(subs, "\n"))
	}
	for _, sub := range c.Commands() {
		writeCommandReference(b, sub)
	}
}

// helpText is a command's Long as Markdown: its indented lines, which are
// examples, become code blocks, and what is left is escaped so that <repo>
// isn't taken for HTML.
func helpText(c *cobra.Command) string {
	var b strings.Builder
	inCode := false
	for _, line := range strings.Split(strings.TrimSpace(c.Long), "\n") {
		// cobra's own commands, such as completion, indent with tabs.
		if strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
			if !inCode {
				b.WriteString("```text\n")
				inCode = true
			}
			b.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "\t"), "  ") + "\n")
			continue
		}
		if inCode {
			b.WriteString("```\n")
			inCode = false
		}
		b.WriteString(escapeMarkdown(line) + "\n")
	}
	if inCode {
		b.WriteString("```\n")
	}
	return strings.TrimSpace(b.String())
}

var htmlEscaper = strings.NewReplacer("<", "&lt;", ">", "&gt;")

// escapeMarkdown escapes what isn't between backticks, where Markdown shows
// <build> as it is.
func escapeMarkdown(s string) string {
	parts := strings.Split(s, "`")
	for i := 0; i < len(parts); i += 2 {
		parts[i] = htmlEscaper.Replace(parts[i])
	}
	return strings.Join(parts, "`")
}

func stripHelpFlag(flags string) string {
	var kept []string
	for _, line := range strings.Split(flags, "\n") {
		if !strings.Contains(line, "--help") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}
