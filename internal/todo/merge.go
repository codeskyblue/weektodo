package todo

import (
	"sort"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

// MergeStats describes the merged result.
type MergeStats struct {
	Categories int `json:"categories"`
	Weeks      int `json:"weeks"`
	Tasks      int `json:"tasks"`
}

type weekKey struct {
	category string
	id       string
}

// MergeData merges two task documents. Weeks are keyed by category and ID,
// tasks by ID, and the newest updated/completed/dropped/created timestamp wins.
func MergeData(local, remote model.Data, now time.Time) (model.Data, MergeStats, error) {
	categories, err := mergeCategories(local.Categories, remote.Categories, now)
	if err != nil {
		return model.Data{}, MergeStats{}, err
	}

	byKey := make(map[weekKey]*model.Week, len(local.Weeks)+len(remote.Weeks))
	keys := make([]weekKey, 0, len(local.Weeks)+len(remote.Weeks))
	add := func(week *model.Week) error {
		if week == nil {
			return nil
		}
		category := week.Category
		if category == "" {
			category = model.DefaultCategoryID
		}
		clone := CloneWeek(week)
		clone.Category = category
		key := weekKey{category: category, id: clone.ID}
		if existing, ok := byKey[key]; ok {
			existing.Tasks = mergeTaskLists(existing.Tasks, clone.Tasks)
			return nil
		}
		byKey[key] = clone
		keys = append(keys, key)
		return nil
	}
	for _, week := range local.Weeks {
		if err := add(week); err != nil {
			return model.Data{}, MergeStats{}, err
		}
	}
	for _, week := range remote.Weeks {
		if err := add(week); err != nil {
			return model.Data{}, MergeStats{}, err
		}
	}

	merged := model.Data{Version: 2, Categories: categories, Weeks: make([]*model.Week, 0, len(keys))}
	stats := MergeStats{Categories: len(categories)}
	for _, key := range keys {
		week := byKey[key]
		if err := NormalizeWeek(week, now); err != nil {
			return model.Data{}, MergeStats{}, err
		}
		merged.Weeks = append(merged.Weeks, week)
		stats.Weeks++
		stats.Tasks += countTasks(week.Tasks)
	}
	sort.Slice(merged.Weeks, func(i, j int) bool {
		if merged.Weeks[i].Category != merged.Weeks[j].Category {
			return merged.Weeks[i].Category < merged.Weeks[j].Category
		}
		return merged.Weeks[i].Start < merged.Weeks[j].Start
	})
	return merged, stats, nil
}

func mergeCategories(local, remote []*model.Category, now time.Time) ([]*model.Category, error) {
	byID := map[string]*model.Category{}
	for _, list := range [][]*model.Category{local, remote} {
		for _, category := range list {
			if category == nil {
				continue
			}
			clone := CloneCategory(category)
			if err := NormalizeCategory(clone, now); err != nil {
				return nil, err
			}
			if existing, ok := byID[clone.ID]; ok {
				if categoryUpdatedAt(clone).After(categoryUpdatedAt(existing)) {
					byID[clone.ID] = clone
				}
				continue
			}
			byID[clone.ID] = clone
		}
	}
	if _, ok := byID[model.DefaultCategoryID]; !ok {
		byID[model.DefaultCategoryID] = &model.Category{
			ID:        model.DefaultCategoryID,
			Name:      "默认",
			Color:     "#3b82f6",
			CreatedAt: now.Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
		}
	}
	out := make([]*model.Category, 0, len(byID))
	for _, category := range byID {
		out = append(out, category)
	}
	SortCategories(out)
	return out, nil
}

func categoryUpdatedAt(category *model.Category) time.Time {
	if category == nil {
		return time.Time{}
	}
	var latest time.Time
	for _, value := range []string{category.UpdatedAt, category.CreatedAt} {
		if parsed, err := time.Parse(time.RFC3339, value); err == nil && parsed.After(latest) {
			latest = parsed
		}
	}
	return latest
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
	out.Categories = make([]*model.Category, 0, len(data.Categories))
	for _, category := range data.Categories {
		out.Categories = append(out.Categories, CloneCategory(category))
	}
	out.Weeks = make([]*model.Week, 0, len(data.Weeks))
	for _, week := range data.Weeks {
		out.Weeks = append(out.Weeks, CloneWeek(week))
	}
	return out
}
