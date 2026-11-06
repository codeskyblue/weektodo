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

// PrepareMovedTask bumps the update time for the moved subtree and clears
// week-specific metadata when the task changes week.
func PrepareMovedTask(task *model.Task, now time.Time, clearWeek bool) {
	stamp := now.Format(time.RFC3339)
	var walk func(*model.Task)
	walk = func(item *model.Task) {
		if item == nil {
			return
		}
		if clearWeek {
			item.Day = ""
			item.CarriedFrom = nil
		}
		item.UpdatedAt = stamp
		for _, child := range item.Children {
			walk(child)
		}
	}
	walk(task)
}

// FindTask returns the task with id from a task tree.
func FindTask(tasks []*model.Task, id string) *model.Task {
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if task.ID == id {
			return task
		}
		if found := FindTask(task.Children, id); found != nil {
			return found
		}
	}
	return nil
}

// ContainsTask reports whether id is task itself or one of its descendants.
func ContainsTask(task *model.Task, id string) bool {
	if task == nil {
		return false
	}
	if task.ID == id {
		return true
	}
	for _, child := range task.Children {
		if ContainsTask(child, id) {
			return true
		}
	}
	return false
}
