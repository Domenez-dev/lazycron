package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
)

const version = "0.1"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println("lazycron v" + version)
		return
	}

	p := tea.NewProgram(InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}
