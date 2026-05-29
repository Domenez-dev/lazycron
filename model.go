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

type scheduleMode int

const (
	scheduleModeManual scheduleMode = iota
	scheduleModeBuild
)

type builderField struct {
	options []string
	optIdx  int    // -1 means custom text
	custom  string
}

func (f builderField) value() string {
	if f.optIdx >= 0 && f.optIdx < len(f.options) {
		return f.options[f.optIdx]
	}
	return f.custom
}

// Preset cycling values for each of the 5 cron fields (min, hr, dom, month, dow).
// Category boundaries are tracked in builderTypeJumps below.
var builderPresets = [5][]string{
	// Minute: *(0), num(1-12), */(13-18), -(19-22), ,(23-25)
	{
		"*",
		"0", "5", "10", "15", "20", "25", "30", "35", "40", "45", "50", "55",
		"*/2", "*/5", "*/10", "*/15", "*/20", "*/30",
		"0-30", "0-15", "15-45", "30-59",
		"0,30", "0,15,30,45", "0,20,40",
	},
	// Hour: *(0), num(1-24), */(25-30), -(31-34), ,(35-37)
	{
		"*",
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11",
		"12", "13", "14", "15", "16", "17", "18", "19", "20", "21", "22", "23",
		"*/2", "*/3", "*/4", "*/6", "*/8", "*/12",
		"9-17", "8-18", "0-12", "12-23",
		"9,17", "8,12,17", "0,6,12,18",
	},
	// DOM: *(0), num(1-8), */(9-12), -(13-15), ,(16-17)
	{
		"*",
		"1", "5", "10", "15", "20", "25", "28", "31",
		"*/2", "*/5", "*/7", "*/14",
		"1-15", "15-31", "1-7",
		"1,15", "1,10,20",
	},
	// Month: *(0), num(1-12), name(13-24), -(25-27), ,(28-29)
	{
		"*",
		"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12",
		"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec",
		"1-6", "6-12", "3-9",
		"1,7", "3,6,9,12",
	},
	// DOW: *(0), num(1-7), name(8-14), -(15-16), ,(17-19)
	{
		"*",
		"0", "1", "2", "3", "4", "5", "6",
		"sun", "mon", "tue", "wed", "thu", "fri", "sat",
		"1-5", "0-6",
		"1,3,5", "0,6", "1,2,3,4,5",
	},
}

// builderTypeJumps[field] = start indices of each type category, used by alt+j.
var builderTypeJumps = [5][]int{
	{0, 1, 13, 19, 23}, // Minute:  *, num, */, -, ,
	{0, 1, 25, 31, 35}, // Hour:    *, num, */, -, ,
	{0, 1, 9, 13, 16},  // DOM:     *, num, */, -, ,
	{0, 1, 13, 25, 28}, // Month:   *, num, name, -, ,
	{0, 1, 8, 15, 17},  // DOW:     *, num, name, -, ,
}

// builderTypeLabels[field][cat] is shown in the "type:" hint while navigating.
var builderTypeLabels = [5][]string{
	{"*", "num", "*/", "-", ","},
	{"*", "num", "*/", "-", ","},
	{"*", "num", "*/", "-", ","},
	{"*", "num", "name", "-", ","},
	{"*", "num", "name", "-", ","},
}

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
	// schedule builder
	schedMode    scheduleMode
	builder      [5]builderField
	builderFocus int
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

func initBuilder() [5]builderField {
	var b [5]builderField
	for i := range b {
		b[i] = builderField{options: builderPresets[i], optIdx: 0}
	}
	return b
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

	return model{table: t, jobs: jobs, formInputs: newFormInputs(), builder: initBuilder()}
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
