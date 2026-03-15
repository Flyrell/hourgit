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

func setupLogEditTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry, e entry.Entry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	p, err := project.CreateProject(homeDir, "Edit Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, p))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	p = project.FindProjectByID(cfg, p.ID)

	e = entry.Entry{
		ID:        "ed01234",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   180,
		Message:   "original work",
		Task:      "coding",
		CreatedAt: time.Date(2025, 6, 16, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, e))

	return homeDir, repoDir, p, e
}

func execLogEdit(homeDir, repoDir, projectFlag, hash string, flags map[string]bool,
	durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag string,
) (string, error) {
	return execLogEditWithConfirm(homeDir, repoDir, projectFlag, hash, flags,
		durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag, nil)
}

func execLogEditWithConfirm(homeDir, repoDir, projectFlag, hash string, flags map[string]bool,
	durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag string,
	confirm ConfirmFunc,
) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := logEditCmd
	cmd.SetOut(stdout)

	if flags == nil {
		flags = map[string]bool{}
	}

	err := runLogEdit(cmd, homeDir, repoDir, projectFlag, hash,
		durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag,
		flags, PromptKit{}, confirm, fixedNow)
	return stdout.String(), err
}

func TestLogEditDurationOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"duration": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "2h", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "duration:")
	assert.Contains(t, stdout, "2h")
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 120, e.Minutes)
	// Start should be preserved
	assert.Equal(t, 9, e.Start.Hour())
	assert.Equal(t, 0, e.Start.Minute())
	// Message and task preserved
	assert.Equal(t, "original work", e.Message)
	assert.Equal(t, "coding", e.Task)
}

func TestLogEditFromToMode(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "10am", "1pm", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 180, e.Minutes)
}

func TestLogEditFromOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	// Original: 9:00 - 12:00 (180 min). Change from to 10:00, keep duration (3h) → to shifts to 13:00.
	flags := map[string]bool{"from": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "10am", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 180, e.Minutes) // duration preserved: 3h
}

func TestLogEditToOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	// Original: 9:00 - 12:00 (180 min). Change end to 2pm, keep start at 9:00.
	flags := map[string]bool{"to": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "2pm", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 9, e.Start.Hour())
	assert.Equal(t, 300, e.Minutes) // 9:00 - 14:00 = 5h
}

func TestLogEditDateOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"date": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "", "2025-07-01", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 2025, e.Start.Year())
	assert.Equal(t, time.July, e.Start.Month())
	assert.Equal(t, 1, e.Start.Day())
	// Time-of-day preserved
	assert.Equal(t, 9, e.Start.Hour())
	assert.Equal(t, 0, e.Start.Minute())
	// Minutes preserved
	assert.Equal(t, 180, e.Minutes)
}

func TestLogEditTask(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"task": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "", "", "reviews", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "task:")
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, "reviews", e.Task)
}

func TestLogEditClearTask(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"task": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "task:")
	assert.Contains(t, stdout, "(none)")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, "", e.Task)
}

func TestLogEditMessage(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"message": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "", "", "", "updated work")

	require.NoError(t, err)
	assert.Contains(t, stdout, "message:")
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, "updated work", e.Message)
}

func TestLogEditEmptyMessageError(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	flags := map[string]bool{"message": true}
	_, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestLogEditDurationAndFrom(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	// Original: 9:00-12:00 (180 min). --duration 2h --from 10am → 10:00-12:00
	flags := map[string]bool{"duration": true, "from": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "2h", "10am", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 120, e.Minutes)
}

func TestLogEditDurationAndTo(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	// Original: 9:00-12:00 (180 min). --duration 2h --to 2pm → from = 12:00, 2h
	flags := map[string]bool{"duration": true, "to": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "2h", "", "2pm", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 12, e.Start.Hour())
	assert.Equal(t, 120, e.Minutes)
}

func TestLogEditAllThreeConsistent(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	// --from 10am --to 12pm --duration 2h → consistent, should succeed
	flags := map[string]bool{"duration": true, "from": true, "to": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "2h", "10am", "12pm", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 120, e.Minutes)
}

func TestLogEditAllThreeInconsistent(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	// --from 10am --to 12pm --duration 3h → inconsistent (2h ≠ 3h)
	flags := map[string]bool{"duration": true, "from": true, "to": true}
	_, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "3h", "10am", "12pm", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func TestLogEditToBeforeStart(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	// Original: 9:00-12:00. Set --to to 8am → error (before 9:00 start)
	flags := map[string]bool{"to": true}
	_, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "8am", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be after")
}

func TestLogEditInvalidFromFlag(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	flags := map[string]bool{"from": true}
	_, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "invalid", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --from time")
}

func TestLogEditInvalidToFlag(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	flags := map[string]bool{"to": true}
	_, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "invalid", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --to time")
}

func TestLogEditExceeds24Hours(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	flags := map[string]bool{"duration": true}
	_, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "25h", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot log more than 24h")
}

func TestLogEditNotFound(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	flags := map[string]bool{"duration": true}
	_, err := execLogEdit(homeDir, repoDir, "", "nonexistent", flags, "2h", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLogEditNoChanges(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	// Set task to same value
	flags := map[string]bool{"task": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "", "", "", "coding", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "no changes")
	assert.NotContains(t, stdout, "updated entry")
}

func TestLogEditPreservesIDAndCreatedAt(t *testing.T) {
	homeDir, repoDir, proj, original := setupLogEditTest(t)

	flags := map[string]bool{"duration": true}
	_, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "5h", "", "", "", "", "")

	require.NoError(t, err)

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, original.ID, e.ID)
	assert.True(t, original.CreatedAt.Equal(e.CreatedAt))
}

func TestLogEditByProjectFlag(t *testing.T) {
	homeDir, _, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"duration": true}
	stdout, err := execLogEdit(homeDir, "", proj.Name, "ed01234", flags, "1h", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 60, e.Minutes)
}

func TestLogEditCrossProjectScan(t *testing.T) {
	homeDir := t.TempDir()

	// Create a project without repo assignment
	p, err := project.CreateProject(homeDir, "Scannable")
	require.NoError(t, err)

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	p = project.FindProjectByID(cfg, p.ID)

	e := entry.Entry{
		ID:        "5c01234",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "scan work",
		CreatedAt: time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, e))

	// No repo, no project flag — should scan and find it
	flags := map[string]bool{"duration": true}
	stdout, err := execLogEdit(homeDir, "", "", "5c01234", flags, "2h", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	updated, err := entry.ReadEntry(homeDir, p.Slug, "5c01234")
	require.NoError(t, err)
	assert.Equal(t, 120, updated.Minutes)
}

func TestLogEditInteractiveMode(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	stdout := new(bytes.Buffer)
	cmd := logEditCmd
	cmd.SetOut(stdout)

	pk := PromptKit{
		PromptWithDefault: func(prompt, defaultValue string) (string, error) {
			switch prompt {
			case "Date (YYYY-MM-DD)":
				return "2025-06-16", nil // same date
			case "From (e.g. 9am, 14:00)":
				return "10:00", nil // change from 9:00 to 10:00
			case "Duration (e.g. 30m, 3h, 3h30m)":
				return "2h", nil // 2h duration → to = 12:00
			case "Task":
				return "coding", nil // same task
			case "Message":
				return "original work", nil // same message
			}
			return defaultValue, nil
		},
	}

	flags := map[string]bool{} // no flags = interactive mode
	err := runLogEdit(cmd, homeDir, repoDir, "", "ed01234",
		"", "", "", "", "", "",
		flags, pk, nil, fixedNow)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 120, e.Minutes) // from 10:00, duration 2h
}

func TestLogEditDateAndFrom(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	flags := map[string]bool{"date": true, "from": true}
	stdout, err := execLogEdit(homeDir, repoDir, "", "ed01234", flags, "", "10am", "", "2025-08-01", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "ed01234")
	require.NoError(t, err)
	assert.Equal(t, 2025, e.Start.Year())
	assert.Equal(t, time.August, e.Start.Month())
	assert.Equal(t, 1, e.Start.Day())
	assert.Equal(t, 10, e.Start.Hour())
	// --from keeps duration (3h), to shifts
	assert.Equal(t, 180, e.Minutes)
}

func TestLogEditScheduleWarningOutsideHours(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	// Original entry: 2025-06-16 (Monday) 9:00–12:00, within default 9–17 schedule.
	// Move it to 22:00–01:00 — completely outside schedule.
	confirmed := false
	confirm := func(prompt string) (bool, error) {
		confirmed = true
		return true, nil
	}

	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execLogEditWithConfirm(homeDir, repoDir, "", "ed01234", flags,
		"", "22:00", "23:00", "", "", "", confirm)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "updated entry")
}

func TestLogEditScheduleWarningDeclined(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	// Move to outside schedule, but decline the warning
	confirm := func(prompt string) (bool, error) {
		return false, nil
	}

	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execLogEditWithConfirm(homeDir, repoDir, "", "ed01234", flags,
		"", "22:00", "23:00", "", "", "", confirm)

	require.NoError(t, err)
	assert.NotContains(t, stdout, "updated entry")
}

func TestLogEditMessageOnlyNoWarning(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	// Only change message — no schedule warning should trigger
	confirmed := false
	confirm := func(prompt string) (bool, error) {
		confirmed = true
		return true, nil
	}

	flags := map[string]bool{"message": true}
	stdout, err := execLogEditWithConfirm(homeDir, repoDir, "", "ed01234", flags,
		"", "", "", "", "", "new message", confirm)

	require.NoError(t, err)
	assert.False(t, confirmed, "should NOT prompt when only message changes")
	assert.Contains(t, stdout, "updated entry")
}

func TestLogEditYesFlagSkipsWarning(t *testing.T) {
	homeDir, repoDir, _, _ := setupLogEditTest(t)

	// Move to outside schedule with --yes (AlwaysYes)
	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execLogEditWithConfirm(homeDir, repoDir, "", "ed01234", flags,
		"", "22:00", "23:00", "", "", "", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")
}

func TestLogEditBudgetExcludesCurrentEntry(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	// Original entry: edt1234, 9:00–12:00 (180 min) on Monday.
	// Add another 5h entry to fill up most of the 8h budget.
	e2 := entry.Entry{
		ID:        "ed05678",
		Start:     time.Date(2025, 6, 16, 12, 0, 0, 0, time.UTC),
		Minutes:   300,
		Message:   "other work",
		CreatedAt: time.Date(2025, 6, 16, 17, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e2))

	// Edit edt1234 from 3h to 2h — total would be 5h+2h=7h < 8h budget.
	// Without excludeID, it would count 3h+5h+2h=10h > 8h and warn.
	confirmed := false
	confirm := func(prompt string) (bool, error) {
		confirmed = true
		return true, nil
	}

	flags := map[string]bool{"duration": true}
	stdout, err := execLogEditWithConfirm(homeDir, repoDir, "", "ed01234", flags,
		"2h", "", "", "", "", "", confirm)

	require.NoError(t, err)
	assert.False(t, confirmed, "should NOT warn — budget is within limit when excluding self")
	assert.Contains(t, stdout, "updated entry")
}

func TestLogEditCheckoutEntryRejected(t *testing.T) {
	homeDir, repoDir, proj, _ := setupLogEditTest(t)

	// Write a checkout entry
	ce := entry.CheckoutEntry{
		ID:        "c009999",
		Type:      "checkout",
		Timestamp: time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, ce))

	flags := map[string]bool{"duration": true}
	_, err := execLogEdit(homeDir, repoDir, "", "c009999", flags, "2h", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be edited")
}
