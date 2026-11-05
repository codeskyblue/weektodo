package todo

import (
	"testing"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

func TestMergeDataUsesNewestTask(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)
	local := model.Data{Version: 1, Weeks: []*model.Week{{
		ID: "2026-W45", Start: "2026-11-02", End: "2026-11-08",
		Tasks: []*model.Task{
			{ID: "p", Text: "local parent", Status: model.StatusTodo, UpdatedAt: "2026-11-05T10:00:00+08:00", Children: []*model.Task{
				{ID: "a", Text: "local child", Status: model.StatusTodo, UpdatedAt: "2026-11-05T10:00:00+08:00", Children: []*model.Task{}},
			}},
		},
	}}}
	remote := model.Data{Version: 1, Weeks: []*model.Week{{
		ID: "2026-W45", Start: "2026-11-02", End: "2026-11-08",
		Tasks: []*model.Task{
			{ID: "p", Text: "remote parent", Status: model.StatusTodo, UpdatedAt: "2026-11-05T11:00:00+08:00", Children: []*model.Task{
				{ID: "a", Text: "remote child", Status: model.StatusDone, CompletedAt: "2026-11-05T11:00:00+08:00", UpdatedAt: "2026-11-05T11:00:00+08:00", Children: []*model.Task{}},
				{ID: "b", Text: "remote only", Status: model.StatusTodo, UpdatedAt: "2026-11-05T11:00:00+08:00", Children: []*model.Task{}},
			}},
		},
	}}}
	merged, stats, err := MergeData(local, remote, now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Weeks != 1 || stats.Tasks != 3 {
		t.Fatalf("stats = %+v", stats)
	}
	parent := merged.Weeks[0].Tasks[0]
	if parent.Text != "remote parent" {
		t.Fatalf("parent = %+v", parent)
	}
	if len(parent.Children) != 2 {
		t.Fatalf("children = %+v", parent.Children)
	}
	if parent.Children[0].ID != "a" || parent.Children[0].Text != "remote child" {
		t.Fatalf("child a = %+v", parent.Children[0])
	}
	if parent.Children[1].ID != "b" {
		t.Fatalf("child b = %+v", parent.Children[1])
	}
}

func TestMergeDataKeepsBothWeeks(t *testing.T) {
	now := time.Now()
	local := model.Data{Version: 1, Weeks: []*model.Week{{ID: "2026-W45", Start: "2026-11-02", End: "2026-11-08"}}}
	remote := model.Data{Version: 1, Weeks: []*model.Week{{ID: "2026-W44", Start: "2026-10-26", End: "2026-11-01"}}}
	merged, stats, err := MergeData(local, remote, now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Weeks != 2 || merged.Weeks[0].ID != "2026-W44" || merged.Weeks[1].ID != "2026-W45" {
		t.Fatalf("merged = %+v", merged.Weeks)
	}
}
