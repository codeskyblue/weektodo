package todo

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
)

// NormalizeCategory validates and canonicalizes category metadata.
func NormalizeCategory(category *model.Category, now time.Time) error {
	if category == nil {
		return fmt.Errorf("category cannot be null")
	}
	category.ID = strings.TrimSpace(category.ID)
	if category.ID == "" {
		return fmt.Errorf("category id cannot be empty")
	}
	category.Name = strings.TrimSpace(category.Name)
	if category.Name == "" {
		category.Name = category.ID
	}
	if len([]rune(category.Name)) > 40 {
		category.Name = string([]rune(category.Name)[:40])
	}
	category.Color = strings.TrimSpace(category.Color)
	if category.Order < 0 {
		category.Order = 0
	}
	if category.CreatedAt == "" || !validTimestamp(category.CreatedAt) {
		if validTimestamp(category.UpdatedAt) {
			category.CreatedAt = category.UpdatedAt
		} else {
			category.CreatedAt = now.Format(time.RFC3339)
		}
	}
	if category.UpdatedAt == "" || !validTimestamp(category.UpdatedAt) {
		category.UpdatedAt = category.CreatedAt
	}
	return nil
}

// CloneCategory deep-copies category metadata.
func CloneCategory(category *model.Category) *model.Category {
	if category == nil {
		return nil
	}
	copy := *category
	return &copy
}

// SortCategories orders categories by their manual order, then name.
func SortCategories(categories []*model.Category) {
	sort.SliceStable(categories, func(i, j int) bool {
		if categories[i].Order != categories[j].Order {
			return categories[i].Order < categories[j].Order
		}
		return categories[i].Name < categories[j].Name
	})
}

// EnsureDefaultCategory adds the built-in default category when missing.
func EnsureDefaultCategory(data *model.Data, name string, now time.Time) bool {
	for _, category := range data.Categories {
		if category != nil && category.ID == model.DefaultCategoryID {
			return false
		}
	}
	stamp := now.Format(time.RFC3339)
	data.Categories = append(data.Categories, &model.Category{
		ID:        model.DefaultCategoryID,
		Name:      name,
		Color:     "#3b82f6",
		Order:     0,
		CreatedAt: stamp,
		UpdatedAt: stamp,
	})
	return true
}
