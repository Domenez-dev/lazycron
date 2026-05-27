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

func (m model) updateAddForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = listView
		return m, nil

	case "tab", "down":
		m.formFocus = (m.formFocus + 1) % 3
		for i := range m.formInputs {
			if i == m.formFocus {
				m.formInputs[i].Focus()
			} else {
				m.formInputs[i].Blur()
			}
		}
		return m, textinput.Blink

	case "shift+tab", "up":
		m.formFocus = (m.formFocus + 2) % 3
		for i := range m.formInputs {
			if i == m.formFocus {
				m.formInputs[i].Focus()
			} else {
				m.formInputs[i].Blur()
			}
		}
		return m, textinput.Blink

	case "enter":
		if m.formFocus < 2 {
			m.formFocus++
			for i := range m.formInputs {
				if i == m.formFocus {
					m.formInputs[i].Focus()
				} else {
					m.formInputs[i].Blur()
				}
			}
			return m, textinput.Blink
		}

		schedule := strings.TrimSpace(m.formInputs[0].Value())
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
