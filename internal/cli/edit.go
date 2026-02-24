package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var editCmd = LeafCommand{
	Use:   "edit <hash>",
	Short: "Edit an existing log entry",
	Args:  cobra.ExactArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "yes", Usage: "skip confirmation prompts"},
	},
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
		{Name: "duration", Usage: "new duration (e.g. 30m, 3h, 3h30m)"},
		{Name: "from", Usage: "new start time (e.g. 9am, 14:00)"},
		{Name: "to", Usage: "new end time (e.g. 5pm, 17:00)"},
		{Name: "date", Usage: "new date (YYYY-MM-DD)"},
		{Name: "task", Usage: "new task label (empty string clears it)"},
		{Name: "message", Shorthand: "m", Usage: "new message"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		repoDir, _ := os.Getwd()
		projectFlag, _ := cmd.Flags().GetString("project")
		durationFlag, _ := cmd.Flags().GetString("duration")
		fromFlag, _ := cmd.Flags().GetString("from")
		toFlag, _ := cmd.Flags().GetString("to")
		dateFlag, _ := cmd.Flags().GetString("date")
		taskFlag, _ := cmd.Flags().GetString("task")
		messageFlag, _ := cmd.Flags().GetString("message")
		yesFlag, _ := cmd.Flags().GetBool("yes")

		flagsChanged := map[string]bool{
			"duration": cmd.Flags().Changed("duration"),
			"from":     cmd.Flags().Changed("from"),
			"to":       cmd.Flags().Changed("to"),
			"date":     cmd.Flags().Changed("date"),
			"task":     cmd.Flags().Changed("task"),
			"message":  cmd.Flags().Changed("message"),
		}

		pk := NewPromptKit()
		var confirm ConfirmFunc
		if yesFlag {
			confirm = AlwaysYes()
		} else {
			confirm = pk.Confirm
		}
		return runEdit(cmd, homeDir, repoDir, projectFlag, args[0],
			durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag,
			flagsChanged, pk, confirm, time.Now)
	},
}.Build()

func runEdit(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, hash string,
	durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag string,
	flagsChanged map[string]bool,
	pk PromptKit,
	confirm ConfirmFunc,
	nowFn func() time.Time,
) error {
	// 1. Locate entry
	slug, proj, e, err := locateEntry(homeDir, repoDir, projectFlag, hash)
	if err != nil {
		return err
	}

	original := e

	// 2. Determine mode — check if any edit flag was explicitly set
	anyFlagSet := false
	for _, changed := range flagsChanged {
		if changed {
			anyFlagSet = true
			break
		}
	}

	if anyFlagSet {
		e, err = applyFlagEdits(e, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag, flagsChanged, nowFn)
	} else {
		e, err = applyInteractiveEdits(e, pk)
	}
	if err != nil {
		return err
	}

	// Validate
	if e.Minutes <= 0 {
		return fmt.Errorf("duration must be positive")
	}
	if e.Minutes > 24*60 {
		return fmt.Errorf("cannot log more than 24h in a single entry")
	}
	if e.Message == "" {
		return fmt.Errorf("message is required")
	}

	// Check schedule warnings if time or date changed
	timeChanged := !e.Start.Equal(original.Start) || e.Minutes != original.Minutes
	if timeChanged && proj != nil {
		proceed, err := checkScheduleWarnings(cmd, homeDir, proj, e.Start, e.Minutes, e.ID, confirm)
		if err != nil {
			return err
		}
		if !proceed {
			return nil
		}
	}

	// Check if anything actually changed
	if e.Start.Equal(original.Start) && e.Minutes == original.Minutes &&
		e.Message == original.Message && e.Task == original.Task {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no changes")
		return nil
	}

	// Write (preserves original ID and CreatedAt)
	if err := entry.WriteEntry(homeDir, slug, e); err != nil {
		return err
	}

	// Display before/after diff
	printEditDiff(cmd, original, e)

	return nil
}

// locateEntry finds the entry by hash, trying project flag, repo context, then scanning all projects.
// Returns the slug, project entry (may be nil for cross-project scan), entry, and error.
func locateEntry(homeDir, repoDir, projectFlag, hash string) (string, *project.ProjectEntry, entry.Entry, error) {
	// Try project flag first
	if projectFlag != "" {
		cfg, err := project.ReadConfig(homeDir)
		if err != nil {
			return "", nil, entry.Entry{}, err
		}
		proj := project.ResolveProject(cfg, projectFlag)
		if proj == nil {
			return "", nil, entry.Entry{}, fmt.Errorf("project '%s' not found", projectFlag)
		}
		e, err := entry.ReadEntry(homeDir, proj.Slug, hash)
		if err != nil {
			if entry.IsCheckoutEntry(homeDir, proj.Slug, hash) {
				return "", nil, entry.Entry{}, fmt.Errorf("entry '%s' is a checkout entry and cannot be edited", hash)
			}
			return "", nil, entry.Entry{}, err
		}
		return proj.Slug, proj, e, nil
	}

	// Try repo context
	if repoDir != "" {
		proj, err := ResolveProjectContext(homeDir, repoDir, "")
		if err == nil {
			e, err := entry.ReadEntry(homeDir, proj.Slug, hash)
			if err == nil {
				return proj.Slug, proj, e, nil
			}
			if entry.IsCheckoutEntry(homeDir, proj.Slug, hash) {
				return "", nil, entry.Entry{}, fmt.Errorf("entry '%s' is a checkout entry and cannot be edited", hash)
			}
		}
	}

	// Scan all projects
	found, err := entry.FindEntryAcrossProjects(homeDir, hash)
	if err != nil {
		return "", nil, entry.Entry{}, err
	}

	// Try to resolve the project entry for schedule warnings
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return found.Slug, nil, found.Entry, nil
	}
	proj := findProjectBySlug(cfg, found.Slug)
	return found.Slug, proj, found.Entry, nil
}

// findProjectBySlug looks up a project by its slug.
func findProjectBySlug(cfg *project.Config, slug string) *project.ProjectEntry {
	for i := range cfg.Projects {
		if cfg.Projects[i].Slug == slug {
			return &cfg.Projects[i]
		}
	}
	return nil
}

func applyFlagEdits(
	e entry.Entry,
	durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag string,
	flagsChanged map[string]bool,
	nowFn func() time.Time,
) (entry.Entry, error) {
	hasDuration := flagsChanged["duration"]
	hasFrom := flagsChanged["from"]
	hasTo := flagsChanged["to"]
	hasDate := flagsChanged["date"]

	// Mutual exclusivity
	if hasDuration && (hasFrom || hasTo) {
		return e, fmt.Errorf("--duration and --from/--to are mutually exclusive")
	}

	// Handle date shift
	if hasDate {
		newDate, err := resolveBaseDate(dateFlag, nowFn())
		if err != nil {
			return e, err
		}
		// Preserve time-of-day, change date
		y, m, d := newDate.Date()
		e.Start = time.Date(y, m, d, e.Start.Hour(), e.Start.Minute(), 0, 0, e.Start.Location())
	}

	// Handle time changes
	if hasDuration {
		minutes, err := entry.ParseDuration(durationFlag)
		if err != nil {
			return e, err
		}
		e.Minutes = minutes
	} else if hasFrom || hasTo {
		// Compute current end time
		oldEnd := e.Start.Add(time.Duration(e.Minutes) * time.Minute)
		oldFromStr := e.Start.Format("15:04")
		oldToStr := oldEnd.Format("15:04")

		fromStr := oldFromStr
		if hasFrom {
			fromStr = fromFlag
		}
		toStr := oldToStr
		if hasTo {
			toStr = toFlag
		}

		y, m, d := e.Start.Date()
		baseDate := time.Date(y, m, d, 0, 0, 0, 0, e.Start.Location())
		start, minutes, err := parseFromTo(fromStr, toStr, baseDate)
		if err != nil {
			return e, err
		}
		e.Start = start
		e.Minutes = minutes
	}

	if flagsChanged["task"] {
		e.Task = taskFlag
	}

	if flagsChanged["message"] {
		if messageFlag == "" {
			return e, fmt.Errorf("message is required")
		}
		e.Message = messageFlag
	}

	return e, nil
}

func applyInteractiveEdits(e entry.Entry, pk PromptKit) (entry.Entry, error) {
	if pk.PromptWithDefault == nil {
		return e, fmt.Errorf("interactive mode not available")
	}

	// Date
	dateStr, err := pk.PromptWithDefault("Date (YYYY-MM-DD)", e.Start.Format("2006-01-02"))
	if err != nil {
		return e, err
	}
	newDate, err := resolveBaseDate(dateStr, e.Start)
	if err != nil {
		return e, err
	}

	// From
	fromStr, err := pk.PromptWithDefault("From (e.g. 9am, 14:00)", e.Start.Format("15:04"))
	if err != nil {
		return e, err
	}

	// To
	endTime := e.Start.Add(time.Duration(e.Minutes) * time.Minute)
	toStr, err := pk.PromptWithDefault("To (e.g. 5pm, 17:00)", endTime.Format("15:04"))
	if err != nil {
		return e, err
	}

	start, minutes, err := parseFromTo(fromStr, toStr, newDate)
	if err != nil {
		return e, err
	}
	e.Start = start
	e.Minutes = minutes

	// Task
	taskStr, err := pk.PromptWithDefault("Task", e.Task)
	if err != nil {
		return e, err
	}
	e.Task = taskStr

	// Message
	msgStr, err := pk.PromptWithDefault("Message", e.Message)
	if err != nil {
		return e, err
	}
	e.Message = msgStr

	return e, nil
}

func printEditDiff(cmd *cobra.Command, before, after entry.Entry) {
	w := cmd.OutOrStdout()

	if !before.Start.Equal(after.Start) {
		_, _ = fmt.Fprintf(w, "  date:     %s → %s\n",
			Silent(before.Start.Format("2006-01-02 15:04")),
			Primary(after.Start.Format("2006-01-02 15:04")),
		)
	}

	if before.Minutes != after.Minutes {
		_, _ = fmt.Fprintf(w, "  duration: %s → %s\n",
			Silent(entry.FormatMinutes(before.Minutes)),
			Primary(entry.FormatMinutes(after.Minutes)),
		)
	}

	if before.Task != after.Task {
		oldTask := before.Task
		if oldTask == "" {
			oldTask = "(none)"
		}
		newTask := after.Task
		if newTask == "" {
			newTask = "(none)"
		}
		_, _ = fmt.Fprintf(w, "  task:     %s → %s\n",
			Silent(oldTask),
			Primary(newTask),
		)
	}

	if before.Message != after.Message {
		_, _ = fmt.Fprintf(w, "  message:  %s → %s\n",
			Silent(before.Message),
			Primary(after.Message),
		)
	}

	_, _ = fmt.Fprintf(w, "updated entry %s\n", Silent(after.ID))
}
