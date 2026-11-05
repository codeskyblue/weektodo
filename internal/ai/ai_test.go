package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/todo"
)

func TestPolishCallsOpenAICompatibleEndpoint(t *testing.T) {
	var gotPath, gotAuth, gotModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		var body struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		gotModel = body.Model
		if len(body.Messages) != 2 || !strings.Contains(body.Messages[1].Content, "2026-W45") {
			t.Errorf("messages = %+v", body.Messages)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{"message": map[string]any{"content": "# 周报\n完成事项"}}},
		})
	}))
	defer server.Close()

	week := &model.Week{ID: "2026-W45", Start: "2026-11-02", End: "2026-11-08", Tasks: []*model.Task{
		{ID: "a", Text: "done", Status: model.StatusDone, CompletedAt: "2026-11-05T10:00:00+08:00", Children: []*model.Task{}},
	}}
	if err := todo.NormalizeWeek(week, time.Now()); err != nil {
		t.Fatal(err)
	}
	client := New()
	result, err := client.Polish(context.Background(), model.AIConfig{
		BaseURL: server.URL + "/v1",
		Model:   "test-model",
		APIKey:  "test-key",
	}, week, "local report")
	if err != nil {
		t.Fatal(err)
	}
	if result != "# 周报\n完成事项" {
		t.Fatalf("result = %q", result)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path = %s", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("auth = %s", gotAuth)
	}
	if gotModel != "test-model" {
		t.Fatalf("model = %s", gotModel)
	}
}

func TestPolishSurfacesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"bad key"}}`, http.StatusUnauthorized)
	}))
	defer server.Close()
	client := New()
	_, err := client.Polish(context.Background(), model.AIConfig{
		BaseURL: server.URL,
		Model:   "m",
		APIKey:  "k",
	}, &model.Week{ID: "2026-W45", Start: "2026-11-02", End: "2026-11-08"}, "report")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateConfig(t *testing.T) {
	cases := []model.AIConfig{
		{},
		{BaseURL: "not-a-url", Model: "m", APIKey: "k"},
		{BaseURL: "https://example.com", Model: "", APIKey: "k"},
		{BaseURL: "https://example.com", Model: "m", APIKey: ""},
	}
	for _, cfg := range cases {
		if err := ValidateConfig(cfg); err == nil {
			t.Fatalf("expected error for %+v", cfg)
		}
	}
	if err := ValidateConfig(model.AIConfig{BaseURL: "https://example.com/v1", Model: "m", APIKey: "k"}); err != nil {
		t.Fatal(err)
	}
}

func TestWeeklyPrompt(t *testing.T) {
	week := &model.Week{ID: "2026-W45", Start: "2026-11-02", End: "2026-11-08", Tasks: []*model.Task{
		{ID: "a", Text: "完成事项", Status: model.StatusDone, CompletedAt: "2026-11-05T10:00:00+08:00", Children: []*model.Task{}},
	}}
	prompt := WeeklyPrompt(week, "本地汇总")
	for _, want := range []string{"2026-11-02", "2026-11-08", "完成事项", "本地汇总"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
}
