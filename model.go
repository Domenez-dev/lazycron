package main

import (
	"fmt"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type viewState int

const (
	listView viewState = iota
	addView
	editView
	deleteView
	gotoView
	expandView
	helpView
)

type model struct {
	table      table.Model
	jobs       []Job
	width      int
	height     int
	state      viewState
	formInputs [3]textinput.Model
	formFocus  int
	formErr    string
	editIndex  int
	gotoInput  textinput.Model
	gotoErr    string
}

type Job struct {
	Number   string
	Enabled  bool
	Schedule string
	NextRun  string
	Cmd      string
	Comment  string
}

func newFormInputs() [3]textinput.Model {
	schedule := textinput.New()
	schedule.Placeholder = "* * * * *"
	schedule.Focus()
	schedule.CharLimit = 64

	command := textinput.New()
	command.Placeholder = "/path/to/script.sh"
	command.CharLimit = 256

	comment := textinput.New()
	comment.Placeholder = "optional description"
	comment.CharLimit = 128

	return [3]textinput.Model{schedule, command, comment}
}

func (m *model) resizeTable() {
	cols := m.table.Columns()
	used := 2 + 2 + 15 + 15 + 40 // #, E, Schedule, NextRun, Comment
	remaining := m.width - used - 6
	if remaining < 10 {
		remaining = 10
	}
	cols[4].Width = remaining
	m.table.SetColumns(cols)
}

func InitialModel() model {
	columns := []table.Column{
		{Title: "#", Width: 2},
		{Title: "E", Width: 2},
		{Title: "Schedule", Width: 15},
		{Title: "Next Run", Width: 15},
		{Title: "Command", Width: 40},
		{Title: "Comment", Width: 40},
	}

	jobs, err := ParseCrontab()
	if err != nil || len(jobs) == 0 {
		jobs = []Job{}
	}

	rows := jobsToRows(jobs)
	if len(rows) == 0 {
		rows = []table.Row{{"-", "-", "No cronjobs found", "-", "-", "-"}}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)
	t = StyledTable(t)

	return model{table: t, jobs: jobs, formInputs: newFormInputs()}
}

func (m model) Init() tea.Cmd {
	return nil
}

// renumberJobs keeps the Number field in sync after insertions/deletions.
func renumberJobs(jobs []Job) {
	for i := range jobs {
		jobs[i].Number = fmt.Sprintf("%d", i+1)
	}
}
