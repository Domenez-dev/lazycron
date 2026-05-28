package main

import (
	"fmt"
	"strconv"
	"strings"
)

var dowName = map[string]string{
	"0": "Sunday", "1": "Monday", "2": "Tuesday", "3": "Wednesday",
	"4": "Thursday", "5": "Friday", "6": "Saturday", "7": "Sunday",
	"sun": "Sunday", "mon": "Monday", "tue": "Tuesday", "wed": "Wednesday",
	"thu": "Thursday", "fri": "Friday", "sat": "Saturday",
}

var monthName = map[string]string{
	"1": "January", "2": "February", "3": "March", "4": "April",
	"5": "May", "6": "June", "7": "July", "8": "August",
	"9": "September", "10": "October", "11": "November", "12": "December",
	"jan": "January", "feb": "February", "mar": "March", "apr": "April",
	"may": "May", "jun": "June", "jul": "July", "aug": "August",
	"sep": "September", "oct": "October", "nov": "November", "dec": "December",
}

// humanReadableSchedule converts a 5-field cron expression to a plain-English description.
// Returns "" if the expression is empty, incomplete, or too complex to describe concisely.
func humanReadableSchedule(expr string) string {
	parts := strings.Fields(strings.TrimSpace(expr))
	if len(parts) != 5 {
		return ""
	}

	minF := strings.ToLower(parts[0])
	hourF := strings.ToLower(parts[1])
	domF := strings.ToLower(parts[2])
	monthF := strings.ToLower(parts[3])
	dowF := strings.ToLower(parts[4])

	if minF == "*" && hourF == "*" && domF == "*" && monthF == "*" && dowF == "*" {
		return "Every minute."
	}

	var segments []string

	if t := buildTimeDesc(minF, hourF); t != "" {
		segments = append(segments, t)
	}

	if dowF != "*" {
		if d := describeField(dowF, dowName); d != "" {
			segments = append(segments, "on "+d)
		}
	} else if domF != "*" {
		if d := buildDOMDesc(domF); d != "" {
			segments = append(segments, d)
		}
	}

	if monthF != "*" {
		if m := describeField(monthF, monthName); m != "" {
			segments = append(segments, "in "+m)
		}
	}

	if len(segments) == 0 {
		return ""
	}

	result := strings.Join(segments, ", ")
	return strings.ToUpper(result[:1]) + result[1:] + "."
}

func buildTimeDesc(min, hour string) string {
	// Every minute
	if min == "*" && hour == "*" {
		return "every minute"
	}

	// Every N minutes (*/N * * * *)
	if step, ok := parseStep(min); ok && hour == "*" {
		if step == 1 {
			return "every minute"
		}
		return fmt.Sprintf("every %d minutes", step)
	}

	// Exact time (MM HH)
	if isInt(min) && isInt(hour) {
		h, _ := strconv.Atoi(hour)
		m, _ := strconv.Atoi(min)
		return fmt.Sprintf("at %02d:%02d", h, m)
	}

	// Every N hours at minute 0 (0 */N)
	if (min == "0" || min == "00") && hour != "*" {
		if step, ok := parseStep(hour); ok {
			if step == 1 {
				return "every hour"
			}
			return fmt.Sprintf("every %d hours", step)
		}
	}

	// Specific minute, every hour (* * wildcard hour)
	if isInt(min) && hour == "*" {
		m, _ := strconv.Atoi(min)
		if m == 0 {
			return "at the start of every hour"
		}
		return fmt.Sprintf("at minute %d of every hour", m)
	}

	// Every minute of a specific hour (* HH)
	if min == "*" && isInt(hour) {
		h, _ := strconv.Atoi(hour)
		return fmt.Sprintf("every minute of hour %02d", h)
	}

	// Comma-separated hours with minute 0: 0 9,17 * * *
	if isInt(min) && strings.Contains(hour, ",") {
		m, _ := strconv.Atoi(min)
		hours := strings.Split(hour, ",")
		formatted := make([]string, 0, len(hours))
		for _, h := range hours {
			h = strings.TrimSpace(h)
			if !isInt(h) {
				return ""
			}
			n, _ := strconv.Atoi(h)
			formatted = append(formatted, fmt.Sprintf("%02d:%02d", n, m))
		}
		if len(formatted) == 2 {
			return "at " + formatted[0] + " and " + formatted[1]
		}
		return "at " + strings.Join(formatted[:len(formatted)-1], ", ") + ", and " + formatted[len(formatted)-1]
	}

	return ""
}

func buildDOMDesc(dom string) string {
	if isInt(dom) {
		return fmt.Sprintf("on day %s of the month", dom)
	}
	if step, ok := parseStep(dom); ok {
		return fmt.Sprintf("every %d days", step)
	}
	if strings.Contains(dom, "-") {
		parts := strings.SplitN(dom, "-", 2)
		if isInt(parts[0]) && isInt(parts[1]) {
			return fmt.Sprintf("on days %s through %s of the month", parts[0], parts[1])
		}
	}
	return ""
}

// describeField resolves a single-value, range (a-b), or comma list against a name map.
func describeField(field string, names map[string]string) string {
	if name, ok := names[field]; ok {
		return name
	}
	if strings.Contains(field, "-") {
		i := strings.Index(field, "-")
		a, b := field[:i], field[i+1:]
		nameA, okA := names[a]
		nameB, okB := names[b]
		if okA && okB {
			return nameA + " through " + nameB
		}
		return ""
	}
	if strings.Contains(field, ",") {
		parts := strings.Split(field, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			name, ok := names[strings.TrimSpace(p)]
			if !ok {
				return ""
			}
			result = append(result, name)
		}
		if len(result) == 2 {
			return result[0] + " and " + result[1]
		}
		return strings.Join(result[:len(result)-1], ", ") + ", and " + result[len(result)-1]
	}
	return ""
}

func parseStep(field string) (int, bool) {
	if !strings.HasPrefix(field, "*/") {
		return 0, false
	}
	n, err := strconv.Atoi(field[2:])
	if err != nil || n < 1 {
		return 0, false
	}
	return n, true
}

func isInt(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
