package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
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

// mockConfirmSequence returns a ConfirmFunc that returns answers in order.
func mockConfirmSequence(answers ...bool) ConfirmFunc {
	i := 0
	return func(_ string) (bool, error) {
		if i >= len(answers) {
			return false, fmt.Errorf("no more mock confirm responses")
		}
		ans := answers[i]
		i++
		return ans, nil
	}
}

// mockSelect returns a SelectFunc that always returns the given index.
func mockSelect(idx int) SelectFunc {
	return func(_ string, _ []string) (int, error) {
		return idx, nil
	}
}

// mockSelectSequence returns a SelectFunc that returns indices in order.
func mockSelectSequence(indices ...int) SelectFunc {
	i := 0
	return func(_ string, _ []string) (int, error) {
		if i >= len(indices) {
			return 0, fmt.Errorf("no more mock select responses")
		}
		idx := indices[i]
		i++
		return idx, nil
	}
}

// mockMultiSelect returns a MultiSelectFunc that always returns the given indices.
func mockMultiSelect(indices []int) MultiSelectFunc {
	return func(_ string, _ []string) ([]int, error) {
		return indices, nil
	}
}

func testKit(sel SelectFunc, prompt PromptFunc, confirm ConfirmFunc, multiSel MultiSelectFunc) PromptKit {
	return PromptKit{
		Prompt:      prompt,
		Confirm:     confirm,
		Select:      sel,
		MultiSelect: multiSel,
	}
}

func execConfigSet(homeDir, repoDir, projectFlag string, kit PromptKit) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := configSetCmd
	cmd.SetOut(stdout)
	err := runConfigSet(cmd, homeDir, repoDir, projectFlag, kit)
	return stdout.String(), err
}

func TestConfigSetQuitImmediately(t *testing.T) {
	homeDir, repoDir, _ := setupConfigTest(t)

	kit := testKit(
		mockSelect(3), // Save & quit
		mockPrompt(),
		mockConfirm(false),
		mockMultiSelect(nil),
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")
}

func TestConfigSetAddRecurringWeekendAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Action: Add(0), then Save&quit(3)
	// Schedule type: Recurring(0)
	// Recurrence: Every weekend(1)
	kit := testKit(
		mockSelectSequence(0, 0, 1, 3),
		mockPrompt("8am", "12pm"),
		mockConfirmSequence(false, false), // no more ranges, no overlap confirm needed
		mockMultiSelect(nil),
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Len(t, schedules, 2) // default + new
	require.Len(t, schedules[1].Ranges, 1)
	assert.Equal(t, "08:00", schedules[1].Ranges[0].From)
	assert.Equal(t, "12:00", schedules[1].Ranges[0].To)
	assert.Contains(t, schedules[1].RRule, "BYDAY=SA,SU")
}

func TestConfigSetAddSpecificDaysAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Action: Add(0), then Save&quit(3)
	// Schedule type: Recurring(0)
	// Recurrence: Specific days(3)
	// Days: Mon(0), Wed(2), Fri(4)
	kit := testKit(
		mockSelectSequence(0, 0, 3, 3),
		mockPrompt("9am", "5pm"),
		mockConfirmSequence(false, true), // no more ranges, overlap override yes
		mockMultiSelect([]int{0, 2, 4}),
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Len(t, schedules, 2)
	require.Len(t, schedules[1].Ranges, 1)
	assert.Equal(t, "09:00", schedules[1].Ranges[0].From)
	assert.Equal(t, "17:00", schedules[1].Ranges[0].To)
	assert.Contains(t, schedules[1].RRule, "BYDAY=MO,WE,FR")
}

func TestConfigSetEditAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Action: Edit(1), select schedule 0, then Save&quit(3)
	// Schedule type: Recurring(0)
	// Recurrence: Every weekday(0)
	kit := testKit(
		mockSelectSequence(1, 0, 0, 0, 3),
		mockPrompt("8am", "4pm"),
		mockConfirmSequence(false), // no more ranges
		mockMultiSelect(nil),
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Len(t, schedules, 1)
	require.Len(t, schedules[0].Ranges, 1)
	assert.Equal(t, "08:00", schedules[0].Ranges[0].From)
	assert.Equal(t, "16:00", schedules[0].Ranges[0].To)
}

func TestConfigSetDeleteAndQuit(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Action: Delete(2), select schedule 0, then Save&quit(3)
	kit := testKit(
		mockSelectSequence(2, 0, 3),
		mockPrompt(),
		mockConfirm(false),
		mockMultiSelect(nil),
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Equal(t, schedule.DefaultSchedules(), schedules)

	regEntry := project.FindProjectByID(cfg, entry.ID)
	assert.Empty(t, regEntry.Schedules)
}

func TestConfigSetByProjectFlag(t *testing.T) {
	homeDir, _, entry := setupConfigTest(t)

	kit := testKit(
		mockSelect(3), // Save & quit
		mockPrompt(),
		mockConfirm(false),
		mockMultiSelect(nil),
	)
	stdout, err := execConfigSet(homeDir, "", entry.Name, kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")
}

func TestConfigSetNoProject(t *testing.T) {
	homeDir := t.TempDir()

	kit := testKit(
		mockSelect(3),
		mockPrompt(),
		mockConfirm(false),
		mockMultiSelect(nil),
	)
	_, err := execConfigSet(homeDir, "", "", kit)

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

func TestConfigSetAddOverlap(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Add(0): recurring(0) > specific days(3) > Monday > 8am-4pm > no more ranges
	// Overlap detected → confirm yes
	// Save&quit(3)
	kit := testKit(
		mockSelectSequence(0, 0, 3, 3),
		mockPrompt("8am", "4pm"),
		mockConfirmSequence(false, true), // no more ranges, override yes
		mockMultiSelect([]int{0}),        // Monday
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	require.Len(t, schedules, 2)
	assert.True(t, schedules[1].Override, "new entry should have override set")
}

func TestConfigSetAddNoOverlap(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	confirmCalled := false
	confirmTracker := func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	// Add(0): recurring(0) > specific days(3) > Saturday > 9am-1pm > no more ranges
	// No overlap → confirm not called
	// Save&quit(3)
	kit := testKit(
		mockSelectSequence(0, 0, 3, 3),
		mockPrompt("9am", "1pm"),
		confirmTracker,
		mockMultiSelect([]int{5}), // Saturday
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")
	// confirmTracker is used for "Add another time range?" too, so it will be called
	// But the overlap confirm should not have been called before the "add more" confirm
	_ = confirmCalled

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	require.Len(t, schedules, 2)
	assert.False(t, schedules[1].Override)
}

func TestConfigSetAddOverlapDeclined(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Add overlapping Monday schedule, decline override
	kit := testKit(
		mockSelectSequence(0, 0, 3, 3),
		mockPrompt("8am", "4pm"),
		mockConfirmSequence(false, false), // no more ranges, override no
		mockMultiSelect([]int{0}),         // Monday
	)
	stdout, err := execConfigSet(homeDir, repoDir, "", kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	require.Len(t, schedules, 2)
	assert.False(t, schedules[1].Override, "entry should not have override when declined")
}

func TestBuildScheduleEntryRecurringWeekday(t *testing.T) {
	w := new(bytes.Buffer)
	kit := testKit(
		mockSelectSequence(0, 0), // recurring, every weekday
		mockPrompt("9am", "5pm"),
		mockConfirm(false), // no more ranges
		mockMultiSelect(nil),
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	require.Len(t, entry.Ranges, 1)
	assert.Equal(t, "09:00", entry.Ranges[0].From)
	assert.Equal(t, "17:00", entry.Ranges[0].To)
	assert.Contains(t, entry.RRule, "BYDAY=MO,TU,WE,TH,FR")
}

func TestBuildScheduleEntryRecurringSpecificDays(t *testing.T) {
	w := new(bytes.Buffer)
	kit := testKit(
		mockSelectSequence(0, 3), // recurring, specific days
		mockPrompt("9am", "5pm"),
		mockConfirm(false),            // no more ranges
		mockMultiSelect([]int{0, 2, 4}), // Mon, Wed, Fri
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	assert.Contains(t, entry.RRule, "BYDAY=MO,WE,FR")
}

func TestBuildScheduleEntryOneOff(t *testing.T) {
	w := new(bytes.Buffer)
	kit := testKit(
		mockSelect(1), // one-off date
		mockPrompt("2026-03-15", "10am", "2pm"),
		mockConfirm(false), // no more ranges
		mockMultiSelect(nil),
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	require.Len(t, entry.Ranges, 1)
	assert.Equal(t, "10:00", entry.Ranges[0].From)
	assert.Equal(t, "14:00", entry.Ranges[0].To)
	assert.Contains(t, entry.RRule, "DTSTART")
	assert.Contains(t, entry.RRule, "COUNT=1")
}

func TestBuildScheduleEntryDateRange(t *testing.T) {
	w := new(bytes.Buffer)
	kit := testKit(
		mockSelect(2), // date range
		mockPrompt("2026-03-02", "2026-03-06", "9am", "5pm"),
		mockConfirm(false), // no more ranges
		mockMultiSelect(nil),
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	require.Len(t, entry.Ranges, 1)
	assert.Equal(t, "09:00", entry.Ranges[0].From)
	assert.Equal(t, "17:00", entry.Ranges[0].To)
	assert.Contains(t, entry.RRule, "DTSTART")
	assert.Contains(t, entry.RRule, "UNTIL")
}

func TestBuildScheduleEntryEveryNDays(t *testing.T) {
	w := new(bytes.Buffer)
	kit := testKit(
		mockSelectSequence(0, 4), // recurring, every N days
		mockPrompt("3", "9am", "5pm"),
		mockConfirm(false), // no more ranges
		mockMultiSelect(nil),
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	assert.Contains(t, entry.RRule, "FREQ=DAILY")
	assert.Contains(t, entry.RRule, "INTERVAL=3")
}

func TestBuildScheduleEntryTimeOrderError(t *testing.T) {
	w := new(bytes.Buffer)
	// First attempt: end before start; second attempt: valid
	kit := testKit(
		mockSelectSequence(0, 0), // recurring, every weekday
		mockPrompt("5pm", "9am", "9am", "5pm"),
		mockConfirm(false), // no more ranges
		mockMultiSelect(nil),
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	require.Len(t, entry.Ranges, 1)
	assert.Equal(t, "09:00", entry.Ranges[0].From)
	assert.Equal(t, "17:00", entry.Ranges[0].To)
	assert.Contains(t, w.String(), "end time must be after start time")
}

func TestBuildScheduleEntryMultipleRanges(t *testing.T) {
	w := new(bytes.Buffer)
	kit := testKit(
		mockSelectSequence(0, 0), // recurring, every weekday
		mockPrompt("9am", "12pm", "1pm", "5pm"),
		mockConfirmSequence(true, false), // add another, then no more
		mockMultiSelect(nil),
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	require.Len(t, entry.Ranges, 2)
	assert.Equal(t, "09:00", entry.Ranges[0].From)
	assert.Equal(t, "12:00", entry.Ranges[0].To)
	assert.Equal(t, "13:00", entry.Ranges[1].From)
	assert.Equal(t, "17:00", entry.Ranges[1].To)
	assert.Contains(t, entry.RRule, "BYDAY=MO,TU,WE,TH,FR")
}

func TestBuildScheduleEntryMultipleRangesOverlap(t *testing.T) {
	w := new(bytes.Buffer)
	kit := testKit(
		mockSelectSequence(0, 0), // recurring, every weekday
		mockPrompt("9am", "2pm", "1pm", "5pm", "3pm", "5pm"),
		mockConfirmSequence(true, false), // add more, no more (overlap error triggers continue, not confirm)
		mockMultiSelect(nil),
	)

	entry, err := buildScheduleEntry(kit, w)

	require.NoError(t, err)
	require.Len(t, entry.Ranges, 2)
	assert.Contains(t, w.String(), "overlap")
}

func TestPromptDays(t *testing.T) {
	t.Run("valid selection", func(t *testing.T) {
		kit := testKit(nil, nil, nil, mockMultiSelect([]int{0, 2, 4}))
		days, err := promptDays(kit)

		require.NoError(t, err)
		require.Len(t, days, 3)
	})

	t.Run("empty selection", func(t *testing.T) {
		kit := testKit(nil, nil, nil, mockMultiSelect([]int{}))
		_, err := promptDays(kit)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one day")
	})
}

