package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupStatusTest(t *testing.T) (homeDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "Status Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, proj
}

func execStatus(
	homeDir, repoDir, projectFlag string,
	gitBranch func() (string, error),
	now func() time.Time,
) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := statusCmd
	cmd.SetOut(stdout)

	err := runStatus(cmd, homeDir, repoDir, projectFlag, gitBranch, now)
	return stdout.String(), err
}

func mockGitBranch(name string) func() (string, error) {
	return func() (string, error) { return name, nil }
}

func mockGitBranchErr() func() (string, error) {
	return func() (string, error) { return "", errors.New("not a git repo") }
}

func mockNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

// weekdaySchedule returns a schedule entry for weekdays with the given time range.
func weekdaySchedule(fromH, fromM, toH, toM int) []schedule.ScheduleEntry {
	return []schedule.ScheduleEntry{
		{
			Ranges: []schedule.TimeRange{
				{From: fmtTime(fromH, fromM), To: fmtTime(toH, toM)},
			},
			RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
		},
	}
}

func fmtTime(h, m int) string {
	return fmt.Sprintf("%02d:%02d", h, m)
}

func TestStatusProjectNotFound(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execStatus(homeDir, "", "nonexistent", mockGitBranch("main"), mockNow(time.Now()))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project 'nonexistent' not found")
}

func TestStatusNoProjectFromRepo(t *testing.T) {
	homeDir := t.TempDir()
	repoDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	_, err := execStatus(homeDir, repoDir, "", mockGitBranch("main"), mockNow(time.Now()))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestStatusBasicOutput(t *testing.T) {
	homeDir, proj := setupStatusTest(t)

	// Set a weekday schedule
	require.NoError(t, project.SetSchedules(homeDir, proj.ID, weekdaySchedule(9, 0, 17, 0)))

	// Use a Wednesday at 10 AM
	now := time.Date(2025, 6, 11, 10, 0, 0, 0, time.UTC)

	stdout, err := execStatus(homeDir, "", proj.Name, mockGitBranch("feature/auth"), mockNow(now))

	require.NoError(t, err)
	assert.Contains(t, stdout, "Status Test")
	assert.Contains(t, stdout, "feature/auth")
	assert.Contains(t, stdout, "logged")
	assert.Contains(t, stdout, "remaining")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
	assert.Contains(t, stdout, "active")
}

func TestStatusWithCheckoutEntry(t *testing.T) {
	homeDir, proj := setupStatusTest(t)

	require.NoError(t, project.SetSchedules(homeDir, proj.ID, weekdaySchedule(9, 0, 17, 0)))

	now := time.Date(2025, 6, 11, 12, 15, 0, 0, time.UTC)

	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, entry.CheckoutEntry{
		ID:        "abc1234",
		Timestamp: time.Date(2025, 6, 11, 10, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature/auth",
	}))

	stdout, err := execStatus(homeDir, "", proj.Name, mockGitBranch("feature/auth"), mockNow(now))

	require.NoError(t, err)
	assert.Contains(t, stdout, "Checked out:")
	assert.Contains(t, stdout, "2h 15m ago")
}

func TestStatusDayOff(t *testing.T) {
	homeDir, proj := setupStatusTest(t)

	require.NoError(t, project.SetSchedules(homeDir, proj.ID, weekdaySchedule(9, 0, 17, 0)))

	// Saturday
	now := time.Date(2025, 6, 14, 10, 0, 0, 0, time.UTC)

	stdout, err := execStatus(homeDir, "", proj.Name, mockGitBranch("main"), mockNow(now))

	require.NoError(t, err)
	assert.Contains(t, stdout, "not a working day")
	assert.NotContains(t, stdout, "Schedule:")
	assert.NotContains(t, stdout, "Tracking:")
}

func TestStatusTrackingInactive(t *testing.T) {
	homeDir, proj := setupStatusTest(t)

	require.NoError(t, project.SetSchedules(homeDir, proj.ID, weekdaySchedule(9, 0, 17, 0)))

	// Wednesday at 6 PM (after hours)
	now := time.Date(2025, 6, 11, 18, 0, 0, 0, time.UTC)

	stdout, err := execStatus(homeDir, "", proj.Name, mockGitBranch("main"), mockNow(now))

	require.NoError(t, err)
	assert.Contains(t, stdout, "inactive")
}

func TestStatusWithLoggedTime(t *testing.T) {
	homeDir, proj := setupStatusTest(t)

	require.NoError(t, project.SetSchedules(homeDir, proj.ID, weekdaySchedule(9, 0, 17, 0)))

	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.UTC)

	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
		ID:        "a0b1c34",
		Start:     time.Date(2025, 6, 11, 9, 0, 0, 0, time.UTC),
		Minutes:   150,
		Message:   "morning work",
		CreatedAt: time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC),
	}))

	stdout, err := execStatus(homeDir, "", proj.Name, mockGitBranch("main"), mockNow(now))

	require.NoError(t, err)
	assert.Contains(t, stdout, "2h 30m logged")
	assert.Contains(t, stdout, "remaining")
}

func TestStatusGitBranchError(t *testing.T) {
	homeDir, proj := setupStatusTest(t)

	require.NoError(t, project.SetSchedules(homeDir, proj.ID, weekdaySchedule(9, 0, 17, 0)))

	// Saturday so we get "not a working day" and skip schedule logic
	now := time.Date(2025, 6, 14, 10, 0, 0, 0, time.UTC)

	stdout, err := execStatus(homeDir, "", proj.Name, mockGitBranchErr(), mockNow(now))

	require.NoError(t, err)
	assert.Contains(t, stdout, "Status Test")
	assert.NotContains(t, stdout, "Branch:")
}

func TestStatusMultiWindowSchedule(t *testing.T) {
	homeDir, proj := setupStatusTest(t)

	schedules := []schedule.ScheduleEntry{
		{
			Ranges: []schedule.TimeRange{
				{From: "09:00", To: "12:00"},
				{From: "13:00", To: "17:00"},
			},
			RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
		},
	}
	require.NoError(t, project.SetSchedules(homeDir, proj.ID, schedules))

	// Wednesday at 10 AM
	now := time.Date(2025, 6, 11, 10, 0, 0, 0, time.UTC)

	stdout, err := execStatus(homeDir, "", proj.Name, mockGitBranch("main"), mockNow(now))

	require.NoError(t, err)
	assert.Contains(t, stdout, "9:00 AM - 12:00 PM")
	assert.Contains(t, stdout, "1:00 PM - 5:00 PM")
}

func TestStatusRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "status")
}
