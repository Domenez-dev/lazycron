package main

import (
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

var (
	BaseStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Foreground(lipgloss.Color("205"))

	CellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)
)

func StyledTable(t table.Model) table.Model {
	s := table.DefaultStyles()

	s.Header = HeaderStyle
	s.Cell = CellStyle
	s.Selected = SelectedStyle

	t.SetStyles(s)

	return t
}
