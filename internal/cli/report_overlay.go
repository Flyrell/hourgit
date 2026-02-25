package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
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
	editFieldDuration editField = iota
	editFieldTask
	editFieldMessage
	editFieldConfirm
)

type editOverlay struct {
	entry    timetrack.CellEntry
	duration string
	task     string
	message  string
	field    editField
	err      string
}

func newEditOverlay(ce timetrack.CellEntry) *editOverlay {
	return &editOverlay{
		entry:    ce,
		duration: entry.FormatMinutes(ce.Minutes),
		task:     ce.Task,
		message:  ce.Message,
	}
}

func (o *editOverlay) Init() tea.Cmd { return nil }

func (o *editOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return o, overlayResultMsg("cancel", nil)
		case "tab", "down":
			if o.field < editFieldConfirm {
				o.field++
			}
		case "shift+tab", "up":
			if o.field > editFieldDuration {
				o.field--
			}
		case "enter":
			if o.field == editFieldConfirm {
				// Validate duration
				mins, err := entry.ParseDuration(o.duration)
				if err != nil {
					o.err = "Invalid duration (e.g. 2h30m, 90m)"
					return o, nil
				}
				o.entry.Minutes = mins
				o.entry.Task = o.task
				o.entry.Message = o.message
				o.err = ""
				return o, overlayResultMsg("edit", nil)
			}
			// Enter on a field moves to next
			if o.field < editFieldConfirm {
				o.field++
			}
		case "backspace":
			o.deleteChar()
		default:
			if len(msg.String()) == 1 {
				o.insertChar(msg.String())
			}
		}
	}
	return o, nil
}

func (o *editOverlay) insertChar(ch string) {
	switch o.field {
	case editFieldDuration:
		o.duration += ch
	case editFieldTask:
		o.task += ch
	case editFieldMessage:
		o.message += ch
	}
}

func (o *editOverlay) deleteChar() {
	switch o.field {
	case editFieldDuration:
		if len(o.duration) > 0 {
			o.duration = o.duration[:len(o.duration)-1]
		}
	case editFieldTask:
		if len(o.task) > 0 {
			o.task = o.task[:len(o.task)-1]
		}
	case editFieldMessage:
		if len(o.message) > 0 {
			o.message = o.message[:len(o.message)-1]
		}
	}
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
	addFieldDuration addField = iota
	addFieldTask
	addFieldMessage
	addFieldConfirm
)

type addOverlay struct {
	day      int
	month    time.Month
	year     int
	task     string // pre-filled from selected row
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
	}
}

func (o *addOverlay) Init() tea.Cmd { return nil }

func (o *addOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return o, overlayResultMsg("cancel", nil)
		case "tab", "down":
			if o.field < addFieldConfirm {
				o.field++
			}
		case "shift+tab", "up":
			if o.field > addFieldDuration {
				o.field--
			}
		case "enter":
			if o.field == addFieldConfirm {
				if o.duration == "" {
					o.err = "Duration is required"
					return o, nil
				}
				_, err := entry.ParseDuration(o.duration)
				if err != nil {
					o.err = "Invalid duration (e.g. 2h30m, 90m)"
					return o, nil
				}
				o.err = ""
				return o, overlayResultMsg("add", nil)
			}
			if o.field < addFieldConfirm {
				o.field++
			}
		case "backspace":
			o.deleteChar()
		default:
			if len(msg.String()) == 1 {
				o.insertChar(msg.String())
			}
		}
	}
	return o, nil
}

func (o *addOverlay) insertChar(ch string) {
	switch o.field {
	case addFieldDuration:
		o.duration += ch
	case addFieldTask:
		o.task += ch
	case addFieldMessage:
		o.message += ch
	}
}

func (o *addOverlay) deleteChar() {
	switch o.field {
	case addFieldDuration:
		if len(o.duration) > 0 {
			o.duration = o.duration[:len(o.duration)-1]
		}
	case addFieldTask:
		if len(o.task) > 0 {
			o.task = o.task[:len(o.task)-1]
		}
	case addFieldMessage:
		if len(o.message) > 0 {
			o.message = o.message[:len(o.message)-1]
		}
	}
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
		Start:     time.Date(o.year, o.month, o.day, 9, 0, 0, 0, time.UTC),
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
	entry  timetrack.CellEntry
	cursor int // 0 = yes, 1 = no
}

func newRemoveOverlay(ce timetrack.CellEntry) *removeOverlay {
	return &removeOverlay{entry: ce, cursor: 1} // default to "no"
}

func (o *removeOverlay) Init() tea.Cmd { return nil }

func (o *removeOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return o, overlayResultMsg("cancel", nil)
		case "left", "h", "right", "l", "tab":
			o.cursor = 1 - o.cursor
		case "enter":
			if o.cursor == 0 {
				return o, overlayResultMsg("remove", nil)
			}
			return o, overlayResultMsg("cancel", nil)
		case "y":
			return o, overlayResultMsg("remove", nil)
		case "n":
			return o, overlayResultMsg("cancel", nil)
		}
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

	yes := "  [Yes]"
	no := "  [No]"
	if o.cursor == 0 {
		yes = overlayActiveStyle.Render("> [Yes]")
	}
	if o.cursor == 1 {
		no = overlayActiveStyle.Render("> [No]")
	}
	b.WriteString(yes + "    " + no)
	b.WriteString("\n\n")
	b.WriteString(overlayMutedStyle.Render("←/→ select  |  enter confirm  |  esc cancel"))

	return overlayBoxStyle.Render(b.String())
}

// --- Submit Overlay ---
// Confirmation to persist all in-memory entries.

type submitOverlay struct {
	inMemoryCount int
	from, to      time.Time
	cursor        int // 0 = yes, 1 = no
}

func newSubmitOverlay(inMemoryCount int, from, to time.Time) *submitOverlay {
	return &submitOverlay{
		inMemoryCount: inMemoryCount,
		from:          from,
		to:            to,
		cursor:        1, // default to "no"
	}
}

func (o *submitOverlay) Init() tea.Cmd { return nil }

func (o *submitOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return o, overlayResultMsg("cancel", nil)
		case "left", "h", "right", "l", "tab":
			o.cursor = 1 - o.cursor
		case "enter":
			if o.cursor == 0 {
				return o, overlayResultMsg("submit", nil)
			}
			return o, overlayResultMsg("cancel", nil)
		case "y":
			return o, overlayResultMsg("submit", nil)
		case "n":
			return o, overlayResultMsg("cancel", nil)
		}
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

	yes := "  [Yes]"
	no := "  [No]"
	if o.cursor == 0 {
		yes = overlayActiveStyle.Render("> [Yes]")
	}
	if o.cursor == 1 {
		no = overlayActiveStyle.Render("> [No]")
	}
	b.WriteString(yes + "    " + no)
	b.WriteString("\n\n")
	b.WriteString(overlayMutedStyle.Render("←/→ select  |  enter confirm  |  esc cancel"))

	return overlayBoxStyle.Render(b.String())
}
