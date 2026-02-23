package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ConfirmFunc prompts the user for confirmation and returns true if confirmed.
type ConfirmFunc func(prompt string) (bool, error)

// NewConfirmFunc creates a ConfirmFunc that reads y/N from the given reader/writer.
func NewConfirmFunc(in io.Reader, out io.Writer) ConfirmFunc {
	return func(prompt string) (bool, error) {
		_, _ = fmt.Fprintf(out, "%s [y/N] ", prompt)
		scanner := bufio.NewScanner(in)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return false, err
			}
			return false, nil
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "y" || answer == "yes", nil
	}
}

// AlwaysYes returns a ConfirmFunc that always confirms.
func AlwaysYes() ConfirmFunc {
	return func(_ string) (bool, error) {
		return true, nil
	}
}
