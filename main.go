package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
