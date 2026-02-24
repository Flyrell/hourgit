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

func setupEditTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry, e entry.Entry) {
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
		ID:        "edt1234",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   180,
		Message:   "original work",
		Task:      "coding",
		CreatedAt: time.Date(2025, 6, 16, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, e))

	return homeDir, repoDir, p, e
}

func execEdit(homeDir, repoDir, projectFlag, hash string, flags map[string]bool,
	durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag string,
) (string, error) {
	return execEditWithConfirm(homeDir, repoDir, projectFlag, hash, flags,
		durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag, nil)
}

func execEditWithConfirm(homeDir, repoDir, projectFlag, hash string, flags map[string]bool,
	durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag string,
	confirm ConfirmFunc,
) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := editCmd
	cmd.SetOut(stdout)

	if flags == nil {
		flags = map[string]bool{}
	}

	err := runEdit(cmd, homeDir, repoDir, projectFlag, hash,
		durationFlag, fromFlag, toFlag, dateFlag, taskFlag, messageFlag,
		flags, PromptKit{}, confirm, fixedNow)
	return stdout.String(), err
}

func TestEditDurationOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	flags := map[string]bool{"duration": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "2h", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "duration:")
	assert.Contains(t, stdout, "2h")
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, 120, e.Minutes)
	// Start should be preserved
	assert.Equal(t, 9, e.Start.Hour())
	assert.Equal(t, 0, e.Start.Minute())
	// Message and task preserved
	assert.Equal(t, "original work", e.Message)
	assert.Equal(t, "coding", e.Task)
}

func TestEditFromToMode(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "10am", "1pm", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 180, e.Minutes)
}

func TestEditFromOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	// Original: 9:00 - 12:00 (180 min). Change from to 10:00, keep end at 12:00.
	flags := map[string]bool{"from": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "10am", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 120, e.Minutes) // 10:00 - 12:00 = 2h
}

func TestEditToOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	// Original: 9:00 - 12:00 (180 min). Change end to 2pm, keep start at 9:00.
	flags := map[string]bool{"to": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "", "2pm", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, 9, e.Start.Hour())
	assert.Equal(t, 300, e.Minutes) // 9:00 - 14:00 = 5h
}

func TestEditDateOnly(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	flags := map[string]bool{"date": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "", "", "2025-07-01", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
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

func TestEditTask(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	flags := map[string]bool{"task": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "", "", "", "reviews", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "task:")
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, "reviews", e.Task)
}

func TestEditClearTask(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	flags := map[string]bool{"task": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "task:")
	assert.Contains(t, stdout, "(none)")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, "", e.Task)
}

func TestEditMessage(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	flags := map[string]bool{"message": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "", "", "", "", "updated work")

	require.NoError(t, err)
	assert.Contains(t, stdout, "message:")
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, "updated work", e.Message)
}

func TestEditEmptyMessageError(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	flags := map[string]bool{"message": true}
	_, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestEditDurationFromToMutuallyExclusive(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	flags := map[string]bool{"duration": true, "from": true}
	_, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "2h", "9am", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestEditExceeds24Hours(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	flags := map[string]bool{"duration": true}
	_, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "25h", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot log more than 24h")
}

func TestEditNotFound(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	flags := map[string]bool{"duration": true}
	_, err := execEdit(homeDir, repoDir, "", "nonexistent", flags, "2h", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEditNoChanges(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	// Set task to same value
	flags := map[string]bool{"task": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "", "", "", "coding", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "no changes")
	assert.NotContains(t, stdout, "updated entry")
}

func TestEditPreservesIDAndCreatedAt(t *testing.T) {
	homeDir, repoDir, proj, original := setupEditTest(t)

	flags := map[string]bool{"duration": true}
	_, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "5h", "", "", "", "", "")

	require.NoError(t, err)

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, original.ID, e.ID)
	assert.True(t, original.CreatedAt.Equal(e.CreatedAt))
}

func TestEditByProjectFlag(t *testing.T) {
	homeDir, _, proj, _ := setupEditTest(t)

	flags := map[string]bool{"duration": true}
	stdout, err := execEdit(homeDir, "", proj.Name, "edt1234", flags, "1h", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, 60, e.Minutes)
}

func TestEditCrossProjectScan(t *testing.T) {
	homeDir := t.TempDir()

	// Create a project without repo assignment
	p, err := project.CreateProject(homeDir, "Scannable")
	require.NoError(t, err)

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	p = project.FindProjectByID(cfg, p.ID)

	e := entry.Entry{
		ID:        "scn1234",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "scan work",
		CreatedAt: time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, e))

	// No repo, no project flag — should scan and find it
	flags := map[string]bool{"duration": true}
	stdout, err := execEdit(homeDir, "", "", "scn1234", flags, "2h", "", "", "", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	updated, err := entry.ReadEntry(homeDir, p.Slug, "scn1234")
	require.NoError(t, err)
	assert.Equal(t, 120, updated.Minutes)
}

func TestEditInteractiveMode(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	stdout := new(bytes.Buffer)
	cmd := editCmd
	cmd.SetOut(stdout)

	pk := PromptKit{
		PromptWithDefault: func(prompt, defaultValue string) (string, error) {
			switch prompt {
			case "Date (YYYY-MM-DD)":
				return "2025-06-16", nil // same date
			case "From (e.g. 9am, 14:00)":
				return "10:00", nil // change from 9:00 to 10:00
			case "To (e.g. 5pm, 17:00)":
				return "12:00", nil // same end
			case "Task":
				return "coding", nil // same task
			case "Message":
				return "original work", nil // same message
			}
			return defaultValue, nil
		},
	}

	flags := map[string]bool{} // no flags = interactive mode
	err := runEdit(cmd, homeDir, repoDir, "", "edt1234",
		"", "", "", "", "", "",
		flags, pk, nil, fixedNow)

	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 120, e.Minutes) // 10:00 - 12:00
}

func TestEditRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "edit")
}

func TestEditDateAndFrom(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	flags := map[string]bool{"date": true, "from": true}
	stdout, err := execEdit(homeDir, repoDir, "", "edt1234", flags, "", "10am", "", "2025-08-01", "", "")

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")

	e, err := entry.ReadEntry(homeDir, proj.Slug, "edt1234")
	require.NoError(t, err)
	assert.Equal(t, 2025, e.Start.Year())
	assert.Equal(t, time.August, e.Start.Month())
	assert.Equal(t, 1, e.Start.Day())
	assert.Equal(t, 10, e.Start.Hour())
	// Original end was 12:00, new from is 10:00 → 2h
	assert.Equal(t, 120, e.Minutes)
}

func TestEditScheduleWarningOutsideHours(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	// Original entry: 2025-06-16 (Monday) 9:00–12:00, within default 9–17 schedule.
	// Move it to 22:00–01:00 — completely outside schedule.
	confirmed := false
	confirm := func(prompt string) (bool, error) {
		confirmed = true
		return true, nil
	}

	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execEditWithConfirm(homeDir, repoDir, "", "edt1234", flags,
		"", "22:00", "23:00", "", "", "", confirm)

	require.NoError(t, err)
	assert.True(t, confirmed, "should have prompted for confirmation")
	assert.Contains(t, stdout, "Warning:")
	assert.Contains(t, stdout, "updated entry")
}

func TestEditScheduleWarningDeclined(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	// Move to outside schedule, but decline the warning
	confirm := func(prompt string) (bool, error) {
		return false, nil
	}

	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execEditWithConfirm(homeDir, repoDir, "", "edt1234", flags,
		"", "22:00", "23:00", "", "", "", confirm)

	require.NoError(t, err)
	assert.NotContains(t, stdout, "updated entry")
}

func TestEditMessageOnlyNoWarning(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	// Only change message — no schedule warning should trigger
	confirmed := false
	confirm := func(prompt string) (bool, error) {
		confirmed = true
		return true, nil
	}

	flags := map[string]bool{"message": true}
	stdout, err := execEditWithConfirm(homeDir, repoDir, "", "edt1234", flags,
		"", "", "", "", "", "new message", confirm)

	require.NoError(t, err)
	assert.False(t, confirmed, "should NOT prompt when only message changes")
	assert.Contains(t, stdout, "updated entry")
}

func TestEditYesFlagSkipsWarning(t *testing.T) {
	homeDir, repoDir, _, _ := setupEditTest(t)

	// Move to outside schedule with --yes (AlwaysYes)
	flags := map[string]bool{"from": true, "to": true}
	stdout, err := execEditWithConfirm(homeDir, repoDir, "", "edt1234", flags,
		"", "22:00", "23:00", "", "", "", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "updated entry")
}

func TestEditBudgetExcludesCurrentEntry(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	// Original entry: edt1234, 9:00–12:00 (180 min) on Monday.
	// Add another 5h entry to fill up most of the 8h budget.
	e2 := entry.Entry{
		ID:        "edt5678",
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
	stdout, err := execEditWithConfirm(homeDir, repoDir, "", "edt1234", flags,
		"2h", "", "", "", "", "", confirm)

	require.NoError(t, err)
	assert.False(t, confirmed, "should NOT warn — budget is within limit when excluding self")
	assert.Contains(t, stdout, "updated entry")
}

func TestEditCheckoutEntryRejected(t *testing.T) {
	homeDir, repoDir, proj, _ := setupEditTest(t)

	// Write a checkout entry
	ce := entry.CheckoutEntry{
		ID:        "chk9999",
		Type:      "checkout",
		Timestamp: time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, ce))

	flags := map[string]bool{"duration": true}
	_, err := execEdit(homeDir, repoDir, "", "chk9999", flags, "2h", "", "", "", "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be edited")
}
