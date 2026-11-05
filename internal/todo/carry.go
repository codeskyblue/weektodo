package todo

import (
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

// CarryPreview describes what the automatic carry-over would copy.
type CarryPreview struct {
	SourceWeekID string     `json:"sourceWeekId"`
	Tasks        []FlatTask `json:"tasks"`
	Count        int        `json:"count"`
	Skipped      int        `json:"skipped"`
}

// PreviewCarry lists unfinished tasks from source that are not already in target.
func PreviewCarry(source, target *model.Week) CarryPreview {
	preview := CarryPreview{Tasks: []FlatTask{}}
	if source == nil {
		return preview
	}
	preview.SourceWeekID = source.ID
	existing := carriedTaskIDs(target, source.ID)
	var walk func(tasks []*model.Task, depth int)
	walk = func(tasks []*model.Task, depth int) {
		for _, task := range tasks {
			if task.Status != model.StatusTodo {
				continue
			}
			if existing[task.ID] {
				preview.Skipped += countTodoNodes(task)
				continue
			}
			preview.Tasks = append(preview.Tasks, FlatTask{Task: task, Depth: depth})
			preview.Count++
			walk(task.Children, depth+1)
		}
	}
	walk(source.Tasks, 0)
	return preview
}

// CarryTasks copies unfinished tasks from source into target, preserving the
// hierarchy of the copied nodes. It is idempotent for tasks already carried
// into target.
func CarryTasks(source, target *model.Week, now time.Time) (copied []*model.Task, count, skipped int) {
	if source == nil || target == nil {
		return nil, 0, 0
	}
	existing := carriedTaskIDs(target, source.ID)
	var copyNode func(task *model.Task) *model.Task
	copyNode = func(task *model.Task) *model.Task {
		if task.Status != model.StatusTodo {
			return nil
		}
		if existing[task.ID] {
			skipped += countTodoNodes(task)
			return nil
		}
		out := &model.Task{
			Text:        task.Text,
			Status:      model.StatusTodo,
			Day:         "",
			CreatedAt:   now.Format(time.RFC3339),
			CarriedFrom: &model.CarriedFrom{WeekID: source.ID, TaskID: task.ID},
			Children:    []*model.Task{},
		}
		count++
		for _, child := range task.Children {
			if cloned := copyNode(child); cloned != nil {
				out.Children = append(out.Children, cloned)
			}
		}
		return out
	}
	for _, task := range source.Tasks {
		if cloned := copyNode(task); cloned != nil {
			copied = append(copied, cloned)
		}
	}
	return copied, count, skipped
}

func carriedTaskIDs(target *model.Week, sourceWeekID string) map[string]bool {
	ids := map[string]bool{}
	if target == nil {
		return ids
	}
	var walk func(tasks []*model.Task)
	walk = func(tasks []*model.Task) {
		for _, task := range tasks {
			if task.CarriedFrom != nil && task.CarriedFrom.WeekID == sourceWeekID {
				ids[task.CarriedFrom.TaskID] = true
			}
			walk(task.Children)
		}
	}
	walk(target.Tasks)
	return ids
}

func countTodoNodes(task *model.Task) int {
	if task == nil || task.Status != model.StatusTodo {
		return 0
	}
	count := 1
	for _, child := range task.Children {
		count += countTodoNodes(child)
	}
	return count
}
