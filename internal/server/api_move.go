package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/store"
	"github.com/uiautodev/weektodo/internal/todo"
)

func (s *Server) handleMoveTask(w http.ResponseWriter, r *http.Request) {
	fromID := r.PathValue("id")
	if !validWeekID(w, fromID) {
		return
	}
	category, ok := s.requestCategory(w, r)
	if !ok {
		return
	}
	var body struct {
		TaskID   string `json:"taskId"`
		ToWeekID string `json:"toWeekId"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	body.TaskID = strings.TrimSpace(body.TaskID)
	body.ToWeekID = strings.TrimSpace(body.ToWeekID)
	if body.TaskID == "" || body.ToWeekID == "" {
		writeError(w, http.StatusBadRequest, "invalid_move", i18n.T(responseLanguage(w),
			"请选择要移动的任务和目标周",
			"Choose a task and a target week"))
		return
	}
	if _, _, err := todo.ParseWeekID(body.ToWeekID); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_week", i18n.T(responseLanguage(w),
			"目标周格式不正确",
			"Target week is invalid"))
		return
	}

	lang := responseLanguage(w)
	from, to, err := s.store.MoveTaskInCategory(category, fromID, body.TaskID, body.ToWeekID, time.Now())
	switch {
	case errors.Is(err, store.ErrSameWeek):
		writeError(w, http.StatusBadRequest, "invalid_move", i18n.T(lang,
			"不能移动到同一周",
			"Cannot move a task to the same week"))
		return
	case errors.Is(err, store.ErrWeekNotFound):
		writeError(w, http.StatusNotFound, "not_found", i18n.T(lang,
			"这一周还没有记录",
			"There is no record for this week"))
		return
	case errors.Is(err, store.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "not_found", i18n.T(lang,
			"任务不存在",
			"Task not found"))
		return
	case err != nil:
		writeError(w, http.StatusBadRequest, "invalid_move", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"from": from,
		"to":   to,
	})
}
