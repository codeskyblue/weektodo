package store

import (
	"encoding/json"
	"errors"
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

func TestDefaultDirUsesConfigHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("WEEKTODO_DATA_DIR", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	want := filepath.Join(home, ".config", "weektodo")
	if got := DefaultDir(); got != want {
		t.Fatalf("DefaultDir = %q, want %q", got, want)
	}

	xdg := filepath.Join(home, "xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	if got, want := DefaultDir(), filepath.Join(xdg, "weektodo"); got != want {
		t.Fatalf("DefaultDir with XDG = %q, want %q", got, want)
	}

	t.Setenv("WEEKTODO_DATA_DIR", filepath.Join(home, "custom"))
	if got, want := DefaultDir(), filepath.Join(home, "custom"); got != want {
		t.Fatalf("DefaultDir with override = %q, want %q", got, want)
	}
}

func TestMigrateLegacyDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("WEEKTODO_DATA_DIR", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	legacy := filepath.Join(home, ".weektodo-dev")
	if err := os.MkdirAll(filepath.Join(legacy, backupDirName), 0o700); err != nil {
		t.Fatal(err)
	}
	data := `{"version":1,"weeks":[{"id":"2026-W45","start":"2026-11-02","end":"2026-11-08","tasks":[{"id":"a","text":"legacy task","status":"todo","children":[]}]}]}`
	if err := os.WriteFile(filepath.Join(legacy, dataFileName), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	settings := `{"ai":{"baseUrl":"https://example.com/v1","model":"m","apiKey":"secret"},"language":"en"}`
	if err := os.WriteFile(filepath.Join(legacy, settingsFileName), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, backupDirName, "data-2026-11-05.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "weektodo")
	if st.Dir() != want {
		t.Fatalf("Dir = %q, want %q", st.Dir(), want)
	}
	if week, ok := st.Week("2026-W45"); !ok || week.Tasks[0].Text != "legacy task" {
		t.Fatalf("week = %+v, %v", week, ok)
	}
	if st.Settings().Language != "en" || st.Settings().AI.APIKey != "secret" {
		t.Fatalf("settings = %+v", st.Settings())
	}
	if _, err := os.Stat(filepath.Join(want, backupDirName, "data-2026-11-05.json")); err != nil {
		t.Fatalf("backup not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacy, dataFileName)); err != nil {
		t.Fatalf("legacy data must be kept: %v", err)
	}
}

func TestExplicitDirDoesNotMigrate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("WEEKTODO_DATA_DIR", "")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	legacy := filepath.Join(home, ".weektodo-dev")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}
	data := `{"version":1,"weeks":[]}`
	if err := os.WriteFile(filepath.Join(legacy, dataFileName), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(st.ListWeeks()) != 0 {
		t.Fatalf("explicit dir should not inherit legacy data")
	}
}

func TestMoveTaskAcrossWeeks(t *testing.T) {
	dir := t.TempDir()
	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)
	source, err := todo.NewWeek("2026-W45", now)
	if err != nil {
		t.Fatal(err)
	}
	source.Tasks = []*model.Task{{
		ID:          "parent",
		Text:        "parent",
		Status:      model.StatusTodo,
		Day:         "2026-11-05",
		UpdatedAt:   now.Add(-time.Hour).Format(time.RFC3339),
		CarriedFrom: &model.CarriedFrom{WeekID: "2026-W44", TaskID: "old"},
		Children: []*model.Task{{
			ID:       "child",
			Text:     "child",
			Status:   model.StatusTodo,
			Day:      "2026-11-06",
			Children: []*model.Task{},
		}},
	}}
	if _, err := st.SaveWeek(source); err != nil {
		t.Fatal(err)
	}

	from, to, err := st.MoveTask("2026-W45", "parent", "2026-W46", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(from.Tasks) != 0 {
		t.Fatalf("source tasks = %+v", from.Tasks)
	}
	if to.ID != "2026-W46" || to.Start != "2026-11-09" || len(to.Tasks) != 1 {
		t.Fatalf("target = %+v", to)
	}
	moved := to.Tasks[0]
	if moved.ID != "parent" || moved.Day != "" || moved.CarriedFrom != nil {
		t.Fatalf("moved = %+v", moved)
	}
	if moved.UpdatedAt != now.Format(time.RFC3339) {
		t.Fatalf("updatedAt = %q", moved.UpdatedAt)
	}
	if len(moved.Children) != 1 || moved.Children[0].Day != "" || moved.Children[0].UpdatedAt != now.Format(time.RFC3339) {
		t.Fatalf("children = %+v", moved.Children)
	}

	reloaded, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	savedTarget, ok := reloaded.Week("2026-W46")
	if !ok || len(savedTarget.Tasks) != 1 || savedTarget.Tasks[0].ID != "parent" {
		t.Fatalf("reloaded target = %+v, %v", savedTarget, ok)
	}
	savedSource, ok := reloaded.Week("2026-W45")
	if !ok || len(savedSource.Tasks) != 0 {
		t.Fatalf("reloaded source = %+v, %v", savedSource, ok)
	}
}

func TestMoveTaskErrors(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	week, err := todo.NewWeek("2026-W45", now)
	if err != nil {
		t.Fatal(err)
	}
	week.Tasks = []*model.Task{{ID: "a", Text: "a", Status: model.StatusTodo, Children: []*model.Task{}}}
	if _, err := st.SaveWeek(week); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.MoveTask("2026-W45", "a", "2026-W45", now); !errors.Is(err, ErrSameWeek) {
		t.Fatalf("same week err = %v", err)
	}
	if _, _, err := st.MoveTask("2026-W44", "a", "2026-W46", now); !errors.Is(err, ErrWeekNotFound) {
		t.Fatalf("missing week err = %v", err)
	}
	if _, _, err := st.MoveTask("2026-W45", "missing", "2026-W46", now); !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("missing task err = %v", err)
	}
	if _, ok := st.Week("2026-W46"); ok {
		t.Fatal("failed move must not create an empty target week")
	}
}

func TestCategoryIsolation(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	categories := st.Categories()
	if len(categories) != 1 || categories[0].ID != model.DefaultCategoryID {
		t.Fatalf("default categories = %+v", categories)
	}
	oss, err := st.SaveCategory(model.Category{Name: "开源", Color: "#8b5cf6"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SaveCategory(model.Category{Name: "开源"}); !errors.Is(err, ErrCategoryNameTaken) {
		t.Fatalf("duplicate name err = %v", err)
	}

	now := time.Now()
	work := &model.Week{ID: "2026-W45", Tasks: []*model.Task{{ID: "work-task", Text: "工作", Status: model.StatusTodo, Children: []*model.Task{}}}}
	side := &model.Week{ID: "2026-W45", Tasks: []*model.Task{{ID: "oss-task", Text: "开源", Status: model.StatusTodo, Children: []*model.Task{}}}}
	if _, err := st.SaveWeekInCategory(model.DefaultCategoryID, work); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SaveWeekInCategory(oss.ID, side); err != nil {
		t.Fatal(err)
	}

	workWeek, ok := st.WeekInCategory(model.DefaultCategoryID, "2026-W45")
	if !ok || len(workWeek.Tasks) != 1 || workWeek.Tasks[0].ID != "work-task" {
		t.Fatalf("work week = %+v, %v", workWeek, ok)
	}
	sideWeek, ok := st.WeekInCategory(oss.ID, "2026-W45")
	if !ok || len(sideWeek.Tasks) != 1 || sideWeek.Tasks[0].ID != "oss-task" {
		t.Fatalf("oss week = %+v, %v", sideWeek, ok)
	}
	if got := st.ListWeeksInCategory(oss.ID); len(got) != 1 {
		t.Fatalf("oss summaries = %+v", got)
	}

	if _, _, err := st.MoveTaskInCategory(oss.ID, "2026-W45", "oss-task", "2026-W46", now); err != nil {
		t.Fatal(err)
	}
	if week, ok := st.WeekInCategory(oss.ID, "2026-W46"); !ok || len(week.Tasks) != 1 {
		t.Fatalf("oss target = %+v, %v", week, ok)
	}
	if week, ok := st.WeekInCategory(model.DefaultCategoryID, "2026-W46"); ok {
		t.Fatalf("default category must not gain the moved task: %+v", week)
	}
	if week, _ := st.WeekInCategory(model.DefaultCategoryID, "2026-W45"); len(week.Tasks) != 1 {
		t.Fatalf("default source changed: %+v", week)
	}

	if err := st.DeleteWeekInCategory(oss.ID, "2026-W45"); err != nil {
		t.Fatal(err)
	}
	if _, ok := st.WeekInCategory(oss.ID, "2026-W45"); ok {
		t.Fatal("oss week was not deleted")
	}
	if _, ok := st.WeekInCategory(model.DefaultCategoryID, "2026-W45"); !ok {
		t.Fatal("deleting an oss week must not touch the default category")
	}
}

func TestCategoryMigration(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"version":1,"weeks":[{"id":"2026-W45","start":"2026-11-02","end":"2026-11-08","tasks":[{"id":"a","text":"legacy","status":"todo","children":[]}]}]}`
	if err := os.WriteFile(filepath.Join(dir, dataFileName), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	settings := `{"language":"en"}`
	if err := os.WriteFile(filepath.Join(dir, settingsFileName), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}

	st, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	categories := st.Categories()
	if len(categories) != 1 || categories[0].ID != model.DefaultCategoryID || categories[0].Name != "Inbox" {
		t.Fatalf("migrated categories = %+v", categories)
	}
	week, ok := st.WeekInCategory(model.DefaultCategoryID, "2026-W45")
	if !ok || len(week.Tasks) != 1 || week.Tasks[0].Text != "legacy" {
		t.Fatalf("migrated week = %+v, %v", week, ok)
	}

	raw, err := os.ReadFile(filepath.Join(dir, dataFileName))
	if err != nil {
		t.Fatal(err)
	}
	var document model.Data
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	if document.Version != 2 || len(document.Categories) != 1 || document.Weeks[0].Category != model.DefaultCategoryID {
		t.Fatalf("persisted migration = %+v", document)
	}
}

func TestMoveTaskUnderParent(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)
	week, err := todo.NewWeek("2026-W45", now)
	if err != nil {
		t.Fatal(err)
	}
	week.Tasks = []*model.Task{
		{ID: "parent", Text: "parent", Status: model.StatusTodo, Day: "2026-11-05", Children: []*model.Task{
			{ID: "child", Text: "child", Status: model.StatusTodo, Day: "2026-11-06", Children: []*model.Task{}},
		}},
		{ID: "target", Text: "target", Status: model.StatusTodo, Children: []*model.Task{}},
		{ID: "dropped", Text: "dropped", Status: model.StatusDropped, Children: []*model.Task{}},
	}
	if _, err := st.SaveWeekInCategory(model.DefaultCategoryID, week); err != nil {
		t.Fatal(err)
	}

	saved, _, err := st.MoveTaskUnder(model.DefaultCategoryID, "2026-W45", "parent", "2026-W45", "target", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Tasks) != 2 {
		t.Fatalf("root tasks = %+v", saved.Tasks)
	}
	target := saved.Tasks[0]
	if target.ID != "target" || len(target.Children) != 1 || target.Children[0].ID != "parent" {
		t.Fatalf("target = %+v", target)
	}
	if target.Children[0].Day != "2026-11-05" {
		t.Fatalf("same-week reparent must keep the date: %+v", target.Children[0])
	}

	if _, _, err := st.MoveTaskUnder(model.DefaultCategoryID, "2026-W45", "parent", "2026-W45", "child", now); !errors.Is(err, ErrMoveCycle) {
		t.Fatalf("cycle err = %v", err)
	}
	if _, _, err := st.MoveTaskUnder(model.DefaultCategoryID, "2026-W45", "parent", "2026-W45", "dropped", now); !errors.Is(err, ErrParentTaskDropped) {
		t.Fatalf("dropped parent err = %v", err)
	}

	targetWeek, sourceWeek, err := st.MoveTaskUnder(model.DefaultCategoryID, "2026-W45", "parent", "2026-W46", "target2", now)
	if err == nil {
		t.Fatalf("expected missing parent error, got %+v %+v", targetWeek, sourceWeek)
	}
	if !errors.Is(err, ErrParentTaskNotFound) {
		t.Fatalf("missing parent err = %v", err)
	}
}

func TestMoveTaskUnderParentCrossWeek(t *testing.T) {
	st, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)
	source, _ := todo.NewWeek("2026-W45", now)
	source.Tasks = []*model.Task{{ID: "moving", Text: "moving", Status: model.StatusTodo, Day: "2026-11-05", Children: []*model.Task{
		{ID: "child", Text: "child", Status: model.StatusTodo, Day: "2026-11-06", Children: []*model.Task{}},
	}}}
	target, _ := todo.NewWeek("2026-W46", now)
	target.Tasks = []*model.Task{{ID: "host", Text: "host", Status: model.StatusTodo, Children: []*model.Task{}}}
	if _, err := st.SaveWeekInCategory(model.DefaultCategoryID, source); err != nil {
		t.Fatal(err)
	}
	if _, err := st.SaveWeekInCategory(model.DefaultCategoryID, target); err != nil {
		t.Fatal(err)
	}

	from, to, err := st.MoveTaskUnder(model.DefaultCategoryID, "2026-W45", "moving", "2026-W46", "host", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(from.Tasks) != 0 {
		t.Fatalf("source = %+v", from.Tasks)
	}
	if len(to.Tasks) != 1 || len(to.Tasks[0].Children) != 1 {
		t.Fatalf("target = %+v", to.Tasks)
	}
	moved := to.Tasks[0].Children[0]
	if moved.ID != "moving" || moved.Day != "" || moved.Children[0].Day != "" {
		t.Fatalf("moved = %+v", moved)
	}
}
