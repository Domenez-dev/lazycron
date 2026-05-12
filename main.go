package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	robfigcron "github.com/robfig/cron/v3"
)

type model struct {
	table  table.Model
	width  int
	height int
}

type Job struct {
	Number   string
	Enabled  string
	Schedule string
	NextRun  string
	Cmd      string
	Comment  string
	Disabled bool
}

func ParseCrontab() ([]table.Row, error) {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")

	var rows []table.Row
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		number := fmt.Sprintf("%d", i+1)
		enabled := "*"
		if strings.HasPrefix(line, "#") {
			enabled = "-"
			line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}

		parts := strings.Fields(line)
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
		var next_run string
		parser := robfigcron.NewParser(
			robfigcron.Minute | robfigcron.Hour | robfigcron.Dom | robfigcron.Month | robfigcron.Dow,
		)
		if sched, err := parser.Parse(schedule); err == nil {
			next := sched.Next(time.Now())
			next_run = next.Format("01-02 15:04")
		} else {
			next_run = "invalid"
		}

		rows = append(rows, table.Row{
			number,
			enabled,
			schedule,
			next_run,
			cmd,
			comment,
		})
	}
	return rows, nil
}

func (m *model) resizeTable() {
	cols := m.table.Columns()
	used := 2 + 2 + 15 + 15 + 40

	remaining := m.width - used - 6 // for padding
	if remaining < 10 {
		remaining = 10
	}
	cols[4].Width = remaining // Comments

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

	rows, err := ParseCrontab()
	if len(rows) == 0 {
		rows = []table.Row{
			{"-", "-", "No cronjobs found", "-"},
		}
	}
	if err != nil {
		return model{}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)

	t = StyledTable(t)

	return model{table: t}
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
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() tea.View {
	v := tea.NewView(m.table.View() + "\nPress q to quit.")
	v.AltScreen = true
	return v
}

func main() {
	p := tea.NewProgram(InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
