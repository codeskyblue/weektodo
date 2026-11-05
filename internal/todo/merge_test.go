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

func TestMergeDataKeepsCategoriesSeparate(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)
	local := model.Data{Version: 2,
		Categories: []*model.Category{{ID: "work", Name: "工作"}},
		Weeks: []*model.Week{{
			ID: "2026-W45", Category: "work", Tasks: []*model.Task{{ID: "work-1", Text: "工作", Status: model.StatusTodo, Children: []*model.Task{}}},
		}},
	}
	remote := model.Data{Version: 2,
		Categories: []*model.Category{{ID: "oss", Name: "开源"}},
		Weeks: []*model.Week{{
			ID: "2026-W45", Category: "oss", Tasks: []*model.Task{{ID: "oss-1", Text: "开源", Status: model.StatusTodo, Children: []*model.Task{}}},
		}},
	}
	merged, stats, err := MergeData(local, remote, now)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Weeks != 2 || stats.Tasks != 2 || stats.Categories != 3 {
		t.Fatalf("stats = %+v", stats)
	}
	byKey := map[string]*model.Week{}
	for _, week := range merged.Weeks {
		byKey[week.Category+"/"+week.ID] = week
	}
	if week := byKey["work/2026-W45"]; week == nil || len(week.Tasks) != 1 || week.Tasks[0].ID != "work-1" {
		t.Fatalf("work week = %+v", week)
	}
	if week := byKey["oss/2026-W45"]; week == nil || len(week.Tasks) != 1 || week.Tasks[0].ID != "oss-1" {
		t.Fatalf("oss week = %+v", week)
	}
	if len(merged.Weeks) != 2 {
		t.Fatalf("same week id in two categories must stay separate: %+v", merged.Weeks)
	}
}

func TestMergeCategoryMetadataUsesNewest(t *testing.T) {
	now := time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local)
	local := model.Data{Version: 2, Categories: []*model.Category{{
		ID: "work", Name: "工作", Color: "#111111", UpdatedAt: "2026-11-05T10:00:00+08:00",
	}}}
	remote := model.Data{Version: 2, Categories: []*model.Category{{
		ID: "work", Name: "Work", Color: "#222222", Archived: true, UpdatedAt: "2026-11-05T11:00:00+08:00",
	}}}
	merged, _, err := MergeData(local, remote, now)
	if err != nil {
		t.Fatal(err)
	}
	var work *model.Category
	for _, category := range merged.Categories {
		if category.ID == "work" {
			work = category
		}
	}
	if work == nil || work.Name != "Work" || work.Color != "#222222" || !work.Archived {
		t.Fatalf("merged category = %+v", work)
	}
}
