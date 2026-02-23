package cli

import (
	"github.com/charmbracelet/huh"
)

// ConfirmFunc prompts the user for confirmation and returns true if confirmed.
type ConfirmFunc func(prompt string) (bool, error)

// NewConfirmFunc creates a ConfirmFunc using huh's interactive confirm component.
func NewConfirmFunc() ConfirmFunc {
	return func(prompt string) (bool, error) {
		var result bool
		err := huh.NewConfirm().
			Title(prompt).
			Value(&result).
			Run()
		return result, err
	}
}

// AlwaysYes returns a ConfirmFunc that always confirms.
func AlwaysYes() ConfirmFunc {
	return func(_ string) (bool, error) {
		return true, nil
	}
}

// PromptFunc prompts the user for free-text input and returns the response.
type PromptFunc func(prompt string) (string, error)

// NewPromptFunc creates a PromptFunc using huh's interactive input component.
func NewPromptFunc() PromptFunc {
	return func(prompt string) (string, error) {
		var result string
		err := huh.NewInput().
			Title(prompt).
			Value(&result).
			Run()
		return result, err
	}
}

// SelectFunc prompts the user to select one option from a list. Returns 0-based index.
type SelectFunc func(title string, options []string) (int, error)

// NewSelectFunc creates a SelectFunc using huh's interactive select component.
func NewSelectFunc() SelectFunc {
	return func(title string, options []string) (int, error) {
		var result int
		opts := make([]huh.Option[int], len(options))
		for i, o := range options {
			opts[i] = huh.NewOption(o, i)
		}
		err := huh.NewSelect[int]().
			Title(title).
			Options(opts...).
			Value(&result).
			Run()
		return result, err
	}
}

// MultiSelectFunc prompts the user to select multiple options. Returns 0-based indices.
type MultiSelectFunc func(title string, options []string) ([]int, error)

// NewMultiSelectFunc creates a MultiSelectFunc using huh's interactive multi-select component.
func NewMultiSelectFunc() MultiSelectFunc {
	return func(title string, options []string) ([]int, error) {
		var result []int
		opts := make([]huh.Option[int], len(options))
		for i, o := range options {
			opts[i] = huh.NewOption(o, i)
		}
		err := huh.NewMultiSelect[int]().
			Title(title).
			Options(opts...).
			Value(&result).
			Run()
		return result, err
	}
}

// PromptKit bundles all prompt function types for dependency injection.
type PromptKit struct {
	Prompt      PromptFunc
	Confirm     ConfirmFunc
	Select      SelectFunc
	MultiSelect MultiSelectFunc
}

// NewPromptKit creates a PromptKit with huh-based interactive implementations.
func NewPromptKit() PromptKit {
	return PromptKit{
		Prompt:      NewPromptFunc(),
		Confirm:     NewConfirmFunc(),
		Select:      NewSelectFunc(),
		MultiSelect: NewMultiSelectFunc(),
	}
}
