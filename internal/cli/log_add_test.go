package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLogAddTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "Log Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, repoDir, proj
}

func fixedNow() time.Time {
	return time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
}

func execLogAdd(homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message string) (string, error) {
	return execLogAddWithPrompts(homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message, PromptKit{
		Confirm: AlwaysYes(),
	})
}

func execLogAddWithPrompts(homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message string, pk PromptKit) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := logAddCmd
	cmd.SetOut(stdout)

	err := runLogAdd(cmd, homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message, pk, fixedNow)
	return stdout.String(), err
}

func TestLogAddDurationMode(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout, err := execLogAdd(homeDir, repoDir, "", "3h", "", "", "", "", "did some work")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "3h")
	assert.Contains(t, stdout, "Log Test")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, "did some work", entries[0].Message)
	assert.Equal(t, 9, entries[0].Start.Hour())
}

func TestLogAddFromToMode(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout, err := execLogAdd(homeDir, repoDir, "", "", "9am", "12pm", "", "", "morning work")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, "morning work", entries[0].Message)
	assert.Equal(t, 9, entries[0].Start.Hour())
}

func TestLogAddByProjectFlag(t *testing.T) {
	homeDir, _, proj := setupLogAddTest(t)

	stdout, err := execLogAdd(homeDir, "", proj.Name, "1h", "", "", "", "", "flagged project")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Log Test")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogAddDurationAndFromToMutuallyExclusive(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	_, err := execLogAdd(homeDir, repoDir, "", "3h", "9am", "12pm", "", "", "conflict")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestLogAddFromWithoutTo(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "To (e.g. 5pm, 17:00)":
				return "12pm", nil
			case "Message":
				return "prompted to", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "9am", "", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, "prompted to", entries[0].Message)
}

func TestLogAddToWithoutFrom(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "From (e.g. 9am, 14:00)":
				return "9am", nil
			case "Message":
				return "prompted from", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "", "12pm", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, "prompted from", entries[0].Message)
}

func TestLogAddFromAfterTo(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	_, err := execLogAdd(homeDir, repoDir, "", "", "5pm", "9am", "", "", "backwards")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be before")
}

func TestLogAddInvalidDuration(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	_, err := execLogAdd(homeDir, repoDir, "", "abc", "", "", "", "", "bad duration")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration")
}

func TestLogAddDurationNoMessage(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			if prompt == "Message" {
				return "prompted msg", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "3h", "", "", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "prompted msg", entries[0].Message)
}

func TestLogAddFromToNoMessage(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			if prompt == "Message" {
				return "prompted msg", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "9am", "5pm", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "8h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "prompted msg", entries[0].Message)
}

func TestLogAddMessageOnly(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Select:  func(_ string, _ []string) (int, error) { return 0, nil },
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "Date (YYYY-MM-DD, default: today)":
				return "", nil
			case "Duration (e.g. 30m, 3h, 3h30m)":
				return "2h", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "", "", "", "", "pre-filled msg", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "2h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "pre-filled msg", entries[0].Message)
}

func TestLogAddDateOnly(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Select:  func(_ string, _ []string) (int, error) { return 0, nil },
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "Duration (e.g. 30m, 3h, 3h30m)":
				return "1h", nil
			case "Message":
				return "date only work", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "", "", "2025-01-10", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "1h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.January, entries[0].Start.Month())
	assert.Equal(t, 10, entries[0].Start.Day())
	assert.Equal(t, "date only work", entries[0].Message)
}

func TestLogAddDurationWithDateNoMessage(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			if prompt == "Message" {
				return "date+dur msg", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "3h", "", "", "2025-01-10", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.January, entries[0].Start.Month())
	assert.Equal(t, "date+dur msg", entries[0].Message)
}

func TestLogAddEmptyMessagePromptedStillRequired(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			return "", nil
		},
	}

	_, err := execLogAddWithPrompts(homeDir, repoDir, "", "1h", "", "", "", "", "", pk)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestLogAddNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execLogAdd(homeDir, "", "", "1h", "", "", "", "", "no project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestLogAddInteractiveModeDuration(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout := new(bytes.Buffer)
	cmd := logAddCmd
	cmd.SetOut(stdout)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Select:  func(_ string, _ []string) (int, error) { return 0, nil },
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "Duration (e.g. 30m, 3h, 3h30m)":
				return "2h", nil
			case "Message":
				return "interactive work", nil
			}
			return "", nil
		},
	}

	err := runLogAdd(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "logged")
	assert.Contains(t, stdout.String(), "2h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 120, entries[0].Minutes)
}

func TestLogAddInteractiveModeFromTo(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout := new(bytes.Buffer)
	cmd := logAddCmd
	cmd.SetOut(stdout)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Select:  func(_ string, _ []string) (int, error) { return 1, nil },
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "From (e.g. 9am, 14:00)":
				return "10am", nil
			case "To (e.g. 5pm, 17:00)":
				return "1pm", nil
			case "Message":
				return "range work", nil
			}
			return "", nil
		},
	}

	err := runLogAdd(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	require.NoError(t, err)

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
}

func TestLogAddInteractiveEmptyMessage(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	stdout := new(bytes.Buffer)
	cmd := logAddCmd
	cmd.SetOut(stdout)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Select:  func(_ string, _ []string) (int, error) { return 0, nil },
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "Duration (e.g. 30m, 3h, 3h30m)":
				return "1h", nil
			case "Message":
				return "", nil
			}
			return "", nil
		},
	}

	err := runLogAdd(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestLogAddDurationModeWithDate(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout, err := execLogAdd(homeDir, repoDir, "", "3h", "", "", "2025-01-10", "", "past work")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	// Duration-only log is placed at the first available schedule slot (9:00)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.January, entries[0].Start.Month())
	assert.Equal(t, 10, entries[0].Start.Day())
	assert.Equal(t, 9, entries[0].Start.Hour())
}

func TestLogAddDurationModeUnscheduledDayFallback(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// 2025-01-11 is a Saturday — no schedule, so findDurationSlot fails
	// and falls back to 9:00 default; the schedule warning system will warn the user
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "3h", "", "", "2025-01-11", "", "weekend work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have warned about unscheduled day")
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.January, entries[0].Start.Month())
	assert.Equal(t, 11, entries[0].Start.Day())
	assert.Equal(t, 9, entries[0].Start.Hour())
}

func TestLogAddDurationModeNoSlotAvailable(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// Fill up Monday (2025-06-16, 9am-5pm = 8h) with an existing 8h entry
	e := entry.Entry{
		ID:      "00f0001",
		Start:   time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes: 480,
		Message: "full day",
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

	// Try to log 1h on same day — no slot available, falls back to 9:00
	// and schedule warning fires about exceeding budget
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "1h", "", "", "2025-06-16", "", "extra work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have warned about budget overrun")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestLogAddFromToModeWithDate(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout, err := execLogAdd(homeDir, repoDir, "", "", "9am", "12pm", "2025-01-10", "", "past morning")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.January, entries[0].Start.Month())
	assert.Equal(t, 10, entries[0].Start.Day())
	assert.Equal(t, 9, entries[0].Start.Hour())
}

func TestLogAddInvalidDate(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	_, err := execLogAdd(homeDir, repoDir, "", "1h", "", "", "not-a-date", "", "work")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --date format")
}

func TestLogAddInteractiveModeWithDate(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout := new(bytes.Buffer)
	cmd := logAddCmd
	cmd.SetOut(stdout)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Select:  func(_ string, _ []string) (int, error) { return 0, nil },
		Prompt: func(prompt string) (string, error) {
			switch prompt {
			case "Date (YYYY-MM-DD, default: today)":
				return "2025-03-01", nil
			case "Duration (e.g. 30m, 3h, 3h30m)":
				return "2h", nil
			case "Message":
				return "interactive past work", nil
			}
			return "", nil
		},
	}

	err := runLogAdd(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.March, entries[0].Start.Month())
	assert.Equal(t, 1, entries[0].Start.Day())
}

func TestLogAddRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	logGroup := findSubcommand(root, "log")
	require.NotNil(t, logGroup, "log group command should exist")

	names := make([]string, len(logGroup.Commands()))
	for i, cmd := range logGroup.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "edit")
	assert.Contains(t, names, "remove")
}

func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func TestLogAddExceeds24Hours(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	_, err := execLogAdd(homeDir, repoDir, "", "25h", "", "", "", "", "too much")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot log more than 24h in a single entry")
}

func TestLogAddScheduleOverrunWarning(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// fixedNow is 2025-06-15 (Sunday), use 2025-06-16 (Monday) for schedule to apply
	// Default schedule: Mon-Fri 9am-5pm = 8h
	// Pre-log 4h so that logging 6h triggers the overrun warning
	e := entry.Entry{
		ID:      "00e0001",
		Start:   time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes: 240,
		Message: "earlier work",
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "6h", "", "", "2025-06-16", "", "overrun work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestLogAddScheduleOverrunDeclined(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// Pre-log 4h on Monday
	e := entry.Entry{
		ID:      "00e0001",
		Start:   time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes: 240,
		Message: "earlier work",
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			return false, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "6h", "", "", "2025-06-16", "", "overrun work", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Warning:")
	assert.NotContains(t, stdout, "logged 6h") // entry was not written

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1) // only the pre-logged entry
}

func TestLogAddScheduleOverrunNoSchedule(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	// 2025-06-15 is a Sunday — no schedule → 0h scheduled
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "1h", "", "", "2025-06-15", "", "weekend work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation on unscheduled day")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "no scheduled working hours")
	assert.Contains(t, stdout, "logged")
}

func TestLogAddScheduleOverrunWithinBudget(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// Monday with 8h schedule, log only 2h — no warning
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "2h", "", "", "2025-06-16", "", "normal work", pk)

	require.NoError(t, err)
	assert.False(t, confirmed, "should not have prompted for confirmation within budget")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogAddOutsideScheduleWindows(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// Monday 7pm-9pm, schedule is 9am-5pm → fully outside
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "7pm", "9pm", "2025-06-16", "", "evening work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "outside your scheduled hours")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogAddOutsideScheduleWindowsDeclined(t *testing.T) {
	homeDir, repoDir, _ := setupLogAddTest(t)

	// Monday 7pm-9pm, schedule is 9am-5pm → fully outside, declined
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			return false, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "7pm", "9pm", "2025-06-16", "", "evening work", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Warning:")
	assert.NotContains(t, stdout, "logged")
}

func TestLogAddPartiallyOutsideScheduleWindows(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// Monday 4pm-7pm, schedule is 9am-5pm → partially outside
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "4pm", "7pm", "2025-06-16", "", "late work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "partially falls outside")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogAddWithinScheduleWindows(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	// Monday 10am-12pm, schedule is 9am-5pm → fully within
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogAddWithPrompts(homeDir, repoDir, "", "", "10am", "12pm", "2025-06-16", "", "morning work", pk)

	require.NoError(t, err)
	assert.False(t, confirmed, "should not have prompted for confirmation within schedule")
	assert.NotContains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogAddWithTask(t *testing.T) {
	homeDir, repoDir, proj := setupLogAddTest(t)

	stdout, err := execLogAdd(homeDir, repoDir, "", "2h", "", "", "", "research", "read documentation")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "2h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "research", entries[0].Task)
	assert.Equal(t, "read documentation", entries[0].Message)
}
