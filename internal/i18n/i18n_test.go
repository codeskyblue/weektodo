package i18n

import (
	"testing"
	"time"
)

func TestNormalizeAndTranslate(t *testing.T) {
	cases := map[string]Language{
		"zh-CN": Zh,
		"zh":    Zh,
		"en-US": En,
		"en":    En,
		"fr":    "",
		"":      "",
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
	if T(Zh, "中文", "English") != "中文" || T(En, "中文", "English") != "English" {
		t.Fatal("T returned unexpected results")
	}
	if got := TranslateError(En, "这一周还没有记录"); got != "There is no record for this week" {
		t.Fatalf("TranslateError = %q", got)
	}
	if got := TranslateError(Zh, "这一周还没有记录"); got != "这一周还没有记录" {
		t.Fatalf("TranslateError zh = %q", got)
	}
}

func TestDates(t *testing.T) {
	date := time.Date(2026, 11, 2, 0, 0, 0, 0, time.Local)
	if got := Weekday(Zh, date); got != "周一" {
		t.Fatalf("Weekday zh = %q", got)
	}
	if got := Weekday(En, date); got != "Mon" {
		t.Fatalf("Weekday en = %q", got)
	}
	if got := MonthDay(Zh, date); got != "11月2日" {
		t.Fatalf("MonthDay zh = %q", got)
	}
	if got := MonthDay(En, date); got != "Nov 2" {
		t.Fatalf("MonthDay en = %q", got)
	}
}
