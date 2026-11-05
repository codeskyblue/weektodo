package server

import (
	"context"
	"net/http"
	"time"

	"github.com/uiautodev/weektodo/internal/ai"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/todo"
)

type carryItem struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Depth int    `json:"depth"`
}

func (s *Server) handleCarryPreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	sourceID := todo.PreviousWeekID(id)
	if sourceID == "" {
		writeError(w, http.StatusBadRequest, "invalid_week", "无法计算上一周")
		return
	}
	source, ok := s.store.Week(sourceID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"sourceWeekId": sourceID,
			"count":        0,
			"skipped":      0,
			"tasks":        []carryItem{},
			"reason":       "no_previous_week",
		})
		return
	}
	target, _ := s.store.Week(id)
	if target == nil {
		target = &model.Week{ID: id, Tasks: []*model.Task{}}
	}
	preview := todo.PreviewCarry(source, target)
	items := make([]carryItem, 0, len(preview.Tasks))
	for _, flat := range preview.Tasks {
		items = append(items, carryItem{ID: flat.Task.ID, Text: flat.Task.Text, Depth: flat.Depth})
	}
	reason := ""
	if preview.Count == 0 {
		reason = "no_unfinished"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sourceWeekId": preview.SourceWeekID,
		"count":        preview.Count,
		"skipped":      preview.Skipped,
		"tasks":        items,
		"reason":       reason,
	})
}

func (s *Server) handleCarry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	sourceID := todo.PreviousWeekID(id)
	if sourceID == "" {
		writeError(w, http.StatusBadRequest, "invalid_week", "无法计算上一周")
		return
	}
	target, exists := s.store.Week(id)
	if !exists {
		fresh, err := todo.NewWeek(id, time.Now())
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_week", err.Error())
			return
		}
		target = fresh
	}
	source, ok := s.store.Week(sourceID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"week":    target,
			"created": 0,
			"skipped": 0,
			"reason":  "no_previous_week",
		})
		return
	}
	copied, created, skipped := todo.CarryTasks(source, target, time.Now())
	if created > 0 {
		target.Tasks = append(target.Tasks, copied...)
		saved, err := s.store.SaveWeek(target)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
			return
		}
		target = saved
	} else if !exists {
		// Do not persist an empty week just because the user asked.
		target.Tasks = []*model.Task{}
	}
	reason := ""
	if created == 0 {
		reason = "no_unfinished"
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"week":    target,
		"created": created,
		"skipped": skipped,
		"reason":  reason,
	})
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	mode := r.URL.Query().Get("mode")
	if mode != "full" {
		mode = "brief"
	}
	week, ok := s.store.Week(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "这一周还没有记录")
		return
	}
	lang := requestLanguage(r, s.store.Settings().Language)
	markdown, stats := todo.BuildReport(week, mode, lang)
	writeJSON(w, http.StatusOK, map[string]any{
		"markdown": markdown,
		"stats":    stats,
		"mode":     mode,
	})
}

func (s *Server) handlePolish(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	var body struct {
		Mode string `json:"mode"`
		Lang string `json:"lang"`
	}
	_ = decodeJSON(w, r, &body)
	mode := body.Mode
	if mode != "full" {
		mode = "brief"
	}
	fallbackLang := body.Lang
	if fallbackLang == "" {
		fallbackLang = s.store.Settings().Language
	}
	lang := requestLanguage(r, fallbackLang)
	week, ok := s.store.Week(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "这一周还没有记录")
		return
	}
	cfg := s.store.Settings().AI
	if err := ai.ValidateConfig(cfg, lang); err != nil {
		writeError(w, http.StatusBadRequest, "ai_not_configured", err.Error())
		return
	}
	report, _ := todo.BuildReport(week, mode, lang)
	ctx, cancel := context.WithTimeout(r.Context(), 100*time.Second)
	defer cancel()
	markdown, err := s.ai.Polish(ctx, cfg, week, report, lang)
	if err != nil {
		writeError(w, http.StatusBadGateway, "ai_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"markdown": markdown,
		"mode":     mode,
	})
}
