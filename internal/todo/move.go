package todo

import (
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

// TakeTask removes the task with id from the tree and returns the remaining
// siblings together with the removed subtree.
func TakeTask(tasks []*model.Task, id string) ([]*model.Task, *model.Task, bool) {
	for index, task := range tasks {
		if task == nil {
			continue
		}
		if task.ID == id {
			remaining := make([]*model.Task, 0, len(tasks)-1)
			remaining = append(remaining, tasks[:index]...)
			remaining = append(remaining, tasks[index+1:]...)
			return remaining, task, true
		}
		children, found, ok := TakeTask(task.Children, id)
		if ok {
			task.Children = children
			return tasks, found, true
		}
	}
	return tasks, nil, false
}

// PrepareMovedTask clears week-specific metadata and bumps the update time for
// the whole moved subtree.
func PrepareMovedTask(task *model.Task, now time.Time) {
	stamp := now.Format(time.RFC3339)
	var walk func(*model.Task)
	walk = func(item *model.Task) {
		if item == nil {
			return
		}
		item.Day = ""
		item.CarriedFrom = nil
		item.UpdatedAt = stamp
		for _, child := range item.Children {
			walk(child)
		}
	}
	walk(task)
}
