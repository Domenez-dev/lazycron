package main

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	robfigcron "github.com/robfig/cron/v3"
)

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

		nr := nextRun(schedule)
		if !enabled {
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
