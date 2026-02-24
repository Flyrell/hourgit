package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLogTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry) {
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

func execLog(homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message string) (string, error) {
	return execLogWithPrompts(homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message, PromptKit{
		Confirm: AlwaysYes(),
	})
}

func execLogWithPrompts(homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message string, pk PromptKit) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := logCmd
	cmd.SetOut(stdout)

	err := runLog(cmd, homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message, pk, fixedNow)
	return stdout.String(), err
}

func TestLogDurationMode(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout, err := execLog(homeDir, repoDir, "", "3h", "", "", "", "", "did some work")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "3h")
	assert.Contains(t, stdout, "Log Test")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, "did some work", entries[0].Message)
}

func TestLogFromToMode(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout, err := execLog(homeDir, repoDir, "", "", "9am", "12pm", "", "", "morning work")

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

func TestLogByProjectFlag(t *testing.T) {
	homeDir, _, proj := setupLogTest(t)

	stdout, err := execLog(homeDir, "", proj.Name, "1h", "", "", "", "", "flagged project")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Log Test")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogDurationAndFromToMutuallyExclusive(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	_, err := execLog(homeDir, repoDir, "", "3h", "9am", "12pm", "", "", "conflict")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestLogFromWithoutTo(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

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

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "", "9am", "", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, "prompted to", entries[0].Message)
}

func TestLogToWithoutFrom(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

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

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "", "", "12pm", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	assert.Equal(t, "prompted from", entries[0].Message)
}

func TestLogFromAfterTo(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	_, err := execLog(homeDir, repoDir, "", "", "5pm", "9am", "", "", "backwards")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be before")
}

func TestLogInvalidDuration(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	_, err := execLog(homeDir, repoDir, "", "abc", "", "", "", "", "bad duration")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid duration")
}

func TestLogDurationNoMessage(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			if prompt == "Message" {
				return "prompted msg", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "3h", "", "", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "prompted msg", entries[0].Message)
}

func TestLogFromToNoMessage(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			if prompt == "Message" {
				return "prompted msg", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "", "9am", "5pm", "", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "8h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "prompted msg", entries[0].Message)
}

func TestLogMessageOnly(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

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

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "", "", "", "", "", "pre-filled msg", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "2h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "pre-filled msg", entries[0].Message)
}

func TestLogDateOnly(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

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

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "", "", "", "2025-01-10", "", "", pk)

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

func TestLogDurationWithDateNoMessage(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			if prompt == "Message" {
				return "date+dur msg", nil
			}
			return "", nil
		},
	}

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "3h", "", "", "2025-01-10", "", "", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.January, entries[0].Start.Month())
	assert.Equal(t, "date+dur msg", entries[0].Message)
}

func TestLogEmptyMessagePromptedStillRequired(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	pk := PromptKit{
		Confirm: AlwaysYes(),
		Prompt: func(prompt string) (string, error) {
			return "", nil
		},
	}

	_, err := execLogWithPrompts(homeDir, repoDir, "", "1h", "", "", "", "", "", pk)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestLogNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execLog(homeDir, "", "", "1h", "", "", "", "", "no project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestLogInteractiveModeDuration(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout := new(bytes.Buffer)
	cmd := logCmd
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

	err := runLog(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "logged")
	assert.Contains(t, stdout.String(), "2h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 120, entries[0].Minutes)
}

func TestLogInteractiveModeFromTo(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout := new(bytes.Buffer)
	cmd := logCmd
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

	err := runLog(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	require.NoError(t, err)

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
}

func TestLogInteractiveEmptyMessage(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	stdout := new(bytes.Buffer)
	cmd := logCmd
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

	err := runLog(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestLogDurationModeWithDate(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout, err := execLog(homeDir, repoDir, "", "3h", "", "", "2025-01-10", "", "past work")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "3h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 180, entries[0].Minutes)
	// fixedNow is 2025-06-15 14:00 UTC, so start = 2025-01-10 at 14:00 - 3h = 11:00
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.January, entries[0].Start.Month())
	assert.Equal(t, 10, entries[0].Start.Day())
	assert.Equal(t, 11, entries[0].Start.Hour())
}

func TestLogFromToModeWithDate(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout, err := execLog(homeDir, repoDir, "", "", "9am", "12pm", "2025-01-10", "", "past morning")

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

func TestLogInvalidDate(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	_, err := execLog(homeDir, repoDir, "", "1h", "", "", "not-a-date", "", "work")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --date format")
}

func TestLogInteractiveModeWithDate(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout := new(bytes.Buffer)
	cmd := logCmd
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

	err := runLog(cmd, homeDir, repoDir, "", "", "", "", "", "", "", pk, fixedNow)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, 2025, entries[0].Start.Year())
	assert.Equal(t, time.March, entries[0].Start.Month())
	assert.Equal(t, 1, entries[0].Start.Day())
}

func TestLogRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "log")
}

func TestLogExceeds24Hours(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	_, err := execLog(homeDir, repoDir, "", "25h", "", "", "", "", "too much")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot log more than 24h in a single entry")
}

func TestLogScheduleOverrunWarning(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	// fixedNow is 2025-06-15 (Sunday), use 2025-06-16 (Monday) for schedule to apply
	// Default schedule: Mon-Fri 9am-5pm = 8h
	// Pre-log 4h so that logging 6h triggers the overrun warning
	e := entry.Entry{
		ID:      "pre0001",
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

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "6h", "", "", "2025-06-16", "", "overrun work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestLogScheduleOverrunDeclined(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	// Pre-log 4h on Monday
	e := entry.Entry{
		ID:      "pre0001",
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

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "6h", "", "", "2025-06-16", "", "overrun work", pk)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Warning:")
	assert.NotContains(t, stdout, "logged 6h") // entry was not written

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1) // only the pre-logged entry
}

func TestLogScheduleOverrunNoSchedule(t *testing.T) {
	homeDir, repoDir, _ := setupLogTest(t)

	// 2025-06-15 is a Sunday — no schedule → 0h scheduled
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "1h", "", "", "2025-06-15", "", "weekend work", pk)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation on unscheduled day")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "logged")
}

func TestLogScheduleOverrunWithinBudget(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	// Monday with 8h schedule, log only 2h — no warning
	confirmed := false
	pk := PromptKit{
		Confirm: func(prompt string) (bool, error) {
			confirmed = true
			return true, nil
		},
	}

	stdout, err := execLogWithPrompts(homeDir, repoDir, "", "2h", "", "", "2025-06-16", "", "normal work", pk)

	require.NoError(t, err)
	assert.False(t, confirmed, "should not have prompted for confirmation within budget")
	assert.Contains(t, stdout, "logged")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestLogWithTask(t *testing.T) {
	homeDir, repoDir, proj := setupLogTest(t)

	stdout, err := execLog(homeDir, repoDir, "", "2h", "", "", "", "research", "read documentation")

	require.NoError(t, err)
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "2h")

	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "research", entries[0].Task)
	assert.Equal(t, "read documentation", entries[0].Message)
}
