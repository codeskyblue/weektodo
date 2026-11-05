package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/uiautodev/weektodo/internal/ai"
	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/store"
	"github.com/uiautodev/weektodo/internal/todo"
	"github.com/uiautodev/weektodo/internal/update"
)

// Options carries build metadata shown in the UI.
type Options struct {
	Version string
	Commit  string
	Desktop bool
	Updater *update.Client
}

// Server exposes the JSON API and the embedded frontend.
type Server struct {
	store    *store.Store
	ai       *ai.Client
	options  Options
	mux      *http.ServeMux
	updateMu sync.Mutex
}

// New creates the HTTP server.
func New(st *store.Store, client *ai.Client, options Options) *Server {
	if options.Updater == nil {
		options.Updater = update.New(update.Options{DataDir: st.Dir(), Desktop: options.Desktop})
	}
	s := &Server{store: st, ai: client, options: options, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/meta", s.handleMeta)
	s.mux.HandleFunc("GET /api/weeks", s.handleListWeeks)
	s.mux.HandleFunc("GET /api/weeks/{id}", s.handleGetWeek)
	s.mux.HandleFunc("POST /api/weeks/{id}", s.handleCreateWeek)
	s.mux.HandleFunc("PUT /api/weeks/{id}", s.handleSaveWeek)
	s.mux.HandleFunc("DELETE /api/weeks/{id}", s.handleDeleteWeek)
	s.mux.HandleFunc("GET /api/weeks/{id}/carry-over", s.handleCarryPreview)
	s.mux.HandleFunc("POST /api/weeks/{id}/carry-over", s.handleCarry)
	s.mux.HandleFunc("GET /api/weeks/{id}/report", s.handleReport)
	s.mux.HandleFunc("POST /api/weeks/{id}/report/polish", s.handlePolish)
	s.mux.HandleFunc("GET /api/settings", s.handleGetSettings)
	s.mux.HandleFunc("PUT /api/settings", s.handleSaveSettings)
	s.mux.HandleFunc("POST /api/settings/ai/test", s.handleTestAI)
	s.mux.HandleFunc("POST /api/settings/ai/models", s.handleAIModels)
	s.mux.HandleFunc("GET /api/update/check", s.handleUpdateCheck)
	s.mux.HandleFunc("POST /api/update/apply", s.handleUpdateApply)
	s.mux.HandleFunc("GET /api/update/status", s.handleUpdateStatus)
	s.mux.HandleFunc("/", s.handleStatic)
}

// Handler returns the root handler.
func (s *Server) Handler() http.Handler {
	return logRequests(withLanguage(s.mux))
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMeta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"dataDir":       s.store.Dir(),
		"version":       s.options.Version,
		"commit":        s.options.Commit,
		"currentWeekId": currentWeekID(),
	})
}

func (s *Server) handleListWeeks(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"currentWeekId": currentWeekID(),
		"weeks":         s.store.ListWeeks(),
	})
}

func (s *Server) handleGetWeek(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	week, ok := s.store.Week(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "这一周还没有记录")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"week": week})
}

func (s *Server) handleCreateWeek(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	if week, ok := s.store.Week(id); ok {
		writeJSON(w, http.StatusOK, map[string]any{"week": week})
		return
	}
	week, err := todo.NewWeek(id, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_week", err.Error())
		return
	}
	week, err = s.store.SaveWeek(week)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"week": week})
}

func (s *Server) handleSaveWeek(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	var body struct {
		Tasks []*model.Task `json:"tasks"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	week, ok := s.store.Week(id)
	if !ok {
		week, err := todo.NewWeek(id, time.Now())
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_week", err.Error())
			return
		}
		week.Tasks = body.Tasks
		saved, err := s.store.SaveWeek(week)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_tasks", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"week": saved})
		return
	}
	week.Tasks = body.Tasks
	saved, err := s.store.SaveWeek(week)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_tasks", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"week": saved})
}

func (s *Server) handleDeleteWeek(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validWeekID(w, id) {
		return
	}
	if err := s.store.DeleteWeek(id); err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func validWeekID(w http.ResponseWriter, id string) bool {
	if _, _, err := todo.ParseWeekID(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_week", err.Error())
		return false
	}
	return true
}

func currentWeekID() string {
	return todo.WeekIDFor(time.Now())
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("请求内容为空")
		}
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	message = i18n.TranslateError(responseLanguage(w), message)
	writeJSON(w, status, map[string]string{"error": message, "code": code})
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Method != http.MethodGet {
			log.Printf("%s %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}
