package cli

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/Flyrell/hourgit/internal/watch"
	"github.com/spf13/cobra"
)

var statusCmd = LeafCommand{
	Use:   "status",
	Short: "Show current tracking status",
	StrFlags: []StringFlag{
		{Name: "project", Shorthand: "p", Usage: "project name or ID"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, repoDir, err := getContextPaths()
		if err != nil {
			return err
		}
		projectFlag, _ := cmd.Flags().GetString("project")
		return runStatus(cmd, homeDir, repoDir, projectFlag, defaultGitBranch, time.Now)
	},

}.Build()

// defaultGitBranch returns the current git branch name for the given repo directory.
func defaultGitBranch(repoDir string) (string, error) {
	out, err := exec.Command("git", "-C", repoDir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func runStatus(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag string,
	gitBranchFunc func(repoDir string) (string, error),
	nowFunc func() time.Time,
) error {
	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	now := nowFunc()
	w := cmd.OutOrStdout()

	// Project
	_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Project:"), Primary(proj.Name))

	// Resolve the repo directory for branch lookup:
	// If CWD is one of the project's repos, use it; otherwise pick the first assigned repo.
	branchRepoDir := ""
	cfgEntry := project.FindProjectByID(cfg, proj.ID)
	if cfgEntry != nil && len(cfgEntry.Repos) > 0 {
		branchRepoDir = cfgEntry.Repos[0]
		for _, r := range cfgEntry.Repos {
			if r == repoDir {
				branchRepoDir = repoDir
				break
			}
		}
	}

	// Branch
	if branchRepoDir != "" {
		branch, branchErr := gitBranchFunc(branchRepoDir)
		if branchErr == nil && branch != "" {
			_, _ = fmt.Fprintf(w, "%s   %s\n", Silent("Branch:"), Primary(branch))
		}
	}

	// Load all entries once
	entries, err := LoadProjectEntries(homeDir, proj.Slug)
	if err != nil {
		return err
	}

	// Last checkout
	if last := findLastCheckout(entries.Checkouts); last != nil {
		ago := formatDurationAgo(now.Sub(last.Timestamp))
		_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Checked out:"), Text(ago+" ago"))
	}

	// Schedule for today
	schedules := project.GetSchedules(cfg, proj.ID)
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayEnd := todayStart.Add(24*time.Hour - time.Second)
	daySchedules, err := schedule.ExpandSchedules(schedules, todayStart, todayEnd)
	if err != nil {
		return err
	}

	// Find today's schedule
	var todaySchedule *schedule.DaySchedule
	for i := range daySchedules {
		if daySchedules[i].Date.Day() == now.Day() &&
			daySchedules[i].Date.Month() == now.Month() &&
			daySchedules[i].Date.Year() == now.Year() {
			todaySchedule = &daySchedules[i]
			break
		}
	}

	if todaySchedule == nil || len(todaySchedule.Windows) == 0 {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Today:"), Text("not a working day"))
		return nil
	}

	// Compute today's logged time (with activity-aware idle trimming)
	// Expand schedules for the whole month (needed by ComputeDayBudget)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, time.UTC)
	monthSchedules, err := schedule.ExpandSchedules(schedules, monthStart, monthEnd)
	if err != nil {
		return err
	}

	budget := timetrack.ComputeDayBudget(
		entries.Checkouts, entries.Logs, entries.Commits,
		monthSchedules, now, now,
		timetrack.ActivityEntries{Stops: entries.ActivityStops, Starts: entries.ActivityStarts},
	)

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "%s  %s  %s  %s\n",
		Silent("Today:"),
		Primary(entry.FormatMinutes(budget.LoggedMinutes)+" logged"),
		Silent("·"),
		Text(entry.FormatMinutes(budget.RemainingMinutes)+" remaining"),
	)

	// Schedule line
	windowStrs := make([]string, len(todaySchedule.Windows))
	for i, win := range todaySchedule.Windows {
		windowStrs[i] = schedule.FormatTimeRange(win.From.String(), win.To.String())
	}
	_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Schedule:"), Text(strings.Join(windowStrs, ", ")))

	// Tracking state
	active, activeUntil := isWithinSchedule(now, todaySchedule.Windows)
	if active {
		// Format the end time using FormatTimeRange and extracting the "to" part
		untilStr := schedule.FormatTimeRange(activeUntil.String(), activeUntil.String())
		// FormatTimeRange returns "H:MM PM - H:MM PM", take the first part
		untilStr = strings.SplitN(untilStr, " - ", 2)[0]
		_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Tracking:"), Info("active (until "+untilStr+")"))
	} else {
		_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Tracking:"), Warning("inactive (no scheduled hours remaining)"))
	}

	// Watcher state (only when precise mode is enabled)
	if cfgEntry != nil && cfgEntry.Precise {
		daemonRunning, _, _ := watch.IsDaemonRunning(homeDir)
		if daemonRunning {
			_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Watcher:"), Info("active"))
		} else {
			_, _ = fmt.Fprintf(w, "%s  %s\n", Silent("Watcher:"), Warning("stopped (run hourgit to restart)"))
		}
	}

	return nil
}

// findLastCheckout returns the most recent checkout entry, or nil if none.
func findLastCheckout(checkouts []entry.CheckoutEntry) *entry.CheckoutEntry {
	if len(checkouts) == 0 {
		return nil
	}
	sort.Slice(checkouts, func(i, j int) bool {
		return checkouts[i].Timestamp.Before(checkouts[j].Timestamp)
	})
	return &checkouts[len(checkouts)-1]
}

// formatDurationAgo formats a duration into a human-friendly "Xh Ym" string.
func formatDurationAgo(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalMins := int(d.Minutes())
	if totalMins < 1 {
		return "just now"
	}
	hours := totalMins / 60
	mins := totalMins % 60
	if hours > 0 && mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dm", mins)
}

// isWithinSchedule checks if the current time falls within any schedule window.
// Returns true and the end time of the current window if active.
func isWithinSchedule(now time.Time, windows []schedule.TimeWindow) (bool, schedule.TimeOfDay) {
	nowMinutes := now.Hour()*60 + now.Minute()
	for _, w := range windows {
		fromMins := w.From.Hour*60 + w.From.Minute
		toMins := w.To.Hour*60 + w.To.Minute
		if nowMinutes >= fromMins && nowMinutes < toMins {
			return true, w.To
		}
	}
	return false, schedule.TimeOfDay{}
}
