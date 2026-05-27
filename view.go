package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

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

// withBottomBar pads content to fill the terminal height and pins the bar to the last row.
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

	content := bold.Render("  Help") + "\n\n" +
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
		row("t / space", "toggle enabled/disabled") +
		"\n" +
		"  " + section.Render("App") + "\n" +
		row("q", "quit") +
		row("?", "show/close this help")

	v := tea.NewView(m.withBottomBar(content))
	v.AltScreen = true
	return v
}
