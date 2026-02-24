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

func setupGenerateTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "Generate Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, repoDir, proj
}

func execGenerate(
	homeDir, repoDir, projectFlag, dateFlag, yearFlag string,
	todayFlag, weekFlag, monthFlag bool,
	pk PromptKit,
) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := generateCmd
	cmd.SetOut(stdout)

	err := runGenerate(cmd, homeDir, repoDir, projectFlag, dateFlag, yearFlag, todayFlag, weekFlag, monthFlag, pk, fixedNow)
	return stdout.String(), err
}

func TestGenerateNoProject(t *testing.T) {
	homeDir := t.TempDir()
	pk := PromptKit{Confirm: AlwaysYes()}
	_, err := execGenerate(homeDir, "", "", "", "", true, false, false, pk)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestGenerateNoCheckouts(t *testing.T) {
	homeDir, repoDir, _ := setupGenerateTest(t)
	pk := PromptKit{Confirm: AlwaysYes()}

	stdout, err := execGenerate(homeDir, repoDir, "", "2025-06-15", "", false, false, false, pk)
	require.NoError(t, err)
	assert.Contains(t, stdout, "No checkout time")
}

func TestGenerateToday(t *testing.T) {
	homeDir, repoDir, proj := setupGenerateTest(t)
	pk := PromptKit{Confirm: AlwaysYes()}

	// fixedNow is 2025-06-15 (Sunday) — use a custom nowFn for Monday June 16
	mondayNow := func() time.Time {
		return time.Date(2025, 6, 16, 14, 0, 0, 0, time.UTC)
	}

	// Create a checkout on June 16 (Monday) at 9am
	co := entry.CheckoutEntry{
		ID:        "co1",
		Timestamp: time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature-x",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, co))

	stdout := new(bytes.Buffer)
	cmd := generateCmd
	cmd.SetOut(stdout)
	err := runGenerate(cmd, homeDir, repoDir, "", "", "", true, false, false, pk, mondayNow)
	require.NoError(t, err)
	out := stdout.String()

	assert.Contains(t, out, "feature-x")
	assert.Contains(t, out, "Generated")

	// Verify log entries were created
	logs, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(logs), 1)

	found := false
	for _, l := range logs {
		if l.Task == "feature-x" && l.Source == "generate" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected a generated entry with task=feature-x")

	// Verify generated-day marker was created
	generatedDays, err := entry.ReadAllGeneratedDayEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, generatedDays, 1)
	assert.Equal(t, "2025-06-16", generatedDays[0].Date)
}

func TestGenerateSpecificDate(t *testing.T) {
	homeDir, repoDir, proj := setupGenerateTest(t)
	pk := PromptKit{Confirm: AlwaysYes()}

	// Checkout before the target date
	co := entry.CheckoutEntry{
		ID:        "co1",
		Timestamp: time.Date(2025, 6, 1, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature-y",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, co))

	// fixedNow is 2025-06-15 14:00 UTC, generate for June 2 (a Monday, should have schedule)
	stdout, err := execGenerate(homeDir, repoDir, "", "2025-06-02", "", false, false, false, pk)
	require.NoError(t, err)
	assert.Contains(t, stdout, "feature-y")
	assert.Contains(t, stdout, "Generated")
}

func TestGenerateOverwriteExisting(t *testing.T) {
	homeDir, repoDir, proj := setupGenerateTest(t)
	pk := PromptKit{Confirm: AlwaysYes()}

	// Use Monday June 16 as "now" (a workday)
	mondayNow := func() time.Time {
		return time.Date(2025, 6, 16, 14, 0, 0, 0, time.UTC)
	}

	// Create a checkout on the workday
	co := entry.CheckoutEntry{
		ID:        "co1",
		Timestamp: time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature-x",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, co))

	// Generate once
	stdout1 := new(bytes.Buffer)
	cmd1 := generateCmd
	cmd1.SetOut(stdout1)
	err := runGenerate(cmd1, homeDir, repoDir, "", "", "", true, false, false, pk, mondayNow)
	require.NoError(t, err)

	logsBefore, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	countBefore := len(logsBefore)

	// Generate again — should overwrite
	stdout2 := new(bytes.Buffer)
	cmd2 := generateCmd
	cmd2.SetOut(stdout2)
	err = runGenerate(cmd2, homeDir, repoDir, "", "", "", true, false, false, pk, mondayNow)
	require.NoError(t, err)
	out := stdout2.String()
	assert.Contains(t, out, "already been generated")
	assert.Contains(t, out, "Generated")

	logsAfter, err := entry.ReadAllEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	// Should have same count (old deleted, new created)
	assert.Equal(t, countBefore, len(logsAfter))
}

func TestGenerateMutuallyExclusiveFlags(t *testing.T) {
	homeDir, repoDir, _ := setupGenerateTest(t)
	pk := PromptKit{Confirm: AlwaysYes()}

	_, err := execGenerate(homeDir, repoDir, "", "2025-06-15", "", true, false, false, pk)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only one of")
}

func TestGenerateByProjectFlag(t *testing.T) {
	homeDir, _, proj := setupGenerateTest(t)
	pk := PromptKit{Confirm: AlwaysYes()}

	// Use Monday June 16 as "now" (a workday)
	mondayNow := func() time.Time {
		return time.Date(2025, 6, 16, 14, 0, 0, 0, time.UTC)
	}

	co := entry.CheckoutEntry{
		ID:        "co1",
		Timestamp: time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature-z",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, co))

	stdout := new(bytes.Buffer)
	cmd := generateCmd
	cmd.SetOut(stdout)
	err := runGenerate(cmd, homeDir, "", proj.Name, "", "", true, false, false, pk, mondayNow)
	require.NoError(t, err)
	assert.Contains(t, stdout.String(), "Generated")
}

func TestGenerateRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "generate")
}

func TestDateRangeToday(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	from, to := dateRangeToday(now)
	assert.Equal(t, "2025-06-15", from.Format("2006-01-02"))
	assert.Equal(t, "2025-06-15", to.Format("2006-01-02"))
}

func TestDateRangeWeek(t *testing.T) {
	// June 15, 2025 is a Sunday
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	from, to := dateRangeWeek(now)
	assert.Equal(t, "2025-06-09", from.Format("2006-01-02")) // Monday
	assert.Equal(t, "2025-06-15", to.Format("2006-01-02"))   // Sunday
}

func TestDateRangeWeek_Monday(t *testing.T) {
	now := time.Date(2025, 6, 9, 14, 0, 0, 0, time.UTC) // Monday
	from, to := dateRangeWeek(now)
	assert.Equal(t, "2025-06-09", from.Format("2006-01-02"))
	assert.Equal(t, "2025-06-15", to.Format("2006-01-02"))
}

func TestDateRangeMonth(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	from, to, err := dateRangeMonth(now, "")
	require.NoError(t, err)
	assert.Equal(t, "2025-06-01", from.Format("2006-01-02"))
	assert.Equal(t, "2025-06-30", to.Format("2006-01-02"))
}

func TestDateRangeSpecific(t *testing.T) {
	from, to, err := dateRangeSpecific("2025-06-10")
	require.NoError(t, err)
	assert.Equal(t, "2025-06-10", from.Format("2006-01-02"))
	assert.Equal(t, "2025-06-10", to.Format("2006-01-02"))
}

func TestDateRangeSpecific_Invalid(t *testing.T) {
	_, _, err := dateRangeSpecific("not-a-date")
	assert.Error(t, err)
}

func TestBuildDateRange(t *testing.T) {
	from := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 13, 0, 0, 0, 0, time.UTC)
	dates := buildDateRange(from, to)
	assert.Equal(t, []string{"2025-06-10", "2025-06-11", "2025-06-12", "2025-06-13"}, dates)
}
