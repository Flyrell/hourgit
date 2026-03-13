package cli

import "fmt"

// validateMode checks that a --mode flag value is valid.
func validateMode(mode string) error {
	if mode != "" && mode != "standard" && mode != "precise" {
		return fmt.Errorf("invalid --mode value %q (supported: standard, precise)", mode)
	}
	return nil
}
