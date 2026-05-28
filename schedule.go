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

	// DOM and DOW use OR logic in cron — show both when both are set.
	domDesc := ""
	if domF != "*" {
		domDesc = buildDOMDesc(domF)
	}
	dowDesc := ""
	if dowF != "*" {
		dowDesc = describeField(dowF, dowName)
	}

	switch {
	case domDesc != "" && dowDesc != "":
		segments = append(segments, "on "+domDesc+" and on "+dowDesc)
	case domDesc != "":
		segments = append(segments, "on "+domDesc)
	case dowDesc != "":
		segments = append(segments, "on "+dowDesc)
	}

	if monthF != "*" {
		if m := describeField(monthF, monthName); m != "" {
			segments = append(segments, "in "+m)
		}
	}

	if len(segments) == 0 {
		return ""
	}

	result := strings.Join(segments, " ")
	return strings.ToUpper(result[:1]) + result[1:] + "."
}

func buildTimeDesc(min, hour string) string {
	minStep, minIsStep := parseStep(min)
	hourStep, hourIsStep := parseStep(hour)
	minIsWild := min == "*"
	hourIsWild := hour == "*"
	minIsInt := isInt(min)
	hourIsInt := isInt(hour)

	switch {
	// * * → every minute
	case minIsWild && hourIsWild:
		return "every minute"

	// */N * → every N minutes
	case minIsStep && hourIsWild:
		if minStep == 1 {
			return "every minute"
		}
		return fmt.Sprintf("every %d minutes", minStep)

	// */N HH → at every Nth minute past hour HH
	case minIsStep && hourIsInt:
		h, _ := strconv.Atoi(hour)
		return fmt.Sprintf("at every %s minute past hour %02d", ordinal(minStep), h)

	// MM HH → at HH:MM
	case minIsInt && hourIsInt:
		h, _ := strconv.Atoi(hour)
		m, _ := strconv.Atoi(min)
		return fmt.Sprintf("at %02d:%02d", h, m)

	// 0 */N → every N hours
	case (min == "0" || min == "00") && hourIsStep:
		if hourStep == 1 {
			return "every hour"
		}
		return fmt.Sprintf("every %d hours", hourStep)

	// 0 * → every hour
	case (min == "0" || min == "00") && hourIsWild:
		return "every hour"

	// MM * → at minute MM past every hour
	case minIsInt && hourIsWild:
		m, _ := strconv.Atoi(min)
		return fmt.Sprintf("at minute %d past every hour", m)

	// * HH → every minute past hour HH
	case minIsWild && hourIsInt:
		h, _ := strconv.Atoi(hour)
		return fmt.Sprintf("every minute past hour %02d", h)

	// MM */N → at minute MM past every Nth hour
	case minIsInt && hourIsStep:
		m, _ := strconv.Atoi(min)
		return fmt.Sprintf("at minute %d past every %s hour", m, ordinal(hourStep))

	// MM HH,HH,... → at HH:MM and HH:MM
	case minIsInt && strings.Contains(hour, ","):
		m, _ := strconv.Atoi(min)
		hourParts := strings.Split(hour, ",")
		times := make([]string, 0, len(hourParts))
		for _, h := range hourParts {
			h = strings.TrimSpace(h)
			if !isInt(h) {
				return ""
			}
			n, _ := strconv.Atoi(h)
			times = append(times, fmt.Sprintf("%02d:%02d", n, m))
		}
		if len(times) == 2 {
			return "at " + times[0] + " and " + times[1]
		}
		return "at " + strings.Join(times[:len(times)-1], ", ") + ", and " + times[len(times)-1]
	}

	return ""
}

func buildDOMDesc(dom string) string {
	if isInt(dom) {
		return fmt.Sprintf("day %s of the month", dom)
	}
	if step, ok := parseStep(dom); ok {
		return fmt.Sprintf("every %s day-of-month", ordinal(step))
	}
	if strings.Contains(dom, "-") {
		parts := strings.SplitN(dom, "-", 2)
		if isInt(parts[0]) && isInt(parts[1]) {
			return fmt.Sprintf("every day-of-month from %s through %s", parts[0], parts[1])
		}
	}
	if strings.Contains(dom, ",") {
		items := strings.Split(dom, ",")
		strs := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if !isInt(item) {
				return ""
			}
			strs = append(strs, item)
		}
		if len(strs) == 2 {
			return fmt.Sprintf("days %s and %s of the month", strs[0], strs[1])
		}
		return "days " + strings.Join(strs[:len(strs)-1], ", ") + ", and " + strs[len(strs)-1] + " of the month"
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

func ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 13 {
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
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
