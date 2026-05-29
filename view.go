package main

import (
	"fmt"
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
		if m.schedMode == scheduleModeBuild && m.formFocus == 0 {
			return k.Render(" ↑↓") + d.Render(" cycle") + s +
				k.Render("←→") + d.Render(" field") + s +
				k.Render("alt+j") + d.Render(" type") + s +
				k.Render("alt+m") + d.Render(" manual") + s +
				k.Render("tab") + d.Render(" command") + s +
				k.Render("esc") + d.Render(" cancel")
		}
		modeHint := "builder"
		if m.schedMode == scheduleModeBuild {
			modeHint = "manual"
		}
		return k.Render(" enter") + d.Render(" next/confirm") + s +
			k.Render("tab") + d.Render(" next field") + s +
			k.Render("alt+m") + d.Render(" "+modeHint+" mode") + s +
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
	var content string
	if m.schedMode == scheduleModeBuild {
		content = m.viewAddFormBuilder()
	} else {
		content = m.viewAddFormManual()
	}
	v := tea.NewView(m.withBottomBar(content))
	v.AltScreen = true
	return v
}

func (m model) viewAddFormManual() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(12)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	hintPad := strings.Repeat(" ", 12)

	titleText := "  Add New Cron Job"
	if m.state == editView {
		titleText = "  Edit Cron Job #" + m.jobs[m.editIndex].Number
	}
	title := bold.Render(titleText) + "\n\n"

	fields := []string{"Schedule", "Command", "Comment"}
	var rows strings.Builder
	for i, f := range fields {
		rows.WriteString("  " + label.Render(f) + m.formInputs[i].View() + "\n")
		if i == 0 {
			if translation := humanReadableSchedule(m.formInputs[0].Value()); translation != "" {
				rows.WriteString("  " + hintPad + hint.Render(translation) + "\n")
			}
		}
	}

	var errLine string
	if m.formErr != "" {
		errLine = "\n  " + errStyle.Render(m.formErr) + "\n"
	}

	return title + rows.String() + errLine
}

func (m model) viewAddFormBuilder() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Width(12)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	focusedBox := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	normalBox := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	customBox := lipgloss.NewStyle().Foreground(lipgloss.Color("226")) // yellow = custom value

	titleText := "  Add New Cron Job"
	if m.state == editView {
		titleText = "  Edit Cron Job #" + m.jobs[m.editIndex].Number
	}
	title := bold.Render(titleText) + "\n\n"

	subNames := [5]string{"Minute", "Hour", "Day", "Month", "weekday"}
	// Each column is 8 chars wide: [%-5s] (7) + 1 space, except last.
	colW := 8

	var labelRow strings.Builder
	labelRow.WriteString("  " + label.Render("Schedule"))
	for i, name := range subNames {
		if i < 4 {
			labelRow.WriteString(fmt.Sprintf("%-*s", colW, name))
		} else {
			labelRow.WriteString(name)
		}
	}
	labelRow.WriteString("\n")

	// Boxes row
	var boxRow strings.Builder
	boxRow.WriteString("  " + label.Render(""))
	for i, bf := range m.builder {
		val := bf.value()
		if len(val) > 5 {
			val = val[:5]
		}
		box := fmt.Sprintf("[%-5s]", val)

		var rendered string
		switch {
		case i == m.builderFocus && bf.optIdx == -1:
			rendered = focusedBox.Render(box)
		case i == m.builderFocus:
			rendered = focusedBox.Render(box)
		case bf.optIdx == -1:
			rendered = customBox.Render(box)
		default:
			rendered = normalBox.Render(box)
		}

		if i < 4 {
			// Pad to colW using lipgloss-aware width so ANSI codes don't throw off alignment.
			rendered += strings.Repeat(" ", colW-lipgloss.Width(box))
		}
		boxRow.WriteString(rendered)
	}
	boxRow.WriteString("\n")

	var typeRow string
	{
		focusedBf := m.builder[m.builderFocus]
		jumps := builderTypeJumps[m.builderFocus]
		labels := builderTypeLabels[m.builderFocus]
		typeName := "custom"
		if focusedBf.optIdx >= 0 {
			cat := 0
			for i := len(jumps) - 1; i >= 0; i-- {
				if focusedBf.optIdx >= jumps[i] {
					cat = i
					break
				}
			}
			typeName = labels[cat]
		}
		colOffset := m.builderFocus * colW
		typeRow = "  " + label.Render("") + strings.Repeat(" ", colOffset) +
			hint.Render("type:"+typeName) + "\n"
	}

	hintPad := strings.Repeat(" ", 12)
	schedule := builderToSchedule(m.builder)
	rawLine := "  " + hintPad + hint.Render(schedule) + "\n"

	var transLine string
	if translation := humanReadableSchedule(schedule); translation != "" {
		transLine = "  " + hintPad + hint.Render(translation) + "\n"
	}

	otherFields := []string{"Command", "Comment"}
	var otherRows strings.Builder
	for i, f := range otherFields {
		otherRows.WriteString("  " + label.Render(f) + m.formInputs[i+1].View() + "\n")
	}

	var errLine string
	if m.formErr != "" {
		errLine = "\n  " + errStyle.Render(m.formErr) + "\n"
	}

	return title + labelRow.String() + boxRow.String() + typeRow + rawLine + transLine + otherRows.String() + errLine
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
		"  " + section.Render("Schedule form") + "\n" +
		row("alt+m", "toggle manual/builder mode") +
		row("↑↓ (builder)", "cycle preset values") +
		row("tab / →", "next sub-field (builder)") +
		"\n" +
		"  " + section.Render("App") + "\n" +
		row("q", "quit") +
		row("?", "show/close this help")

	v := tea.NewView(m.withBottomBar(content))
	v.AltScreen = true
	return v
}
