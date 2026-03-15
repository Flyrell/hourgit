package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/Flyrell/hourgit/internal/timetrack"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// overlayResult is sent when an overlay completes.
type overlayResult struct {
	action string // "cancel", "edit", "add", "remove", "submit"
	err    error
}

func overlayResultMsg(action string, err error) tea.Cmd {
	return func() tea.Msg {
		return overlayResult{action: action, err: err}
	}
}

var (
	overlayBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2).
			Width(50)
	overlayTitleStyle  = lipgloss.NewStyle().Bold(true)
	overlayActiveStyle = lipgloss.NewStyle().Reverse(true)
	overlayMutedStyle  = lipgloss.NewStyle().Faint(true)
)

// --- Text field helper ---

// textField provides insertChar/deleteChar on a string pointer.
type textField struct{ value *string }

func (f textField) insertChar(ch string) {
	*f.value += ch
}

func (f textField) deleteChar() {
	if len(*f.value) > 0 {
		*f.value = (*f.value)[:len(*f.value)-1]
	}
}

// --- Confirmation overlay helper ---

// confirmOverlay provides shared Update/button-rendering for yes/no confirmation overlays.
type confirmOverlay struct {
	action string // result action name on confirm (e.g. "remove", "submit")
	cursor int    // 0 = yes, 1 = no
}

func (o *confirmOverlay) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return true, overlayResultMsg("cancel", nil)
		case "left", "h", "right", "l", "tab":
			o.cursor = 1 - o.cursor
		case "enter":
			if o.cursor == 0 {
				return true, overlayResultMsg(o.action, nil)
			}
			return true, overlayResultMsg("cancel", nil)
		case "y":
			return true, overlayResultMsg(o.action, nil)
		case "n":
			return true, overlayResultMsg("cancel", nil)
		}
	}
	return false, nil
}

func (o *confirmOverlay) renderButtons() string {
	yes := "  [Yes]"
	no := "  [No]"
	if o.cursor == 0 {
		yes = overlayActiveStyle.Render("> [Yes]")
	}
	if o.cursor == 1 {
		no = overlayActiveStyle.Render("> [No]")
	}
	return yes + "    " + no
}

// --- Entry Selector Overlay ---
// Used when a cell has multiple entries and user needs to pick one.

type entrySelectorOverlay struct {
	entries []timetrack.CellEntry
	cursor  int
	title   string
	action  string // what happens after selection: "edit" or "remove"
}

func newEntrySelectorOverlay(entries []timetrack.CellEntry, title, action string) *entrySelectorOverlay {
	return &entrySelectorOverlay{
		entries: entries,
		title:   title,
		action:  action,
	}
}

func (o *entrySelectorOverlay) Init() tea.Cmd { return nil }

func (o *entrySelectorOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return o, overlayResultMsg("cancel", nil)
		case "up", "k":
			if o.cursor > 0 {
				o.cursor--
			}
		case "down", "j":
			if o.cursor < len(o.entries)-1 {
				o.cursor++
			}
		case "enter":
			// Selection made — parent will handle based on action
			return o, overlayResultMsg("select", nil)
		}
	}
	return o, nil
}

func (o *entrySelectorOverlay) View() string {
	var b strings.Builder
	b.WriteString(overlayTitleStyle.Render(o.title))
	b.WriteString("\n\n")

	for i, e := range o.entries {
		label := fmt.Sprintf("%s  %s", entry.FormatMinutes(e.Minutes), e.Message)
		if !e.Persisted {
			label += " (generated)"
		}
		if i == o.cursor {
			b.WriteString(overlayActiveStyle.Render("> "+label) + "\n")
		} else {
			b.WriteString("  " + label + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(overlayMutedStyle.Render("↑/↓ select  |  enter confirm  |  esc cancel"))

	return overlayBoxStyle.Render(b.String())
}

func (o *entrySelectorOverlay) selectedEntry() timetrack.CellEntry {
	return o.entries[o.cursor]
}

// --- Edit Overlay ---
// Form to modify duration, task, message of a selected entry.

type editField int

const (
	editFieldFrom editField = iota
	editFieldTo
	editFieldDuration
	editFieldTask
	editFieldMessage
	editFieldConfirm
)

type editOverlay struct {
	entry    timetrack.CellEntry
	from     string
	to       string
	duration string
	task     string
	message  string
	field    editField
	err      string
}

func newEditOverlay(ce timetrack.CellEntry) *editOverlay {
	endTime := ce.Start.Add(time.Duration(ce.Minutes) * time.Minute)
	return &editOverlay{
		entry:    ce,
		from:     ce.Start.Format("15:04"),
		to:       endTime.Format("15:04"),
		duration: entry.FormatMinutes(ce.Minutes),
		task:     ce.Task,
		message:  ce.Message,
	}
}

func (o *editOverlay) Init() tea.Cmd { return nil }

func (o *editOverlay) activeTextField() *textField {
	switch o.field {
	case editFieldFrom:
		return &textField{&o.from}
	case editFieldTo:
		return &textField{&o.to}
	case editFieldDuration:
		return &textField{&o.duration}
	case editFieldTask:
		return &textField{&o.task}
	case editFieldMessage:
		return &textField{&o.message}
	}
	return nil
}

// recomputeFromField recalculates to = from + duration when leaving the from field.
func (o *editOverlay) recomputeFromField() {
	fromTOD, err := schedule.ParseTimeOfDay(o.from)
	if err != nil {
		return
	}
	mins, err := entry.ParseDuration(o.duration)
	if err != nil {
		return
	}
	endTime := time.Date(2000, 1, 1, fromTOD.Hour, fromTOD.Minute, 0, 0, time.UTC).
		Add(time.Duration(mins) * time.Minute)
	o.to = endTime.Format("15:04")
}

// recomputeToField recalculates duration = to - from when leaving the to field.
func (o *editOverlay) recomputeToField() {
	fromTOD, err := schedule.ParseTimeOfDay(o.from)
	if err != nil {
		return
	}
	toTOD, err := schedule.ParseTimeOfDay(o.to)
	if err != nil {
		return
	}
	fromMins := fromTOD.Hour*60 + fromTOD.Minute
	toMins := toTOD.Hour*60 + toTOD.Minute
	if toMins > fromMins {
		o.duration = entry.FormatMinutes(toMins - fromMins)
	}
}

// recomputeDurationField recalculates to = from + duration when leaving the duration field.
func (o *editOverlay) recomputeDurationField() {
	o.recomputeFromField() // same logic: to = from + duration
}

func (o *editOverlay) advanceField() {
	prev := o.field
	if o.field < editFieldConfirm {
		o.field++
	}
	o.recomputeOnLeave(prev)
}

func (o *editOverlay) retreatField() {
	prev := o.field
	if o.field > editFieldFrom {
		o.field--
	}
	o.recomputeOnLeave(prev)
}

func (o *editOverlay) recomputeOnLeave(prev editField) {
	switch prev {
	case editFieldFrom:
		o.recomputeFromField()
	case editFieldTo:
		o.recomputeToField()
	case editFieldDuration:
		o.recomputeDurationField()
	}
}

func (o *editOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return o, overlayResultMsg("cancel", nil)
		case "tab", "down":
			o.advanceField()
		case "shift+tab", "up":
			o.retreatField()
		case "enter":
			if o.field == editFieldConfirm {
				fromTOD, err := schedule.ParseTimeOfDay(o.from)
				if err != nil {
					o.err = "Invalid from time (e.g. 9am, 14:00)"
					return o, nil
				}
				toTOD, err := schedule.ParseTimeOfDay(o.to)
				if err != nil {
					o.err = "Invalid to time (e.g. 5pm, 17:00)"
					return o, nil
				}
				fromMins := fromTOD.Hour*60 + fromTOD.Minute
				toMins := toTOD.Hour*60 + toTOD.Minute
				if toMins <= fromMins {
					o.err = "To must be after From"
					return o, nil
				}
				mins := toMins - fromMins
				// Update entry start and minutes
				y, mo, d := o.entry.Start.Date()
				o.entry.Start = time.Date(y, mo, d, fromTOD.Hour, fromTOD.Minute, 0, 0, o.entry.Start.Location())
				o.entry.Minutes = mins
				o.entry.Task = o.task
				o.entry.Message = o.message
				o.err = ""
				return o, overlayResultMsg("edit", nil)
			}
			o.advanceField()
		case "backspace":
			if tf := o.activeTextField(); tf != nil {
				tf.deleteChar()
			}
		default:
			if len(msg.String()) == 1 {
				if tf := o.activeTextField(); tf != nil {
					tf.insertChar(msg.String())
				}
			}
		}
	}
	return o, nil
}

func (o *editOverlay) View() string {
	var b strings.Builder
	b.WriteString(overlayTitleStyle.Render("Edit Entry"))
	b.WriteString("\n\n")

	fields := []struct {
		label string
		value string
		field editField
	}{
		{"From", o.from, editFieldFrom},
		{"To", o.to, editFieldTo},
		{"Duration", o.duration, editFieldDuration},
		{"Task", o.task, editFieldTask},
		{"Message", o.message, editFieldMessage},
	}

	for _, f := range fields {
		prefix := "  "
		if o.field == f.field {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%s: %s", prefix, f.label, f.value)
		if o.field == f.field {
			b.WriteString(overlayActiveStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if o.field == editFieldConfirm {
		b.WriteString(overlayActiveStyle.Render("> [Save]"))
	} else {
		b.WriteString("  [Save]")
	}
	b.WriteString("\n")

	if o.err != "" {
		b.WriteString("\n")
		b.WriteString(Error(o.err))
	}

	b.WriteString("\n")
	b.WriteString(overlayMutedStyle.Render("tab/↑/↓ navigate  |  enter confirm  |  esc cancel"))

	return overlayBoxStyle.Render(b.String())
}

// --- Add Overlay ---
// Form to create a new log entry for a (task, day) cell.

type addField int

const (
	addFieldFrom addField = iota
	addFieldTo
	addFieldDuration
	addFieldTask
	addFieldMessage
	addFieldConfirm
)

type addOverlay struct {
	day      int
	month    time.Month
	year     int
	task     string // pre-filled from selected row
	from     string
	to       string
	duration string
	message  string
	field    addField
	err      string
}

func newAddOverlay(day int, month time.Month, year int, task string) *addOverlay {
	return &addOverlay{
		day:   day,
		month: month,
		year:  year,
		task:  task,
		from:  "9:00",
	}
}

func (o *addOverlay) Init() tea.Cmd { return nil }

func (o *addOverlay) activeTextField() *textField {
	switch o.field {
	case addFieldFrom:
		return &textField{&o.from}
	case addFieldTo:
		return &textField{&o.to}
	case addFieldDuration:
		return &textField{&o.duration}
	case addFieldTask:
		return &textField{&o.task}
	case addFieldMessage:
		return &textField{&o.message}
	}
	return nil
}

// recomputeFromField recalculates to = from + duration when leaving the from field.
func (o *addOverlay) recomputeFromField() {
	fromTOD, err := schedule.ParseTimeOfDay(o.from)
	if err != nil {
		return
	}
	mins, err := entry.ParseDuration(o.duration)
	if err != nil {
		return
	}
	endTime := time.Date(2000, 1, 1, fromTOD.Hour, fromTOD.Minute, 0, 0, time.UTC).
		Add(time.Duration(mins) * time.Minute)
	o.to = endTime.Format("15:04")
}

// recomputeToField recalculates duration = to - from when leaving the to field.
func (o *addOverlay) recomputeToField() {
	fromTOD, err := schedule.ParseTimeOfDay(o.from)
	if err != nil {
		return
	}
	toTOD, err := schedule.ParseTimeOfDay(o.to)
	if err != nil {
		return
	}
	fromMins := fromTOD.Hour*60 + fromTOD.Minute
	toMins := toTOD.Hour*60 + toTOD.Minute
	if toMins > fromMins {
		o.duration = entry.FormatMinutes(toMins - fromMins)
	}
}

// recomputeDurationField recalculates to = from + duration when leaving the duration field.
func (o *addOverlay) recomputeDurationField() {
	o.recomputeFromField() // same logic: to = from + duration
}

func (o *addOverlay) advanceField() {
	prev := o.field
	if o.field < addFieldConfirm {
		o.field++
	}
	o.recomputeOnLeave(prev)
}

func (o *addOverlay) retreatField() {
	prev := o.field
	if o.field > addFieldFrom {
		o.field--
	}
	o.recomputeOnLeave(prev)
}

func (o *addOverlay) recomputeOnLeave(prev addField) {
	switch prev {
	case addFieldFrom:
		o.recomputeFromField()
	case addFieldTo:
		o.recomputeToField()
	case addFieldDuration:
		o.recomputeDurationField()
	}
}

func (o *addOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return o, overlayResultMsg("cancel", nil)
		case "tab", "down":
			o.advanceField()
		case "shift+tab", "up":
			o.retreatField()
		case "enter":
			if o.field == addFieldConfirm {
				fromTOD, err := schedule.ParseTimeOfDay(o.from)
				if err != nil {
					o.err = "Invalid from time (e.g. 9am, 14:00)"
					return o, nil
				}
				if o.duration == "" {
					o.err = "Duration is required"
					return o, nil
				}
				mins, err := entry.ParseDuration(o.duration)
				if err != nil {
					o.err = "Invalid duration (e.g. 2h30m, 90m)"
					return o, nil
				}
				// Validate to if provided
				if o.to != "" {
					toTOD, err := schedule.ParseTimeOfDay(o.to)
					if err != nil {
						o.err = "Invalid to time (e.g. 5pm, 17:00)"
						return o, nil
					}
					toMins := toTOD.Hour*60 + toTOD.Minute
					fromMins := fromTOD.Hour*60 + fromTOD.Minute
					if toMins <= fromMins {
						o.err = "To must be after From"
						return o, nil
					}
				}
				_ = mins
				o.err = ""
				return o, overlayResultMsg("add", nil)
			}
			o.advanceField()
		case "backspace":
			if tf := o.activeTextField(); tf != nil {
				tf.deleteChar()
			}
		default:
			if len(msg.String()) == 1 {
				if tf := o.activeTextField(); tf != nil {
					tf.insertChar(msg.String())
				}
			}
		}
	}
	return o, nil
}

func (o *addOverlay) View() string {
	var b strings.Builder
	b.WriteString(overlayTitleStyle.Render(fmt.Sprintf("Add Entry — %s %d", o.month, o.day)))
	b.WriteString("\n\n")

	fields := []struct {
		label string
		value string
		field addField
	}{
		{"From", o.from, addFieldFrom},
		{"To", o.to, addFieldTo},
		{"Duration", o.duration, addFieldDuration},
		{"Task", o.task, addFieldTask},
		{"Message", o.message, addFieldMessage},
	}

	for _, f := range fields {
		prefix := "  "
		if o.field == f.field {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%s: %s", prefix, f.label, f.value)
		if o.field == f.field {
			b.WriteString(overlayActiveStyle.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if o.field == addFieldConfirm {
		b.WriteString(overlayActiveStyle.Render("> [Add]"))
	} else {
		b.WriteString("  [Add]")
	}
	b.WriteString("\n")

	if o.err != "" {
		b.WriteString("\n")
		b.WriteString(Error(o.err))
	}

	b.WriteString("\n")
	b.WriteString(overlayMutedStyle.Render("tab/↑/↓ navigate  |  enter confirm  |  esc cancel"))

	return overlayBoxStyle.Render(b.String())
}

func (o *addOverlay) buildEntry(now time.Time) (entry.Entry, error) {
	fromTOD, err := schedule.ParseTimeOfDay(o.from)
	if err != nil {
		return entry.Entry{}, fmt.Errorf("invalid from time: %w", err)
	}

	mins, err := entry.ParseDuration(o.duration)
	if err != nil {
		return entry.Entry{}, err
	}

	task := o.task
	msg := o.message
	if msg == "" {
		msg = task
	}

	return entry.Entry{
		ID:        hashutil.GenerateID("add"),
		Start:     time.Date(o.year, o.month, o.day, fromTOD.Hour, fromTOD.Minute, 0, 0, time.UTC),
		Minutes:   mins,
		Message:   msg,
		Task:      task,
		Source:    "manual",
		CreatedAt: now.UTC(),
	}, nil
}

// --- Remove Overlay ---
// Confirmation to remove an entry.

type removeOverlay struct {
	entry   timetrack.CellEntry
	confirm confirmOverlay
}

func newRemoveOverlay(ce timetrack.CellEntry) *removeOverlay {
	return &removeOverlay{
		entry:   ce,
		confirm: confirmOverlay{action: "remove", cursor: 1},
	}
}

func (o *removeOverlay) Init() tea.Cmd { return nil }

func (o *removeOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	handled, cmd := o.confirm.Update(msg)
	if handled {
		return o, cmd
	}
	return o, nil
}

func (o *removeOverlay) View() string {
	var b strings.Builder
	b.WriteString(overlayTitleStyle.Render("Remove Entry"))
	b.WriteString("\n\n")

	label := fmt.Sprintf("%s  %s", entry.FormatMinutes(o.entry.Minutes), o.entry.Message)
	if !o.entry.Persisted {
		label += " (generated)"
	}
	b.WriteString("  " + label)
	b.WriteString("\n\n")
	b.WriteString("  Remove this entry?\n\n")

	b.WriteString(o.confirm.renderButtons())
	b.WriteString("\n\n")
	b.WriteString(overlayMutedStyle.Render("←/→ select  |  enter confirm  |  esc cancel"))

	return overlayBoxStyle.Render(b.String())
}

// --- Submit Overlay ---
// Confirmation to persist all in-memory entries.

type submitOverlay struct {
	inMemoryCount int
	from, to      time.Time
	confirm       confirmOverlay
}

func newSubmitOverlay(inMemoryCount int, from, to time.Time) *submitOverlay {
	return &submitOverlay{
		inMemoryCount: inMemoryCount,
		from:          from,
		to:            to,
		confirm:       confirmOverlay{action: "submit", cursor: 1},
	}
}

func (o *submitOverlay) Init() tea.Cmd { return nil }

func (o *submitOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	handled, cmd := o.confirm.Update(msg)
	if handled {
		return o, cmd
	}
	return o, nil
}

func (o *submitOverlay) View() string {
	var b strings.Builder
	b.WriteString(overlayTitleStyle.Render("Submit Period"))
	b.WriteString("\n\n")

	period := fmt.Sprintf("  %s — %s",
		o.from.Format("Jan 2, 2006"),
		o.to.Format("Jan 2, 2006"),
	)
	b.WriteString(period)
	b.WriteString("\n\n")

	if o.inMemoryCount > 0 {
		fmt.Fprintf(&b, "  %d generated entries will be persisted.\n\n", o.inMemoryCount)
	} else {
		b.WriteString("  No generated entries to persist.\n\n")
	}

	b.WriteString("  Submit?\n\n")

	b.WriteString(o.confirm.renderButtons())
	b.WriteString("\n\n")
	b.WriteString(overlayMutedStyle.Render("←/→ select  |  enter confirm  |  esc cancel"))

	return overlayBoxStyle.Render(b.String())
}
