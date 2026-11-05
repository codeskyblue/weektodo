package server

import (
	"log"
	"net/http"
	"time"
)

func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	info, err := s.options.Updater.Check(r.Context(), s.options.Version)
	if err != nil {
		writeError(w, http.StatusBadGateway, "update_check_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleUpdateApply(w http.ResponseWriter, r *http.Request) {
	if !s.updateMu.TryLock() {
		writeError(w, http.StatusConflict, "update_in_progress", "已有更新正在进行")
		return
	}
	info, err := s.options.Updater.Check(r.Context(), s.options.Version)
	if err != nil {
		s.updateMu.Unlock()
		writeError(w, http.StatusBadGateway, "update_check_failed", err.Error())
		return
	}
	if !info.Available {
		s.updateMu.Unlock()
		writeError(w, http.StatusConflict, "up_to_date", "当前已经是最新版本")
		return
	}
	result, err := s.options.Updater.Apply(r.Context(), info)
	if err != nil {
		s.updateMu.Unlock()
		writeError(w, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"installed":  result.Installed,
		"path":       result.Path,
		"version":    info.Latest,
		"source":     info.Source,
		"restarting": true,
	})
	go func() {
		time.Sleep(700 * time.Millisecond)
		log.Printf("restarting after update to %s", info.Latest)
		if err := s.options.Updater.Restart(); err != nil {
			log.Printf("restart after update: %v", err)
			s.updateMu.Unlock()
		}
	}()
}

func (s *Server) handleUpdateStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.options.Updater.Progress())
}
