package todo

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
)

// BuildReport renders a Markdown weekly report. mode is "brief" or "full".
func BuildReport(w *model.Week, mode string, lang i18n.Language) (string, model.Stats) {
	stats := WeekStats(w)
	if mode != "full" {
		mode = "brief"
	}
	var b strings.Builder
	start, _ := ParseDate(w.Start)
	end, _ := ParseDate(w.End)
	if lang == i18n.En {
		fmt.Fprintf(&b, "# Weekly Report | %s ~ %s\n\n", w.Start, w.End)
		fmt.Fprintf(&b, "Completed %d, dropped %d, open %d this week.\n\n", stats.Done, stats.Dropped, max(0, stats.Total-stats.Done))
	} else {
		fmt.Fprintf(&b, "# 周报｜%s ~ %s\n\n", w.Start, w.End)
		fmt.Fprintf(&b, "本周完成 %d 项，废弃 %d 项，未完成 %d 项。\n\n", stats.Done, stats.Dropped, max(0, stats.Total-stats.Done))
	}
	if len(w.Tasks) == 0 {
		b.WriteString(i18n.T(lang, "本周还没有任务。", "No tasks this week.") + "\n")
		return b.String(), stats
	}
	if stats.Done == 0 {
		b.WriteString(i18n.T(lang, "## 已完成\n\n本周还没有完成的任务。\n", "## Completed\n\nNo completed tasks this week.\n"))
		return b.String(), stats
	}
	if mode == "full" {
		b.WriteString(i18n.T(lang, "## 已完成\n\n", "## Completed\n\n"))
		for _, task := range topDoneTasks(w.Tasks) {
			writeFullTask(&b, task, 0, lang)
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
			b.WriteString(i18n.T(lang, "## 本周\n\n", "## This week\n\n"))
		} else {
			fmt.Fprintf(&b, "## %s · %s\n\n", i18n.Weekday(lang, date), i18n.MonthDay(lang, date))
		}
		for _, task := range groups[day] {
			line := taskLabel(task, lang)
			if children := countDoneDescendants(task); children > 0 {
				line += i18n.T(lang, fmt.Sprintf("（含 %d 项子任务）", children), fmt.Sprintf(" (%d subtasks)", children))
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

func writeFullTask(b *strings.Builder, task *model.Task, depth int, lang i18n.Language) {
	if task.Status != model.StatusDone {
		return
	}
	b.WriteString(strings.Repeat("  ", depth))
	fmt.Fprintf(b, "- %s\n", taskLabel(task, lang))
	for _, child := range task.Children {
		writeFullTask(b, child, depth+1, lang)
	}
}

func taskLabel(task *model.Task, lang i18n.Language) string {
	text := strings.TrimSpace(task.Text)
	if text == "" {
		return i18n.T(lang, "（无标题任务）", "(Untitled task)")
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
