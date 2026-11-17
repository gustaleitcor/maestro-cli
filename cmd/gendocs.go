package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var genDocsStamp string

var genDocsCmd = &cobra.Command{
	Use:   "gen-docs <file>",
	Short: "Write the book's CLI reference, as Markdown",
	Long: `Writes every command, its flags and its help to one Markdown page, "-" for standard output. The book is built with it.

--stamp also writes the checksum of what the book is built from, the book
being the one the page is in, as <book>/src/reference/cli.md.`,
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE:   runGenDocs,
}

func init() {
	genDocsCmd.Flags().StringVar(&genDocsStamp, "stamp", "", "the file to write the checksum of the book's sources to")
	rootCmd.AddCommand(genDocsCmd)
}

func runGenDocs(cmd *cobra.Command, args []string) error {
	reference, err := cliReference()
	if err != nil {
		return err
	}
	if args[0] == "-" {
		_, err := cmd.OutOrStdout().Write(reference)
		return err
	}
	if err := os.WriteFile(args[0], reference, 0o644); err != nil {
		return err
	}
	if genDocsStamp == "" {
		return nil
	}
	sum, err := bookSources(filepath.Join(filepath.Dir(args[0]), "..", ".."), reference)
	if err != nil {
		return err
	}
	return os.WriteFile(genDocsStamp, []byte(sum+"\n"), 0o644)
}

// cliReference is the book's page of every command, as this maestro has
// them. help, completion and every command's -h are cobra's own, added as
// commands run: they are all added here, so that the page is the same
// whatever ran before, in a test as in gen-docs.
func cliReference() ([]byte, error) {
	prepare()
	rootCmd.InitDefaultHelpCmd()
	rootCmd.InitDefaultCompletionCmd()
	var helpFlags func(c *cobra.Command)
	helpFlags = func(c *cobra.Command) {
		c.InitDefaultHelpFlag()
		for _, sub := range c.Commands() {
			helpFlags(sub)
		}
	}
	helpFlags(rootCmd)
	var page bytes.Buffer
	err := writeCLIReference(&page, rootCmd)
	return page.Bytes(), err
}

// bookReference is the page gen-docs writes, from the book's src.
const bookReference = "reference/cli.md"

// bookSources is the checksum of what the book in dir is built from: its
// book.toml, every file under its src, and reference, the page of commands
// as written now. The copy of that page on disk is left out, since it is
// only there once the book was built. book/build.sh keeps the checksum
// beside the pages it built, and a test compares the two, so that the book
// maestro carries is never older than a page or a command's help.
//
// A line ending counts as "\n", so that a checkout on Windows gives the
// same checksum.
func bookSources(dir string, reference []byte) (string, error) {
	h := sha256.New()
	add := func(name string, content []byte) {
		content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
		fmt.Fprintf(h, "%s\x00%d\x00", name, len(content))
		h.Write(content)
	}
	config, err := os.ReadFile(filepath.Join(dir, "book.toml"))
	if err != nil {
		return "", err
	}
	add("book.toml", config)
	// In the order of their names, which WalkDir keeps.
	src := os.DirFS(filepath.Join(dir, "src"))
	err = fs.WalkDir(src, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || name == bookReference {
			return err
		}
		content, err := fs.ReadFile(src, name)
		if err != nil {
			return err
		}
		add(name, content)
		return nil
	})
	if err != nil {
		return "", err
	}
	add(bookReference, reference)
	return hex.EncodeToString(h.Sum(nil)), nil
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
