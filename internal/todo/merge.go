package todo

import (
	"sort"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

// MergeStats describes the merged result.
type MergeStats struct {
	Weeks int `json:"weeks"`
	Tasks int `json:"tasks"`
}

// MergeData merges two task documents. Tasks with the same ID are resolved by
// their latest updated/completed/dropped/created timestamp; children are merged
// recursively so tasks created on either side are retained.
func MergeData(local, remote model.Data, now time.Time) (model.Data, MergeStats, error) {
	byID := make(map[string]*model.Week, len(local.Weeks)+len(remote.Weeks))
	order := make([]string, 0, len(local.Weeks)+len(remote.Weeks))
	for _, week := range local.Weeks {
		if week == nil {
			continue
		}
		if _, ok := byID[week.ID]; !ok {
			order = append(order, week.ID)
		}
		byID[week.ID] = CloneWeek(week)
	}
	for _, week := range remote.Weeks {
		if week == nil {
			continue
		}
		if existing, ok := byID[week.ID]; ok {
			existing.Tasks = mergeTaskLists(existing.Tasks, week.Tasks)
			continue
		}
		byID[week.ID] = CloneWeek(week)
		order = append(order, week.ID)
	}
	merged := model.Data{Version: 1, Weeks: make([]*model.Week, 0, len(order))}
	stats := MergeStats{}
	for _, id := range order {
		week := byID[id]
		if week == nil {
			continue
		}
		if err := NormalizeWeek(week, now); err != nil {
			return model.Data{}, MergeStats{}, err
		}
		merged.Weeks = append(merged.Weeks, week)
		stats.Weeks++
		stats.Tasks += countTasks(week.Tasks)
	}
	sort.Slice(merged.Weeks, func(i, j int) bool { return merged.Weeks[i].Start < merged.Weeks[j].Start })
	return merged, stats, nil
}

func mergeTaskLists(local, remote []*model.Task) []*model.Task {
	remoteByID := make(map[string]*model.Task, len(remote))
	localByID := make(map[string]bool, len(local))
	for _, task := range remote {
		if task != nil {
			remoteByID[task.ID] = task
		}
	}
	for _, task := range local {
		if task == nil {
			continue
		}
		localByID[task.ID] = true
	}
	out := make([]*model.Task, 0, len(local)+len(remote))
	for _, task := range local {
		if task == nil {
			continue
		}
		if counterpart, ok := remoteByID[task.ID]; ok {
			out = append(out, mergeTask(task, counterpart))
		} else {
			out = append(out, CloneTask(task))
		}
	}
	for _, task := range remote {
		if task == nil || localByID[task.ID] {
			continue
		}
		out = append(out, CloneTask(task))
	}
	return out
}

func mergeTask(local, remote *model.Task) *model.Task {
	chosen, other := local, remote
	if taskUpdatedAt(remote).After(taskUpdatedAt(local)) {
		chosen, other = remote, local
	}
	out := CloneTask(chosen)
	out.Children = mergeTaskLists(chosen.Children, other.Children)
	return out
}

func taskUpdatedAt(task *model.Task) time.Time {
	if task == nil {
		return time.Time{}
	}
	var latest time.Time
	for _, value := range []string{task.UpdatedAt, task.CompletedAt, task.DroppedAt, task.CreatedAt} {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil && parsed.After(latest) {
			latest = parsed
		}
	}
	return latest
}

func countTasks(tasks []*model.Task) int {
	count := 0
	var walk func([]*model.Task)
	walk = func(list []*model.Task) {
		for _, task := range list {
			if task == nil {
				continue
			}
			count++
			walk(task.Children)
		}
	}
	walk(tasks)
	return count
}

// CloneData deep-copies a complete document.
func CloneData(data model.Data) model.Data {
	out := model.Data{Version: data.Version}
	out.Weeks = make([]*model.Week, 0, len(data.Weeks))
	for _, week := range data.Weeks {
		out.Weeks = append(out.Weeks, CloneWeek(week))
	}
	return out
}
