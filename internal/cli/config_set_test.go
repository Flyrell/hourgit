package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/Flyrell/hour-git/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPrompt returns a PromptFunc that feeds pre-determined responses.
func mockPrompt(responses ...string) PromptFunc {
	i := 0
	return func(_ string) (string, error) {
		if i >= len(responses) {
			return "", fmt.Errorf("no more mock responses")
		}
		resp := responses[i]
		i++
		return resp, nil
	}
}

// mockConfirm returns a ConfirmFunc that returns a pre-determined answer.
func mockConfirm(answer bool) ConfirmFunc {
	return func(_ string) (bool, error) {
		return answer, nil
	}
}

func execConfigSet(homeDir, repoDir, projectFlag string, prompt PromptFunc, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := configSetCmd
	cmd.SetOut(stdout)
	err := runConfigSet(cmd, homeDir, repoDir, projectFlag, prompt, confirm)
	return stdout.String(), err
}

func TestConfigSetQuitImmediately(t *testing.T) {
	homeDir, repoDir, _ := setupConfigTest(t)

	prompt := mockPrompt("q")
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")
}

func TestConfigSetAddRecurringWeekdayAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Guided flow: add → recurring → every weekend → 8am → 12pm → quit
	prompt := mockPrompt(
		"a",
		"1",    // schedule type: recurring
		"2",    // every weekend (Sat-Sun)
		"8am",  // start time
		"12pm", // end time
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(reg, entry.ID)
	assert.Len(t, schedules, 2) // default + new
	assert.Equal(t, "08:00", schedules[1].From)
	assert.Equal(t, "12:00", schedules[1].To)
	assert.Contains(t, schedules[1].RRule, "BYDAY=SA,SU")
}

func TestConfigSetAddSpecificDaysAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Guided flow: add → recurring → specific days → Mon,Wed,Fri → 9am → 5pm → quit
	prompt := mockPrompt(
		"a",
		"1",     // recurring
		"4",     // specific days
		"1,3,5", // Mon, Wed, Fri
		"9am",   // start time
		"5pm",   // end time
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(true))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(reg, entry.ID)
	assert.Len(t, schedules, 2)
	assert.Equal(t, "09:00", schedules[1].From)
	assert.Equal(t, "17:00", schedules[1].To)
	assert.Contains(t, schedules[1].RRule, "BYDAY=MO,WE,FR")
}

func TestConfigSetEditAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Guided flow: edit 1 → recurring → every weekday → 8am → 4pm → quit
	prompt := mockPrompt(
		"e 1",
		"1",   // recurring
		"1",   // every weekday
		"8am", // start time
		"4pm", // end time
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(reg, entry.ID)
	assert.Len(t, schedules, 1)
	assert.Equal(t, "08:00", schedules[0].From)
	assert.Equal(t, "16:00", schedules[0].To)
}

func TestConfigSetDeleteAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	prompt := mockPrompt(
		"d 1",
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(reg, entry.ID)
	assert.Equal(t, schedule.DefaultSchedules(), schedules)

	regEntry := project.FindProjectByID(reg, entry.ID)
	assert.Empty(t, regEntry.Schedules)
}

func TestConfigSetInvalidAction(t *testing.T) {
	homeDir, repoDir, _ := setupConfigTest(t)

	prompt := mockPrompt(
		"x",
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "unknown action")
}

func TestConfigSetInvalidIndex(t *testing.T) {
	homeDir, repoDir, _ := setupConfigTest(t)

	prompt := mockPrompt(
		"e 99",
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "out of range")
}

func TestConfigSetByProjectFlag(t *testing.T) {
	homeDir, _, entry := setupConfigTest(t)

	prompt := mockPrompt("q")
	stdout, err := execConfigSet(homeDir, "", entry.Name, prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")
}

func TestConfigSetNoProject(t *testing.T) {
	homeDir := t.TempDir()

	prompt := mockPrompt("q")
	_, err := execConfigSet(homeDir, "", "", prompt, mockConfirm(false))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestConfigSetRegisteredAsSubcommand(t *testing.T) {
	commands := configCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "set")
}

func TestParseActionIndex(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		count   int
		wantIdx int
		wantErr bool
	}{
		{"valid", "e 1", 3, 0, false},
		{"valid last", "d 3", 3, 2, false},
		{"too high", "e 4", 3, 0, true},
		{"too low", "e 0", 3, 0, true},
		{"not a number", "e abc", 3, 0, true},
		{"no number", "e", 3, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx, err := parseActionIndex(tt.action, tt.count)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantIdx, idx)
			}
		})
	}
}

func TestConfigSetAddOverlap(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Add recurring Monday schedule (overlaps with default weekday) — answer "y" to override
	prompt := mockPrompt(
		"a",
		"1",     // recurring
		"4",     // specific days
		"1",     // Monday
		"8am",   // start time
		"4pm",   // end time
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(true))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(reg, entry.ID)
	require.Len(t, schedules, 2)
	assert.True(t, schedules[1].Override, "new entry should have override set")
}

func TestConfigSetAddNoOverlap(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	confirmCalled := false
	trackingConfirm := func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	// Add Saturday schedule (no overlap with weekday default)
	prompt := mockPrompt(
		"a",
		"1",   // recurring
		"4",   // specific days
		"6",   // Saturday
		"9am", // start time
		"1pm", // end time
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, trackingConfirm)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")
	assert.False(t, confirmCalled, "override prompt should not be shown for non-overlapping entry")

	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(reg, entry.ID)
	require.Len(t, schedules, 2)
	assert.False(t, schedules[1].Override)
}

func TestConfigSetAddOverlapDeclined(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Add overlapping entry, decline override
	prompt := mockPrompt(
		"a",
		"1",   // recurring
		"4",   // specific days
		"1",   // Monday
		"8am", // start time
		"4pm", // end time
		"q",
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", prompt, mockConfirm(false))

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(reg, entry.ID)
	require.Len(t, schedules, 2)
	assert.False(t, schedules[1].Override, "entry should not have override when declined")
}

func TestBuildScheduleEntryRecurringWeekday(t *testing.T) {
	w := new(bytes.Buffer)
	prompt := mockPrompt(
		"1",   // recurring
		"1",   // every weekday
		"9am", // start time
		"5pm", // end time
	)

	entry, err := buildScheduleEntry(prompt, w)

	require.NoError(t, err)
	assert.Equal(t, "09:00", entry.From)
	assert.Equal(t, "17:00", entry.To)
	assert.Contains(t, entry.RRule, "BYDAY=MO,TU,WE,TH,FR")
}

func TestBuildScheduleEntryRecurringSpecificDays(t *testing.T) {
	w := new(bytes.Buffer)
	prompt := mockPrompt(
		"1",     // recurring
		"4",     // specific days
		"1,3,5", // Mon, Wed, Fri
		"9am",
		"5pm",
	)

	entry, err := buildScheduleEntry(prompt, w)

	require.NoError(t, err)
	assert.Contains(t, entry.RRule, "BYDAY=MO,WE,FR")
}

func TestBuildScheduleEntryOneOff(t *testing.T) {
	w := new(bytes.Buffer)
	prompt := mockPrompt(
		"2",          // one-off date
		"2026-03-15", // date
		"10am",       // start time
		"2pm",        // end time
	)

	entry, err := buildScheduleEntry(prompt, w)

	require.NoError(t, err)
	assert.Equal(t, "10:00", entry.From)
	assert.Equal(t, "14:00", entry.To)
	assert.Contains(t, entry.RRule, "DTSTART")
	assert.Contains(t, entry.RRule, "COUNT=1")
}

func TestBuildScheduleEntryDateRange(t *testing.T) {
	w := new(bytes.Buffer)
	prompt := mockPrompt(
		"3",          // date range
		"2026-03-02", // start date
		"2026-03-06", // end date
		"9am",        // start time
		"5pm",        // end time
	)

	entry, err := buildScheduleEntry(prompt, w)

	require.NoError(t, err)
	assert.Equal(t, "09:00", entry.From)
	assert.Equal(t, "17:00", entry.To)
	assert.Contains(t, entry.RRule, "DTSTART")
	assert.Contains(t, entry.RRule, "UNTIL")
}

func TestBuildScheduleEntryEveryNDays(t *testing.T) {
	w := new(bytes.Buffer)
	prompt := mockPrompt(
		"1",   // recurring
		"5",   // every N days
		"3",   // N=3
		"9am",
		"5pm",
	)

	entry, err := buildScheduleEntry(prompt, w)

	require.NoError(t, err)
	assert.Contains(t, entry.RRule, "FREQ=DAILY")
	assert.Contains(t, entry.RRule, "INTERVAL=3")
}

func TestBuildScheduleEntryTimeOrderError(t *testing.T) {
	w := new(bytes.Buffer)
	// First attempt: end before start; second attempt: valid
	prompt := mockPrompt(
		"1",   // recurring
		"1",   // every weekday
		"5pm", // start time (wrong order)
		"9am", // end time (wrong order)
		"9am", // start time (retry)
		"5pm", // end time (retry)
	)

	entry, err := buildScheduleEntry(prompt, w)

	require.NoError(t, err)
	assert.Equal(t, "09:00", entry.From)
	assert.Equal(t, "17:00", entry.To)
	assert.Contains(t, w.String(), "end time must be after start time")
}

func TestPromptDays(t *testing.T) {
	t.Run("valid selection", func(t *testing.T) {
		w := new(bytes.Buffer)
		prompt := mockPrompt("1,3,5")
		days, err := promptDays(prompt, w)

		require.NoError(t, err)
		require.Len(t, days, 3)
	})

	t.Run("invalid then valid", func(t *testing.T) {
		w := new(bytes.Buffer)
		prompt := mockPrompt("abc", "1,2")
		days, err := promptDays(prompt, w)

		require.NoError(t, err)
		require.Len(t, days, 2)
		assert.Contains(t, w.String(), "please enter day numbers")
	})

	t.Run("out of range then valid", func(t *testing.T) {
		w := new(bytes.Buffer)
		prompt := mockPrompt("0,8", "7")
		days, err := promptDays(prompt, w)

		require.NoError(t, err)
		require.Len(t, days, 1)
		assert.Contains(t, w.String(), "please enter day numbers")
	})
}
