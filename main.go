package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	robfigcron "github.com/robfig/cron/v3"
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

func nextRun(schedule string) string {
	parser := robfigcron.NewParser(
		robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow,
	)
	if sched, err := parser.Parse(schedule); err == nil {
		return sched.Next(time.Now()).Format("01-02 15:04")
	}
	return "invalid"
}

func ParseCrontab() ([]Job, error) {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")

	var jobs []Job
	lineNum := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		lineNum++

		enabled := true
		parsed := line
		if strings.HasPrefix(line, "#") {
			enabled = false
			parsed = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}

		parts := strings.Fields(parsed)
		if len(parts) < 6 {
			continue
		}

		schedule := strings.Join(parts[:5], " ")
		rest := strings.Join(parts[5:], " ")

		cmd := rest
		comment := ""
		if i := strings.Index(rest, "#"); i != -1 {
			cmd = strings.TrimSpace(rest[:i])
			comment = strings.TrimSpace(rest[i+1:])
		}

		var nr string
		if enabled {
			nr = nextRun(schedule)
		} else {
			nr = "-----"
		}

		jobs = append(jobs, Job{
			Number:   fmt.Sprintf("%d", lineNum),
			Enabled:  enabled,
			Schedule: schedule,
			NextRun:  nr,
			Cmd:      cmd,
			Comment:  comment,
		})
	}
	return jobs, nil
}

func jobsToRows(jobs []Job) []table.Row {
	rows := make([]table.Row, len(jobs))
	for i, j := range jobs {
		e := "*"
		if !j.Enabled {
			e = "-"
		}
		rows[i] = table.Row{j.Number, e, j.Schedule, j.NextRun, j.Cmd, j.Comment}
	}
	return rows
}

func writeCrontab(jobs []Job) error {
	var sb strings.Builder
	for _, j := range jobs {
		line := fmt.Sprintf("%s %s", j.Schedule, j.Cmd)
		if j.Comment != "" {
			line += " # " + j.Comment
		}
		if !j.Enabled {
			line = "# " + line
		}
		sb.WriteString(line + "\n")
	}
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(sb.String())
	return cmd.Run()
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
		// Submit
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
		for i := range m.jobs {
			m.jobs[i].Number = fmt.Sprintf("%d", i+1)
		}
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

func (m model) bottomBar() string {
	k := lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	s := d.Render("  ")
	switch m.state {
	case listView:
		left := k.Render(" q") + d.Render(" quit") + s +
			k.Render("a") + d.Render(" add") + s +
			k.Render("e") + d.Render(" edit") + s +
			k.Render("d") + d.Render(" delete") + s +
			k.Render("t") + d.Render(" toggle") + s +
			k.Render("enter") + d.Render(" expand") + s +
			k.Render(":") + d.Render(" goto")
		right := k.Render("?") + d.Render(" help ")
		pad := m.width - lipgloss.Width(left) - lipgloss.Width(right)
		if pad < 1 {
			pad = 1
		}
		return left + strings.Repeat(" ", pad) + right
	case addView, editView:
		return k.Render(" enter") + d.Render(" next/confirm") + s +
			k.Render("tab") + d.Render(" next field") + s +
			k.Render("esc") + d.Render(" cancel")
	case deleteView:
		return k.Render(" y") + d.Render(" confirm delete") + s +
			k.Render("n") + d.Render("/") + k.Render("esc") + d.Render(" cancel")
	case gotoView:
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		bar := k.Render(":") + " " + m.gotoInput.View()
		if m.gotoErr != "" {
			bar += s + errStyle.Render(m.gotoErr)
		}
		return bar + s + k.Render("enter") + d.Render(" go") + s + k.Render("esc") + d.Render(" cancel")
	case expandView:
		return k.Render(" esc") + d.Render("/") + k.Render("enter") + d.Render(" close")
	case helpView:
		return k.Render(" esc") + d.Render("/") + k.Render("?") + d.Render("/") + k.Render("q") + d.Render(" close")
	}
	return ""
}

func (m model) withBottomBar(content string) string {
	bar := m.bottomBar()
	if m.height == 0 {
		return content + "\n" + bar
	}
	contentLines := strings.Count(content, "\n") + 1
	padding := m.height - contentLines - 1
	if padding < 0 {
		padding = 0
	}
	return content + strings.Repeat("\n", padding) + bar
}

func (m model) View() tea.View {
	switch m.state {
	case addView, editView:
		return m.viewAddForm()
	case deleteView:
		return m.viewDeleteConfirm()
	case gotoView:
		return m.viewGoto()
	case expandView:
		return m.viewExpand()
	case helpView:
		return m.viewHelp()
	}
	v := tea.NewView(m.withBottomBar(m.table.View()))
	v.AltScreen = true
	return v
}

func (m model) viewDeleteConfirm() tea.View {
	j := m.jobs[m.editIndex]
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(12)
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	title := bold.Render("  Delete Cron Job #"+j.Number+"?") + "\n\n"
	info := "  " + label.Render("Schedule") + val.Render(j.Schedule) + "\n" +
		"  " + label.Render("Command") + val.Render(j.Cmd) + "\n"
	if j.Comment != "" {
		info += "  " + label.Render("Comment") + val.Render(j.Comment) + "\n"
	}

	v := tea.NewView(m.withBottomBar(title + info))
	v.AltScreen = true
	return v
}

func (m model) viewAddForm() tea.View {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(12)
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	titleText := "  Add New Cron Job"
	if m.state == editView {
		titleText = "  Edit Cron Job #" + m.jobs[m.editIndex].Number
	}
	title := bold.Render(titleText) + "\n\n"

	fields := []string{"Schedule", "Command", "Comment"}
	var rows strings.Builder
	for i, f := range fields {
		rows.WriteString("  " + label.Render(f) + m.formInputs[i].View() + "\n")
	}

	var errLine string
	if m.formErr != "" {
		errLine = "\n  " + errStyle.Render(m.formErr) + "\n"
	}

	v := tea.NewView(m.withBottomBar(title + rows.String() + errLine))
	v.AltScreen = true
	return v
}

func (m model) viewGoto() tea.View {
	v := tea.NewView(m.withBottomBar(m.table.View()))
	v.AltScreen = true
	return v
}

func (m model) viewExpand() tea.View {
	j := m.jobs[m.editIndex]
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(12)
	val := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	enabled := "yes"
	if !j.Enabled {
		enabled = "no"
	}

	title := bold.Render("  Job #"+j.Number) + "\n\n"
	info := "  " + label.Render("Schedule") + val.Render(j.Schedule) + "\n" +
		"  " + label.Render("Next Run") + val.Render(j.NextRun) + "\n" +
		"  " + label.Render("Command") + val.Render(j.Cmd) + "\n" +
		"  " + label.Render("Comment") + val.Render(j.Comment) + "\n" +
		"  " + label.Render("Enabled") + val.Render(enabled) + "\n"

	v := tea.NewView(m.withBottomBar(title + info))
	v.AltScreen = true
	return v
}

func (m model) viewHelp() tea.View {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	section := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
	k := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Width(20)
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	row := func(keys, desc string) string {
		return "  " + k.Render(keys) + d.Render(desc) + "\n"
	}

	content := bold.Render("  Help") + "\n" +
		"  " + "LazyCron is A fast, keyboard-driven terminal UI for managing cron jobs on Linux." +
		"\n\n" +
		"  " + section.Render("Navigation") + "\n" +
		row("j / ↓", "move down") +
		row("k / ↑", "move up") +
		row("g / home", "go to top") +
		row("G / end", "go to bottom") +
		row(": <n> enter", "jump to row n") +
		row("enter", "expand job details") +
		"\n" +
		"  " + section.Render("Actions") + "\n" +
		row("a", "add new cron job") +
		row("e", "edit selected job") +
		row("d", "delete selected job") +
		row("t", "toggle enabled/disabled") +
		"\n" +
		"  " + section.Render("App") + "\n" +
		row("q", "quit") +
		row("?", "show/close this help")

	v := tea.NewView(m.withBottomBar(content))
	v.AltScreen = true
	return v
}

func main() {
	p := tea.NewProgram(InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
