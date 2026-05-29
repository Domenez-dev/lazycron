package main

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.state == listView {
			m.table.SetWidth(msg.Width)
			m.table.SetHeight(msg.Height - 2)
			m.resizeTable()
		}

	case tea.KeyMsg:
		if m.state == addView || m.state == editView {
			return m.updateAddForm(msg)
		}
		if m.state == deleteView {
			return m.updateDeleteConfirm(msg)
		}
		if m.state == gotoView {
			return m.updateGoto(msg)
		}
		if m.state == expandView {
			return m.updateExpand(msg)
		}
		if m.state == helpView {
			switch msg.String() {
			case "esc", "?", "q":
				m.state = listView
			}
			return m, nil
		}

		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "a":
			m.state = addView
			m.formInputs = newFormInputs()
			m.builder = initBuilder()
			m.builderFocus = 0
			m.formFocus = 0
			m.formErr = ""
			return m, textinput.Blink

		case "e":
			cursor := m.table.Cursor()
			if len(m.jobs) == 0 {
				return m, nil
			}
			j := m.jobs[cursor]
			inputs := newFormInputs()
			inputs[0].SetValue(j.Schedule)
			inputs[1].SetValue(j.Cmd)
			inputs[2].SetValue(j.Comment)
			m.formInputs = inputs
			m.builder = parseScheduleToBuilder(j.Schedule)
			m.builderFocus = 0
			m.formFocus = 0
			m.formErr = ""
			m.editIndex = cursor
			m.state = editView
			return m, textinput.Blink

		case "d":
			if len(m.jobs) == 0 {
				return m, nil
			}
			m.editIndex = m.table.Cursor()
			m.state = deleteView
			return m, nil

		case "t", " ":
			cursor := m.table.Cursor()
			if len(m.jobs) == 0 {
				return m, nil
			}
			m.jobs[cursor].Enabled = !m.jobs[cursor].Enabled
			if m.jobs[cursor].Enabled {
				m.jobs[cursor].NextRun = nextRun(m.jobs[cursor].Schedule)
			} else {
				m.jobs[cursor].NextRun = "-----"
			}
			_ = writeCrontab(m.jobs)
			m.table.SetRows(jobsToRows(m.jobs))
			return m, nil

		case "up", "k":
			if len(m.jobs) > 0 {
				if m.table.Cursor() == 0 {
					m.table.SetCursor(len(m.jobs) - 1)
				} else {
					m.table.MoveUp(1)
				}
			}
			return m, nil

		case "down", "j":
			if len(m.jobs) > 0 {
				if m.table.Cursor() == len(m.jobs)-1 {
					m.table.SetCursor(0)
				} else {
					m.table.MoveDown(1)
				}
			}
			return m, nil

		case "enter":
			if len(m.jobs) > 0 {
				m.editIndex = m.table.Cursor()
				m.state = expandView
			}
			return m, nil

		case ":":
			m.gotoInput = textinput.New()
			m.gotoInput.Placeholder = "row #"
			m.gotoInput.CharLimit = 4
			m.gotoInput.Focus()
			m.gotoErr = ""
			m.state = gotoView
			return m, textinput.Blink

		case "?":
			m.state = helpView
			return m, nil
		}
	}

	if m.state == listView {
		var cmd tea.Cmd
		m.table, cmd = m.table.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) updateGoto(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = listView
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.gotoInput.Value())
		n, err := strconv.Atoi(val)
		if err != nil || n < 1 || n > len(m.jobs) {
			m.gotoErr = fmt.Sprintf("invalid row (1-%d)", len(m.jobs))
			return m, nil
		}
		m.table.SetCursor(n - 1)
		m.state = listView
		return m, nil
	}
	var cmd tea.Cmd
	m.gotoInput, cmd = m.gotoInput.Update(msg)
	return m, cmd
}

func (m model) updateExpand(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.state = listView
	}
	return m, nil
}

// scheduleValue returns the current schedule string from whichever input mode is active.
func (m model) scheduleValue() string {
	if m.schedMode == scheduleModeBuild {
		return builderToSchedule(m.builder)
	}
	return strings.TrimSpace(m.formInputs[0].Value())
}

// toggleScheduleMode switches between manual and builder modes, syncing values.
func (m model) toggleScheduleMode() model {
	if m.schedMode == scheduleModeManual {
		m.builder = parseScheduleToBuilder(strings.TrimSpace(m.formInputs[0].Value()))
		m.builderFocus = 0
		m.schedMode = scheduleModeBuild
		m.formInputs[0].Blur()
	} else {
		sched := builderToSchedule(m.builder)
		m.formInputs[0].SetValue(sched)
		m.schedMode = scheduleModeManual
		if m.formFocus == 0 {
			m.formInputs[0].Focus()
		}
	}
	return m
}

// focusFormField updates which text input is focused, respecting builder mode.
func (m model) focusFormField(idx int) model {
	m.formFocus = idx
	for i := range m.formInputs {
		// Never focus input[0] in builder mode — the builder owns that field.
		if i == m.formFocus && !(i == 0 && m.schedMode == scheduleModeBuild) {
			m.formInputs[i].Focus()
		} else {
			m.formInputs[i].Blur()
		}
	}
	return m
}

func (m model) updateAddForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "esc" {
		m.state = listView
		return m, nil
	}

	if key == "alt+m" {
		m = m.toggleScheduleMode()
		return m, textinput.Blink
	}

	// In builder mode, schedule field (formFocus==0) is handled separately.
	if m.schedMode == scheduleModeBuild && m.formFocus == 0 {
		return m.updateBuilderField(msg)
	}

	switch key {
	case "tab", "down":
		m = m.focusFormField((m.formFocus + 1) % 3)
		if m.formFocus == 0 {
			m.builderFocus = 0
		}
		return m, textinput.Blink

	case "shift+tab", "up":
		m = m.focusFormField((m.formFocus + 2) % 3)
		if m.formFocus == 0 {
			m.builderFocus = 0
		}
		return m, textinput.Blink

	case "enter":
		if m.formFocus < 2 {
			m = m.focusFormField(m.formFocus + 1)
			return m, textinput.Blink
		}

		schedule := m.scheduleValue()
		command := strings.TrimSpace(m.formInputs[1].Value())
		comment := strings.TrimSpace(m.formInputs[2].Value())

		if schedule == "" || command == "" {
			m.formErr = "schedule and command are required"
			return m, nil
		}
		if nextRun(schedule) == "invalid" {
			m.formErr = "invalid cron schedule"
			return m, nil
		}

		if m.state == editView {
			j := &m.jobs[m.editIndex]
			j.Schedule = schedule
			j.Cmd = command
			j.Comment = comment
			if j.Enabled {
				j.NextRun = nextRun(schedule)
			}
		} else {
			m.jobs = append(m.jobs, Job{
				Number:   fmt.Sprintf("%d", len(m.jobs)+1),
				Enabled:  true,
				Schedule: schedule,
				NextRun:  nextRun(schedule),
				Cmd:      command,
				Comment:  comment,
			})
		}
		_ = writeCrontab(m.jobs)

		rows := jobsToRows(m.jobs)
		m.table.SetRows(rows)
		m.table.SetHeight(len(rows) + 1)
		m.state = listView
		return m, nil
	}

	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

// updateBuilderField handles key events when the schedule builder is focused.
func (m model) updateBuilderField(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	bf := &m.builder[m.builderFocus]

	switch key {
	// tab exits the builder entirely and moves to the Command field.
	case "tab":
		m = m.focusFormField(1)
		return m, textinput.Blink

	// right/enter advance within builder sub-fields; on the last one, go to Command.
	case "right", "enter":
		if m.builderFocus < 4 {
			m.builderFocus++
		} else {
			m = m.focusFormField(1)
		}
		return m, textinput.Blink

	// left/shift+tab go back within builder sub-fields.
	case "left", "shift+tab":
		if m.builderFocus > 0 {
			m.builderFocus--
		}
		return m, textinput.Blink

	case "down":
		if bf.optIdx >= 0 {
			bf.optIdx = (bf.optIdx + 1) % len(bf.options)
		} else {
			bf.optIdx = 0
			bf.custom = ""
		}
		return m, nil

	case "up":
		if bf.optIdx >= 0 {
			bf.optIdx = (bf.optIdx - 1 + len(bf.options)) % len(bf.options)
		} else {
			bf.optIdx = len(bf.options) - 1
			bf.custom = ""
		}
		return m, nil

	// alt+j jumps to the next type category (*, num, */, name, -).
	case "alt+j":
		jumps := builderTypeJumps[m.builderFocus]
		if bf.optIdx == -1 {
			bf.optIdx = 0
			bf.custom = ""
		} else {
			cat := 0
			for i := len(jumps) - 1; i >= 0; i-- {
				if bf.optIdx >= jumps[i] {
					cat = i
					break
				}
			}
			bf.optIdx = jumps[(cat+1)%len(jumps)]
		}
		return m, nil

	case "backspace", "ctrl+h":
		if bf.optIdx == -1 {
			if len(bf.custom) > 0 {
				bf.custom = bf.custom[:len(bf.custom)-1]
			}
			if bf.custom == "" {
				bf.optIdx = 0
			}
		}
		return m, nil

	default:
		// Any printable character enters/appends to custom mode.
		// No special-casing for '*' so "*/3" can be typed naturally.
		if len(key) == 1 {
			if bf.optIdx >= 0 {
				bf.custom = key
				bf.optIdx = -1
			} else {
				bf.custom += key
			}
		}
		return m, nil
	}
}

func (m model) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y":
		m.jobs = append(m.jobs[:m.editIndex], m.jobs[m.editIndex+1:]...)
		renumberJobs(m.jobs)
		_ = writeCrontab(m.jobs)
		rows := jobsToRows(m.jobs)
		if len(rows) == 0 {
			rows = []table.Row{{"-", "-", "No cronjobs found", "-", "-", "-"}}
		}
		m.table.SetRows(rows)
		m.table.SetHeight(len(rows) + 1)
		m.state = listView
	case "n", "esc":
		m.state = listView
	}
	return m, nil
}
