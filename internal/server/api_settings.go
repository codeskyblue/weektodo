package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/uiautodev/weektodo/internal/ai"
	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/webdav"
)

func (s *Server) handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	settings := s.store.Settings()
	writeJSON(w, http.StatusOK, map[string]any{
		"ai":       publicAI(settings.AI),
		"webdav":   publicWebDAV(settings.WebDAV),
		"weekSpan": settings.WeekSpan,
		"language": settings.Language,
	})
}

func (s *Server) handleSaveSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AI struct {
			BaseURL     *string `json:"baseUrl"`
			Model       *string `json:"model"`
			APIKey      *string `json:"apiKey"`
			ClearAPIKey bool    `json:"clearApiKey"`
		} `json:"ai"`
		WebDAV struct {
			URL           *string `json:"url"`
			Username      *string `json:"username"`
			Password      *string `json:"password"`
			RemoteDir     *string `json:"remoteDir"`
			ClearPassword bool    `json:"clearPassword"`
		} `json:"webdav"`
		WeekSpan *int    `json:"weekSpan"`
		Language *string `json:"language"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	current := s.store.Settings()
	if body.AI.BaseURL != nil {
		current.AI.BaseURL = *body.AI.BaseURL
	}
	if body.AI.Model != nil {
		current.AI.Model = *body.AI.Model
	}
	if body.AI.ClearAPIKey {
		current.AI.APIKey = ""
	} else if body.AI.APIKey != nil && strings.TrimSpace(*body.AI.APIKey) != "" {
		current.AI.APIKey = *body.AI.APIKey
	}
	if body.WebDAV.URL != nil {
		current.WebDAV.URL = *body.WebDAV.URL
	}
	if body.WebDAV.Username != nil {
		current.WebDAV.Username = *body.WebDAV.Username
	}
	if body.WebDAV.RemoteDir != nil {
		current.WebDAV.RemoteDir = *body.WebDAV.RemoteDir
	}
	if body.WebDAV.ClearPassword {
		current.WebDAV.Password = ""
	} else if body.WebDAV.Password != nil && strings.TrimSpace(*body.WebDAV.Password) != "" {
		current.WebDAV.Password = *body.WebDAV.Password
	}
	if strings.TrimSpace(current.WebDAV.URL) != "" {
		if err := webdav.Validate(current.WebDAV, responseLanguage(w)); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_webdav", err.Error())
			return
		}
	}
	if body.WeekSpan != nil {
		switch *body.WeekSpan {
		case 1, 2, 3:
			current.WeekSpan = *body.WeekSpan
		default:
			writeError(w, http.StatusBadRequest, "invalid_week_span", i18n.T(responseLanguage(w),
				"显示周数只能是 1、2 或 3 周",
				"Weeks shown must be 1, 2 or 3"))
			return
		}
	}
	if body.Language != nil {
		if lang := i18n.Normalize(*body.Language); lang != "" {
			current.Language = string(lang)
		}
	}
	if err := s.store.UpdateSettings(current); err != nil {
		writeError(w, http.StatusInternalServerError, "save_failed", err.Error())
		return
	}
	settings := s.store.Settings()
	writeJSON(w, http.StatusOK, map[string]any{
		"ai":       publicAI(settings.AI),
		"webdav":   publicWebDAV(settings.WebDAV),
		"weekSpan": settings.WeekSpan,
		"language": settings.Language,
	})
}

func (s *Server) handleAIModels(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseURL *string `json:"baseUrl"`
		APIKey  *string `json:"apiKey"`
	}
	_ = decodeJSON(w, r, &body)
	cfg := s.store.Settings().AI
	if body.BaseURL != nil && strings.TrimSpace(*body.BaseURL) != "" {
		cfg.BaseURL = *body.BaseURL
	}
	if body.APIKey != nil && strings.TrimSpace(*body.APIKey) != "" {
		cfg.APIKey = *body.APIKey
	}
	lang := responseLanguage(w)
	if err := ai.ValidateEndpoint(cfg, lang); err != nil {
		writeError(w, http.StatusBadRequest, "ai_not_configured", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	models, err := s.ai.Models(ctx, cfg, lang)
	if err != nil {
		writeError(w, http.StatusBadGateway, "ai_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"models": models,
		"count":  len(models),
	})
}

func (s *Server) handleTestAI(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BaseURL *string `json:"baseUrl"`
		Model   *string `json:"model"`
		APIKey  *string `json:"apiKey"`
	}
	_ = decodeJSON(w, r, &body)
	cfg := s.store.Settings().AI
	if body.BaseURL != nil && strings.TrimSpace(*body.BaseURL) != "" {
		cfg.BaseURL = *body.BaseURL
	}
	if body.Model != nil && strings.TrimSpace(*body.Model) != "" {
		cfg.Model = *body.Model
	}
	if body.APIKey != nil && strings.TrimSpace(*body.APIKey) != "" {
		cfg.APIKey = *body.APIKey
	}
	lang := responseLanguage(w)
	if err := ai.ValidateConfig(cfg, lang); err != nil {
		writeError(w, http.StatusBadRequest, "ai_not_configured", err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	message, err := s.ai.Test(ctx, cfg, lang)
	if err != nil {
		writeError(w, http.StatusBadGateway, "ai_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": message,
		"model":   cfg.Model,
	})
}

func publicAI(cfg model.AIConfig) map[string]any {
	configured := strings.TrimSpace(cfg.BaseURL) != "" &&
		strings.TrimSpace(cfg.Model) != "" &&
		strings.TrimSpace(cfg.APIKey) != ""
	return map[string]any{
		"baseUrl":    cfg.BaseURL,
		"model":      cfg.Model,
		"configured": configured,
		"apiKeyHint": keyHint(cfg.APIKey),
	}
}

func keyHint(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	runes := []rune(key)
	if len(runes) <= 4 {
		return "••••"
	}
	return "••••" + string(runes[len(runes)-4:])
}

func publicWebDAV(cfg model.WebDAVConfig) map[string]any {
	return map[string]any{
		"url":          cfg.URL,
		"username":     cfg.Username,
		"remoteDir":    cfg.RemoteDir,
		"hasPassword":  strings.TrimSpace(cfg.Password) != "",
		"passwordHint": keyHint(cfg.Password),
	}
}
