package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/todo"
	"github.com/uiautodev/weektodo/internal/webdav"
)

const webdavFileName = "data.json"

// webdavDraft is the optional request body used by test and sync.
type webdavDraft struct {
	URL       *string `json:"url"`
	Username  *string `json:"username"`
	Password  *string `json:"password"`
	RemoteDir *string `json:"remoteDir"`
}

// webdavConfig applies the draft on top of the saved settings. An empty
// password keeps the saved one so the UI never has to resend it.
func (s *Server) webdavConfig(w http.ResponseWriter, r *http.Request) (model.WebDAVConfig, bool) {
	var draft webdavDraft
	_ = decodeJSON(w, r, &draft)
	saved := s.store.Settings().WebDAV
	cfg := saved
	if draft.URL != nil {
		cfg.URL = *draft.URL
	}
	if draft.Username != nil {
		cfg.Username = *draft.Username
	}
	if draft.RemoteDir != nil {
		cfg.RemoteDir = *draft.RemoteDir
	}
	if draft.Password != nil && strings.TrimSpace(*draft.Password) != "" {
		cfg.Password = *draft.Password
	} else {
		cfg.Password = saved.Password
	}
	lang := responseLanguage(w)
	if err := webdav.Validate(cfg, lang); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_webdav", err.Error())
		return model.WebDAVConfig{}, false
	}
	return cfg, true
}

func (s *Server) handleTestWebDAV(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.webdavConfig(w, r)
	if !ok {
		return
	}
	client := webdav.New(cfg, responseLanguage(w))
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if err := client.Test(ctx); err != nil {
		writeError(w, http.StatusBadGateway, "webdav_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleSyncWebDAV(w http.ResponseWriter, r *http.Request) {
	cfg, ok := s.webdavConfig(w, r)
	if !ok {
		return
	}
	lang := responseLanguage(w)
	client := webdav.New(cfg, lang)

	s.webdavMu.Lock()
	defer s.webdavMu.Unlock()

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	remote := model.Data{Version: 1, Weeks: []*model.Week{}}
	raw, exists, err := client.Get(ctx, webdavFileName)
	if err != nil {
		writeError(w, http.StatusBadGateway, "webdav_failed", err.Error())
		return
	}
	if exists {
		if err := json.Unmarshal(raw, &remote); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "webdav_invalid_remote",
				i18n.T(lang, "远端 data.json 解析失败，未做任何修改", "Remote data.json is invalid; nothing was changed")+": "+err.Error())
			return
		}
	}

	merged, stats, err := todo.MergeData(s.store.Snapshot(), remote, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tasks", err.Error())
		return
	}
	saved, err := s.store.ReplaceData(merged)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	payload, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	if err := client.Put(ctx, webdavFileName, payload); err != nil {
		writeError(w, http.StatusBadGateway, "webdav_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"weeks":      stats.Weeks,
		"tasks":      stats.Tasks,
		"remotePath": strings.TrimPrefix(strings.TrimRight(cfg.RemoteDir, "/")+"/"+webdavFileName, "/"),
		"uploaded":   true,
	})
}
