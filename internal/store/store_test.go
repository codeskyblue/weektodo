package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/todo"
)

func testWeek(t *testing.T) *model.Week {
	t.Helper()
	week, err := todo.NewWeek("2026-W45", time.Date(2026, 11, 5, 9, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	week.Tasks = []*model.Task{
		{ID: "a", Text: "today task", Status: model.StatusTodo, Day: "2026-11-05", Children: []*model.Task{}},
		{ID: "b", Text: "done task", Status: model.StatusDone, Children: []*model.Task{}},
	}
	return week
}

func TestSaveLoadAndBackup(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SaveWeek(testWeek(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SaveWeek(testWeek(t)); err != nil {
		t.Fatal(err)
	}

	dataPath := filepath.Join(dir, dataFileName)
	info, err := os.Stat(dataPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("data mode = %o", info.Mode().Perm())
	}
	backups, err := os.ReadDir(filepath.Join(dir, backupDirName))
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) == 0 {
		t.Fatal("expected a daily backup")
	}

	reloaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	week, ok := reloaded.Week("2026-W45")
	if !ok || len(week.Tasks) != 2 {
		t.Fatalf("reloaded week = %+v", week)
	}
	if got := reloaded.TodayCount(time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)); got != 1 {
		t.Fatalf("today count = %d", got)
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	settings := model.Settings{AI: model.AIConfig{
		BaseURL: " https://api.example.com/v1/ ",
		Model:   " test-model ",
		APIKey:  " secret-key ",
	}}
	if err := st.UpdateSettings(settings); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, settingsFileName))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("settings mode = %o", info.Mode().Perm())
	}
	reloaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Settings().AI
	if got.BaseURL != "https://api.example.com/v1" || got.Model != "test-model" || got.APIKey != "secret-key" {
		t.Fatalf("settings = %+v", got)
	}
}

func TestRecoverFromBackup(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SaveWeek(testWeek(t)); err != nil {
		t.Fatal(err)
	}
	backupDir := filepath.Join(dir, backupDirName)
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, dataFileName))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "data-2999-01-01.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, dataFileName), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	reloaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reloaded.Week("2026-W45"); !ok {
		t.Fatal("expected recovery from backup")
	}
}

func TestListWeeksSorted(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"2026-W46", "2026-W44"} {
		week, err := todo.NewWeek(id, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		week.Tasks = []*model.Task{{ID: id + "-a", Text: id, Status: model.StatusTodo, Children: []*model.Task{}}}
		if _, err := st.SaveWeek(week); err != nil {
			t.Fatal(err)
		}
	}
	weeks := st.ListWeeks()
	if len(weeks) != 2 || weeks[0].ID != "2026-W44" || weeks[1].ID != "2026-W46" {
		t.Fatalf("weeks = %+v", weeks)
	}
}
