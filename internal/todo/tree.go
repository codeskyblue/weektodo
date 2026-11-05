package todo

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/uiautodev/weektodo/internal/model"
)

const (
	MaxDepth     = 8
	MaxTasks     = 2000
	MaxTextRunes = 500
)

// NewWeek creates an empty canonical week.
func NewWeek(id string, now time.Time) (*model.Week, error) {
	start, end, err := WeekRange(id)
	if err != nil {
		return nil, err
	}
	return &model.Week{
		ID:        id,
		Start:     DateString(start),
		End:       DateString(end),
		CreatedAt: now.Format(time.RFC3339),
		Tasks:     []*model.Task{},
	}, nil
}

// NormalizeWeek validates and canonicalizes a week's task tree.
func NormalizeWeek(w *model.Week, now time.Time) error {
	start, end, err := WeekRange(w.ID)
	if err != nil {
		return err
	}
	w.Start = DateString(start)
	w.End = DateString(end)
	if !validTimestamp(w.CreatedAt) {
		w.CreatedAt = now.Format(time.RFC3339)
	}
	if w.Tasks == nil {
		w.Tasks = []*model.Task{}
	}
	ids := map[string]bool{}
	count := 0
	for _, task := range w.Tasks {
		if task == nil {
			return fmt.Errorf("task cannot be null")
		}
		if err := normalizeTask(task, w.Start, w.End, now, 1, ids, &count); err != nil {
			return err
		}
	}
	return nil
}

func normalizeTask(t *model.Task, start, end string, now time.Time, depth int, ids map[string]bool, count *int) error {
	if depth > MaxDepth {
		return fmt.Errorf("任务层级最多 %d 层", MaxDepth)
	}
	*count++
	if *count > MaxTasks {
		return fmt.Errorf("单周任务数量不能超过 %d 个", MaxTasks)
	}
	if t.Status == "" {
		t.Status = model.StatusTodo
	}
	switch t.Status {
	case model.StatusTodo, model.StatusDone, model.StatusDropped:
	default:
		return fmt.Errorf("invalid task status %q", t.Status)
	}
	t.Text = cleanText(t.Text)
	if t.ID == "" || ids[t.ID] {
		t.ID = newTaskID(ids)
	}
	ids[t.ID] = true
	if t.Children == nil {
		t.Children = []*model.Task{}
	}
	if t.Day != "" {
		if _, err := ParseDate(t.Day); err != nil || t.Day < start || t.Day > end {
			t.Day = ""
		}
	}
	if !validTimestamp(t.CreatedAt) {
		t.CreatedAt = now.Format(time.RFC3339)
	}
	if !validTimestamp(t.UpdatedAt) {
		t.UpdatedAt = t.CreatedAt
	}
	if !validTimestamp(t.CompletedAt) {
		t.CompletedAt = ""
	}
	if !validTimestamp(t.DroppedAt) {
		t.DroppedAt = ""
	}
	if t.Status == model.StatusDropped {
		for _, child := range t.Children {
			if child == nil {
				return fmt.Errorf("task cannot be null")
			}
			if err := normalizeTask(child, start, end, now, depth+1, ids, count); err != nil {
				return err
			}
		}
		cascadeDrop(t, now)
		return nil
	}
	t.DroppedAt = ""
	for _, child := range t.Children {
		if child == nil {
			return fmt.Errorf("task cannot be null")
		}
		if err := normalizeTask(child, start, end, now, depth+1, ids, count); err != nil {
			return err
		}
	}
	deriveParent(t, now)
	return nil
}

func deriveParent(t *model.Task, now time.Time) {
	if len(t.Children) == 0 {
		if t.Status == model.StatusDone && t.CompletedAt == "" {
			t.CompletedAt = now.Format(time.RFC3339)
		}
		if t.Status != model.StatusDone {
			t.CompletedAt = ""
		}
		return
	}
	active := 0
	allDone := true
	latest := ""
	for _, child := range t.Children {
		if child.Status == model.StatusDropped {
			continue
		}
		active++
		if child.Status != model.StatusDone {
			allDone = false
			continue
		}
		if newerTimestamp(child.CompletedAt, latest) {
			latest = child.CompletedAt
		}
	}
	if active > 0 {
		if allDone {
			t.Status = model.StatusDone
			if latest == "" {
				latest = now.Format(time.RFC3339)
			}
			t.CompletedAt = latest
			return
		}
		t.Status = model.StatusTodo
		t.CompletedAt = ""
		return
	}
	// All children are dropped. Keep the parent's explicit state, but keep
	// timestamps consistent.
	if t.Status != model.StatusDone {
		t.CompletedAt = ""
	} else if t.CompletedAt == "" {
		t.CompletedAt = now.Format(time.RFC3339)
	}
}

func cascadeDrop(t *model.Task, now time.Time) {
	t.Status = model.StatusDropped
	if t.DroppedAt == "" {
		t.DroppedAt = now.Format(time.RFC3339)
	}
	t.CompletedAt = ""
	for _, child := range t.Children {
		if child == nil {
			continue
		}
		cascadeDrop(child, now)
	}
}

func cleanText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) > MaxTextRunes {
		runes := []rune(value)
		value = string(runes[:MaxTextRunes])
	}
	return value
}

func validTimestamp(value string) bool {
	if value == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func newerTimestamp(candidate, current string) bool {
	if candidate == "" {
		return false
	}
	if current == "" {
		return true
	}
	parsedCandidate, errCandidate := time.Parse(time.RFC3339, candidate)
	parsedCurrent, errCurrent := time.Parse(time.RFC3339, current)
	if errCandidate != nil || errCurrent != nil {
		return candidate > current
	}
	return parsedCandidate.After(parsedCurrent)
}

func newTaskID(existing map[string]bool) string {
	for {
		buf := make([]byte, 8)
		if _, err := rand.Read(buf); err != nil {
			panic("weektodo: cannot read random bytes: " + err.Error())
		}
		id := "t_" + hex.EncodeToString(buf)
		if !existing[id] {
			return id
		}
	}
}

// CloneTasks deep-copies a task tree.
func CloneTasks(tasks []*model.Task) []*model.Task {
	if tasks == nil {
		return nil
	}
	out := make([]*model.Task, len(tasks))
	for i, task := range tasks {
		out[i] = CloneTask(task)
	}
	return out
}

// CloneTask deep-copies one task.
func CloneTask(task *model.Task) *model.Task {
	if task == nil {
		return nil
	}
	out := *task
	if task.CarriedFrom != nil {
		carried := *task.CarriedFrom
		out.CarriedFrom = &carried
	}
	out.Children = CloneTasks(task.Children)
	return &out
}

// CloneWeek deep-copies a week.
func CloneWeek(w *model.Week) *model.Week {
	if w == nil {
		return nil
	}
	out := *w
	out.Tasks = CloneTasks(w.Tasks)
	return &out
}

// WeekStats counts leaf tasks, excluding dropped leaves from the total.
func WeekStats(w *model.Week) model.Stats {
	stats := model.Stats{}
	if w == nil {
		return stats
	}
	var walk func(tasks []*model.Task)
	walk = func(tasks []*model.Task) {
		for _, task := range tasks {
			if len(task.Children) > 0 {
				walk(task.Children)
				continue
			}
			switch task.Status {
			case model.StatusDropped:
				stats.Dropped++
			case model.StatusDone:
				stats.Total++
				stats.Done++
			default:
				stats.Total++
				if task.Day == "" {
					stats.Unplanned++
				}
			}
		}
	}
	walk(w.Tasks)
	return stats
}

// TodayCount returns unfinished leaf tasks scheduled for the given date.
func TodayCount(w *model.Week, day string) int {
	if w == nil {
		return 0
	}
	count := 0
	var walk func(tasks []*model.Task)
	walk = func(tasks []*model.Task) {
		for _, task := range tasks {
			if len(task.Children) > 0 {
				walk(task.Children)
				continue
			}
			if task.Status == model.StatusTodo && task.Day == day {
				count++
			}
		}
	}
	walk(w.Tasks)
	return count
}

// FlatTask is a task with its depth for previews.
type FlatTask struct {
	Task  *model.Task `json:"task"`
	Depth int         `json:"depth"`
}

// UnfinishedTasks lists unfinished tasks in tree order.
func UnfinishedTasks(w *model.Week) []FlatTask {
	out := []FlatTask{}
	if w == nil {
		return out
	}
	var walk func(tasks []*model.Task, depth int)
	walk = func(tasks []*model.Task, depth int) {
		for _, task := range tasks {
			if task.Status != model.StatusTodo {
				continue
			}
			out = append(out, FlatTask{Task: task, Depth: depth})
			walk(task.Children, depth+1)
		}
	}
	walk(w.Tasks, 0)
	return out
}

// DoneText renders completed tasks as an indented plain-text tree.
func DoneText(w *model.Week) string {
	if w == nil {
		return ""
	}
	var b strings.Builder
	var walk func(tasks []*model.Task, depth int)
	walk = func(tasks []*model.Task, depth int) {
		for _, task := range tasks {
			if task.Status != model.StatusDone {
				continue
			}
			b.WriteString(strings.Repeat("  ", depth))
			b.WriteString("- ")
			b.WriteString(task.Text)
			b.WriteByte('\n')
			walk(task.Children, depth+1)
		}
	}
	walk(w.Tasks, 0)
	return b.String()
}
