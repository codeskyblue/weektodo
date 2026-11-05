package todo

import "testing"

func TestWeekRangeAndIDs(t *testing.T) {
	start, end, err := WeekRange("2026-W45")
	if err != nil {
		t.Fatal(err)
	}
	if got := DateString(start); got != "2026-11-02" {
		t.Fatalf("start = %s", got)
	}
	if got := DateString(end); got != "2026-11-08" {
		t.Fatalf("end = %s", got)
	}
	if got := WeekIDFor(start); got != "2026-W45" {
		t.Fatalf("week id = %s", got)
	}
	if got := PreviousWeekID("2026-W45"); got != "2026-W44" {
		t.Fatalf("previous = %s", got)
	}
	if got := NextWeekID("2026-W45"); got != "2026-W46" {
		t.Fatalf("next = %s", got)
	}
	if _, _, err := WeekRange("2026-W54"); err == nil {
		t.Fatal("expected invalid week error")
	}
	if _, _, err := WeekRange("2026-45"); err == nil {
		t.Fatal("expected invalid format error")
	}
}

func TestWeekRangeAcrossYearBoundary(t *testing.T) {
	start, _, err := WeekRange("2027-W01")
	if err != nil {
		t.Fatal(err)
	}
	if got := DateString(start); got != "2027-01-04" {
		t.Fatalf("start = %s", got)
	}
	if got := WeekIDFor(start.AddDate(0, 0, -1)); got != "2026-W53" {
		t.Fatalf("previous year week = %s", got)
	}
}
