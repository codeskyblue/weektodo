package todo

import (
	"strings"
	"testing"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

func reportFixture(t *testing.T) *model.Week {
	t.Helper()
	parent := &model.Task{
		ID: "p", Text: "重构登录页", Status: model.StatusDone,
		Children: []*model.Task{
			{ID: "a", Text: "拆分组件", Status: model.StatusDone, CompletedAt: "2026-11-03T09:00:00+08:00"},
			{ID: "b", Text: "接入接口", Status: model.StatusDone, CompletedAt: "2026-11-04T18:00:00+08:00"},
		},
	}
	week := &model.Week{ID: "2026-W45", Tasks: []*model.Task{
		parent,
		{ID: "s", Text: "写文档", Status: model.StatusDone, CompletedAt: "2026-11-03T11:00:00+08:00"},
		{ID: "d", Text: "废弃事项", Status: model.StatusDropped},
		{ID: "t", Text: "未完成事项", Status: model.StatusTodo},
	}}
	if err := NormalizeWeek(week, fixedNow()); err != nil {
		t.Fatal(err)
	}
	return week
}

func TestBuildBriefReport(t *testing.T) {
	week := reportFixture(t)
	markdown, stats := BuildReport(week, "brief")
	if stats.Done != 3 || stats.Total != 4 || stats.Dropped != 1 {
		t.Fatalf("stats = %+v", stats)
	}
	for _, want := range []string{
		"# 周报｜2026-11-02 ~ 2026-11-08",
		"本周完成 3 项，废弃 1 项，未完成 1 项。",
		"## 周二 · 11月3日",
		"- 写文档",
		"## 周三 · 11月4日",
		"- 重构登录页（含 2 项子任务）",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("report missing %q:\n%s", want, markdown)
		}
	}
	if strings.Contains(markdown, "拆分组件") {
		t.Fatalf("brief report should collapse children:\n%s", markdown)
	}
}

func TestBuildFullReport(t *testing.T) {
	week := reportFixture(t)
	markdown, _ := BuildReport(week, "full")
	for _, want := range []string{
		"- 重构登录页",
		"  - 拆分组件",
		"  - 接入接口",
		"- 写文档",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("full report missing %q:\n%s", want, markdown)
		}
	}
}

func TestReportEmptyWeek(t *testing.T) {
	week, err := NewWeek("2026-W45", fixedNow())
	if err != nil {
		t.Fatal(err)
	}
	markdown, _ := BuildReport(week, "brief")
	if !strings.Contains(markdown, "本周还没有任务") {
		t.Fatalf("unexpected report:\n%s", markdown)
	}
}

func TestNewWeek(t *testing.T) {
	week, err := NewWeek("2026-W45", time.Date(2026, 11, 5, 10, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if week.Start != "2026-11-02" || week.End != "2026-11-08" || week.Tasks == nil {
		t.Fatalf("week = %+v", week)
	}
}
