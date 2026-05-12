package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	robfigcron "github.com/robfig/cron/v3"
)

type model struct {
	table  table.Model
	jobs   []Job
	width  int
	height int
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

	return model{table: t, jobs: jobs}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width)
		m.table.SetHeight(msg.Height - 2)
		m.resizeTable()

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "t":
			cursor := m.table.Cursor()

			m.jobs[cursor].Enabled = !m.jobs[cursor].Enabled
			if m.jobs[cursor].Enabled {
				m.jobs[cursor].NextRun = nextRun(m.jobs[cursor].Schedule)
			} else {
				m.jobs[cursor].NextRun = "-----"
			}

			_ = writeCrontab(m.jobs)
			m.table.SetRows(jobsToRows(m.jobs))
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	legend := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("  q") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(" quit") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("  t") +
		lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(" toggle enabled/disabled")
	v := tea.NewView(m.table.View() + "\n" + legend)
	v.AltScreen = true
	return v
}

func main() {
	p := tea.NewProgram(InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
