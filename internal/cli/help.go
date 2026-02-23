package cli

import (
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// Matches section headers like "Usage:", "Available Commands:", "Flags:"
	sectionHeaderRe = regexp.MustCompile(`^[A-Z][A-Za-z ]+:$`)
	// Matches command/alias listings: "  commandname   description text"
	commandListingRe = regexp.MustCompile(`^( {2})(\S+)(\s{2,}.*)$`)
	// Matches flag lines: "  -f, --flag-name type   description"
	flagLineRe = regexp.MustCompile(`^( +)(-.+?)( {2,}.*)$`)
	// Matches footer lines: "Use "..." for more information"
	footerRe = regexp.MustCompile(`^Use "`)
)

// colorizedHelpFunc returns a custom help function that colorizes Cobra's default help output.
func colorizedHelpFunc() func(*cobra.Command, []string) {
	return func(cmd *cobra.Command, args []string) {
		// Save original writer before replacing
		origOut := cmd.OutOrStdout()

		// Generate default help text
		var buf strings.Builder
		cmd.SetOut(&buf)
		cmd.InitDefaultHelpFlag()
		_ = cmd.Usage()
		cmd.SetOut(origOut)

		raw := buf.String()
		lines := strings.Split(raw, "\n")

		var result strings.Builder
		for _, line := range lines {
			result.WriteString(colorizeLine(line))
			result.WriteString("\n")
		}

		// Remove trailing double newline
		output := strings.TrimRight(result.String(), "\n") + "\n"
		cmd.Print(output)
	}
}

// colorizeLine applies color rules to a single line of help output.
func colorizeLine(line string) string {
	// Section headers
	if sectionHeaderRe.MatchString(strings.TrimSpace(line)) {
		return Info(line)
	}

	// Footer lines
	if footerRe.MatchString(strings.TrimSpace(line)) {
		return Silent(line)
	}

	// Flag lines
	if m := flagLineRe.FindStringSubmatch(line); m != nil {
		return m[1] + Primary(m[2]) + Text(m[3])
	}

	// Command listings
	if m := commandListingRe.FindStringSubmatch(line); m != nil {
		return m[1] + Primary(m[2]) + Text(m[3])
	}

	return Text(line)
}
