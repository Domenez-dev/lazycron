package main

import (
	"fmt"
	"os/exec"
	"strings"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

type model struct {
	table  table.Model
	width  int
	height int
}

func ParseCrontab() ([]table.Row, error) {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(out), "\n")

	var rows []table.Row
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		enabled := "yes"
		if strings.HasPrefix(line, "#") {
			enabled = "no"
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

		rows = append(rows, table.Row{
			enabled,
			schedule,
			cmd,
			comment,
		})
	}
	return rows, nil
}

func (m *model) resizeTable() {
	cols := m.table.Columns()

	// fixed widths
	cols[0].Width = 8  // Enabled
	cols[1].Width = 15 // Schedule
	cols[2].Width = 40 // Command

	// remaining width for Comment
	used := cols[0].Width + cols[1].Width + cols[2].Width

	// account for spacing/padding (~4–6 depending on styles)
	remaining := m.width - used - 6
	if remaining < 10 {
		remaining = 10 // minimum to avoid collapse
	}
	cols[3].Width = remaining

	m.table.SetColumns(cols)
}

func InitialModel() model {
	columns := []table.Column{
		{Title: "Enabled", Width: 8},
		{Title: "Schedule", Width: 15},
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
