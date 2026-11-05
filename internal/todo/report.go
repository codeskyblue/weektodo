package todo

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

// BuildReport renders a Markdown weekly report. mode is "brief" or "full".
func BuildReport(w *model.Week, mode string) (string, model.Stats) {
	stats := WeekStats(w)
	if mode != "full" {
		mode = "brief"
	}
	var b strings.Builder
	start, _ := ParseDate(w.Start)
	end, _ := ParseDate(w.End)
	fmt.Fprintf(&b, "# 周报｜%s ~ %s\n\n", w.Start, w.End)
	fmt.Fprintf(&b, "本周完成 %d 项，废弃 %d 项，未完成 %d 项。\n\n", stats.Done, stats.Dropped, max(0, stats.Total-stats.Done))
	if len(w.Tasks) == 0 {
		b.WriteString("本周还没有任务。\n")
		return b.String(), stats
	}
	if stats.Done == 0 {
		b.WriteString("## 已完成\n\n本周还没有完成的任务。\n")
		return b.String(), stats
	}
	if mode == "full" {
		b.WriteString("## 已完成\n\n")
		for _, task := range topDoneTasks(w.Tasks) {
			writeFullTask(&b, task, 0)
		}
		return b.String(), stats
	}

	groups := map[string][]*model.Task{}
	for _, task := range topDoneTasks(w.Tasks) {
		day := completionDay(task, start.Unix(), end.Unix())
		groups[day] = append(groups[day], task)
	}
	days := make([]string, 0, len(groups))
	for day := range groups {
		days = append(days, day)
	}
	sort.Strings(days)
	for _, day := range days {
		date, err := ParseDate(day)
		if err != nil {
			b.WriteString("## 本周\n\n")
		} else {
			fmt.Fprintf(&b, "## %s · %s\n\n", WeekdayCN(date), monthDay(date))
		}
		for _, task := range groups[day] {
			line := taskLabel(task)
			if children := countDoneDescendants(task); children > 0 {
				line += fmt.Sprintf("（含 %d 项子任务）", children)
			}
			fmt.Fprintf(&b, "- %s\n", line)
		}
		b.WriteByte('\n')
	}
	return b.String(), stats
}

func topDoneTasks(tasks []*model.Task) []*model.Task {
	out := []*model.Task{}
	var walk func(tasks []*model.Task)
	walk = func(tasks []*model.Task) {
		for _, task := range tasks {
			if task.Status == model.StatusDone {
				out = append(out, task)
				continue
			}
			if task.Status == model.StatusDropped {
				continue
			}
			walk(task.Children)
		}
	}
	walk(tasks)
	return out
}

func writeFullTask(b *strings.Builder, task *model.Task, depth int) {
	if task.Status != model.StatusDone {
		return
	}
	b.WriteString(strings.Repeat("  ", depth))
	fmt.Fprintf(b, "- %s\n", taskLabel(task))
	for _, child := range task.Children {
		writeFullTask(b, child, depth+1)
	}
}

func taskLabel(task *model.Task) string {
	text := strings.TrimSpace(task.Text)
	if text == "" {
		return "（无标题任务）"
	}
	return text
}

func countDoneDescendants(task *model.Task) int {
	count := 0
	var walk func(tasks []*model.Task)
	walk = func(tasks []*model.Task) {
		for _, child := range tasks {
			if child.Status != model.StatusDone {
				continue
			}
			count++
			walk(child.Children)
		}
	}
	walk(task.Children)
	return count
}

func completionDay(task *model.Task, startUnix, endUnix int64) string {
	if task.CompletedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, task.CompletedAt); err == nil {
			local := parsed.In(time.Local)
			if local.Unix() >= startUnix && local.Unix() < endUnix+86400 {
				return DateString(local)
			}
		}
	}
	if task.Day != "" {
		return task.Day
	}
	return DateString(time.Unix(startUnix, 0))
}

func monthDay(t time.Time) string {
	return fmt.Sprintf("%d月%d日", int(t.Month()), t.Day())
}
