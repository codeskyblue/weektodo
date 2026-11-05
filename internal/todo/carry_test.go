package todo

import (
	"testing"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

func carryFixture(t *testing.T) (*model.Week, *model.Week) {
	t.Helper()
	now := time.Date(2026, 10, 29, 10, 0, 0, 0, time.Local)
	source := &model.Week{ID: "2026-W44", Tasks: []*model.Task{
		&model.Task{ID: "p", Text: "parent", Status: model.StatusTodo, Day: "2026-10-29", Children: []*model.Task{
			&model.Task{ID: "a", Text: "done child", Status: model.StatusDone},
			&model.Task{ID: "b", Text: "todo child", Status: model.StatusTodo, Day: "2026-10-30"},
		}},
		task("d", "dropped", model.StatusDropped),
		task("s", "single", model.StatusTodo),
	}}
	if err := NormalizeWeek(source, now); err != nil {
		t.Fatal(err)
	}
	target := &model.Week{ID: "2026-W45", Tasks: []*model.Task{}}
	if err := NormalizeWeek(target, now); err != nil {
		t.Fatal(err)
	}
	return source, target
}

func TestCarryTasksPreservesUnfinishedHierarchy(t *testing.T) {
	source, target := carryFixture(t)
	copied, count, skipped := CarryTasks(source, target, fixedNow())
	if count != 3 || skipped != 0 {
		t.Fatalf("count=%d skipped=%d", count, skipped)
	}
	if len(copied) != 2 {
		t.Fatalf("copied roots = %d", len(copied))
	}
	parent := copied[0]
	if parent.Text != "parent" || parent.Day != "" {
		t.Fatalf("parent = %+v", parent)
	}
	if parent.CarriedFrom == nil || parent.CarriedFrom.TaskID != "p" || parent.CarriedFrom.WeekID != "2026-W44" {
		t.Fatalf("carriedFrom = %+v", parent.CarriedFrom)
	}
	if len(parent.Children) != 1 || parent.Children[0].Text != "todo child" {
		t.Fatalf("children = %+v", parent.Children)
	}
	if parent.Children[0].Day != "" {
		t.Fatal("carried day should be cleared")
	}
	if copied[1].Text != "single" {
		t.Fatalf("second root = %+v", copied[1])
	}
}

func TestCarryIsIdempotent(t *testing.T) {
	source, target := carryFixture(t)
	first, count, _ := CarryTasks(source, target, fixedNow())
	if count != 3 {
		t.Fatalf("count = %d", count)
	}
	target.Tasks = append(target.Tasks, first...)
	if err := NormalizeWeek(target, fixedNow()); err != nil {
		t.Fatal(err)
	}
	preview := PreviewCarry(source, target)
	if preview.Count != 0 || preview.Skipped != 3 {
		t.Fatalf("preview = %+v", preview)
	}
	second, count, skipped := CarryTasks(source, target, fixedNow())
	if len(second) != 0 || count != 0 || skipped != 3 {
		t.Fatalf("second=%d count=%d skipped=%d", len(second), count, skipped)
	}
}
