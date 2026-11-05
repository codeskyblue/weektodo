package todo

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var weekIDPattern = regexp.MustCompile(`^(\d{4})-W(\d{2})$`)

// ParseWeekID validates an ISO week id such as 2026-W45.
func ParseWeekID(id string) (int, int, error) {
	m := weekIDPattern.FindStringSubmatch(id)
	if m == nil {
		return 0, 0, fmt.Errorf("invalid week id %q", id)
	}
	year, _ := strconv.Atoi(m[1])
	week, _ := strconv.Atoi(m[2])
	if week < 1 || week > 53 {
		return 0, 0, fmt.Errorf("invalid week id %q", id)
	}
	return year, week, nil
}

// WeekRange returns the Monday and Sunday of an ISO week id.
func WeekRange(id string) (time.Time, time.Time, error) {
	year, week, err := ParseWeekID(id)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)
	offset := (int(jan4.Weekday()) + 6) % 7
	monday := jan4.AddDate(0, 0, -offset+(week-1)*7)
	if y, w := monday.ISOWeek(); y != year || w != week {
		return time.Time{}, time.Time{}, fmt.Errorf("week %q does not exist", id)
	}
	return monday, monday.AddDate(0, 0, 6), nil
}

// WeekIDFor returns the ISO week id for a date.
func WeekIDFor(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

// PreviousWeekID returns the ISO week id before id.
func PreviousWeekID(id string) string {
	start, _, err := WeekRange(id)
	if err != nil {
		return ""
	}
	return WeekIDFor(start.AddDate(0, 0, -7))
}

// NextWeekID returns the ISO week id after id.
func NextWeekID(id string) string {
	start, _, err := WeekRange(id)
	if err != nil {
		return ""
	}
	return WeekIDFor(start.AddDate(0, 0, 7))
}

// DateString formats a date as YYYY-MM-DD.
func DateString(t time.Time) string {
	return t.Format("2006-01-02")
}

// ParseDate parses YYYY-MM-DD in local time.
func ParseDate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, time.Local)
}
