package todo

import (
	"testing"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

func fixedNow() time.Time {
	return time.Date(2026, 11, 5, 10, 0, 0, 0, time.Local)
}

func task(id, text string, status model.Status, children ...*model.Task) *model.Task {
	return &model.Task{ID: id, Text: text, Status: status, Children: children}
}

func TestNormalizeDerivesParentStatus(t *testing.T) {
	now := fixedNow()
	week := &model.Week{ID: "2026-W45", Tasks: []*model.Task{
		task("p", "  parent  ", model.StatusTodo,
			task("a", "a", model.StatusDone),
			task("b", "b", model.StatusDone),
		),
		task("q", "q", model.StatusDone,
			task("c", "c", model.StatusDone),
			task("d", "d", model.StatusTodo),
		),
	}}
	if err := NormalizeWeek(week, now); err != nil {
		t.Fatal(err)
	}
	if week.Start != "2026-11-02" || week.End != "2026-11-08" {
		t.Fatalf("range = %s..%s", week.Start, week.End)
	}
	if week.Tasks[0].Status != model.StatusDone {
		t.Fatalf("parent should be done, got %s", week.Tasks[0].Status)
	}
	if week.Tasks[0].Text != "parent" {
		t.Fatalf("text = %q", week.Tasks[0].Text)
	}
	if week.Tasks[1].Status != model.StatusTodo {
		t.Fatalf("parent with todo child should be todo, got %s", week.Tasks[1].Status)
	}
}

func TestNormalizeRejectsDeepTree(t *testing.T) {
	var children []*model.Task
	current := task("leaf", "leaf", model.StatusTodo)
	for i := 0; i < MaxDepth; i++ {
		children = []*model.Task{current}
		current = task("", "level", model.StatusTodo, children...)
	}
	week := &model.Week{ID: "2026-W45", Tasks: []*model.Task{current}}
	if err := NormalizeWeek(week, fixedNow()); err == nil {
		t.Fatal("expected depth error")
	}
}

func TestNormalizeClampsDayAndCascadesDrop(t *testing.T) {
	week := &model.Week{ID: "2026-W45", Tasks: []*model.Task{
		{ID: "x", Text: "x", Status: model.StatusTodo, Day: "2026-12-01", DroppedAt: "2026-11-04T10:00:00+08:00"},
		{ID: "y", Text: "y", Status: model.StatusDropped, Children: []*model.Task{
			{ID: "z", Text: "z", Status: model.StatusTodo},
		}},
	}}
	if err := NormalizeWeek(week, fixedNow()); err != nil {
		t.Fatal(err)
	}
	if week.Tasks[0].Day != "" {
		t.Fatalf("day should be cleared, got %q", week.Tasks[0].Day)
	}
	if week.Tasks[0].DroppedAt != "" {
		t.Fatalf("droppedAt should be cleared, got %q", week.Tasks[0].DroppedAt)
	}
	if week.Tasks[1].Status != model.StatusDropped || week.Tasks[1].Children[0].Status != model.StatusDropped {
		t.Fatal("drop should cascade")
	}
}

func TestStatsAndTodayCount(t *testing.T) {
	week := &model.Week{ID: "2026-W45", Tasks: []*model.Task{
		task("p", "p", model.StatusTodo,
			&model.Task{ID: "a", Text: "a", Status: model.StatusDone, Day: "2026-11-05"},
			&model.Task{ID: "b", Text: "b", Status: model.StatusTodo, Day: "2026-11-05"},
		),
		&model.Task{ID: "c", Text: "c", Status: model.StatusTodo},
		&model.Task{ID: "d", Text: "d", Status: model.StatusDropped},
	}}
	if err := NormalizeWeek(week, fixedNow()); err != nil {
		t.Fatal(err)
	}
	stats := WeekStats(week)
	if stats.Total != 3 || stats.Done != 1 || stats.Dropped != 1 || stats.Unplanned != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	if got := TodayCount(week, "2026-11-05"); got != 1 {
		t.Fatalf("today count = %d", got)
	}
}
