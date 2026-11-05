package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/store"
)

func (s *Server) requestCategory(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := strings.TrimSpace(r.URL.Query().Get("category"))
	if id == "" {
		id = model.DefaultCategoryID
	}
	category, ok := s.store.Category(id)
	if !ok || category.Archived {
		writeError(w, http.StatusNotFound, "not_found", i18n.T(responseLanguage(w),
			"分类不存在或已归档",
			"Category does not exist or is archived"))
		return "", false
	}
	return id, true
}

func (s *Server) handleListCategories(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"categories": s.store.Categories(),
	})
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name  string `json:"name"`
		Color string `json:"color"`
		Order int    `json:"order"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	lang := responseLanguage(w)
	if strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "invalid_category", i18n.T(lang,
			"分类名称不能为空",
			"Category name cannot be empty"))
		return
	}
	category, err := s.store.SaveCategory(model.Category{
		Name:  body.Name,
		Color: body.Color,
		Order: body.Order,
	})
	if err != nil {
		writeCategoryError(w, lang, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"category": category})
}

func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	current, ok := s.store.Category(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", i18n.T(responseLanguage(w),
			"分类不存在",
			"Category not found"))
		return
	}
	var body struct {
		Name     *string `json:"name"`
		Color    *string `json:"color"`
		Order    *int    `json:"order"`
		Archived *bool   `json:"archived"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	lang := responseLanguage(w)
	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name == "" {
			writeError(w, http.StatusBadRequest, "invalid_category", i18n.T(lang,
				"分类名称不能为空",
				"Category name cannot be empty"))
			return
		}
		current.Name = name
	}
	if body.Color != nil {
		current.Color = *body.Color
	}
	if body.Order != nil {
		current.Order = *body.Order
	}
	if body.Archived != nil {
		if *body.Archived && !current.Archived {
			active := 0
			for _, category := range s.store.Categories() {
				if !category.Archived && category.ID != current.ID {
					active++
				}
			}
			if active == 0 {
				writeError(w, http.StatusBadRequest, "invalid_category", i18n.T(lang,
					"至少保留一个未归档的分类",
					"Keep at least one active category"))
				return
			}
		}
		current.Archived = *body.Archived
	}
	category, err := s.store.SaveCategory(current)
	if err != nil {
		writeCategoryError(w, lang, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"category": category})
}

func writeCategoryError(w http.ResponseWriter, lang i18n.Language, err error) {
	switch {
	case errors.Is(err, store.ErrCategoryNameTaken):
		writeError(w, http.StatusConflict, "category_name_taken", i18n.T(lang,
			"分类名称已存在",
			"Category name already exists"))
	case errors.Is(err, store.ErrCategoryNotFound):
		writeError(w, http.StatusNotFound, "not_found", i18n.T(lang,
			"分类不存在",
			"Category not found"))
	default:
		writeError(w, http.StatusBadRequest, "invalid_category", err.Error())
	}
}
