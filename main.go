package main

import (
	"fmt"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct{}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	out, _ := exec.Command("crontab", "-l").Output()

	message := "Cronjobs on your machine:\n\n" + string(out) + "\n\nPress q to quit.\n"
	return message
}

func main() {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
