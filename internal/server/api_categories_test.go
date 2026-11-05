package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func createCategory(t *testing.T, ts *httptest.Server, name string) string {
	t.Helper()
	status, raw := request(t, ts, http.MethodPost, "/api/categories", map[string]any{
		"name":  name,
		"color": "#8b5cf6",
	})
	if status != http.StatusCreated {
		t.Fatalf("create category: %d %s", status, raw)
	}
	return decodeMap(t, raw)["category"].(map[string]any)["id"].(string)
}

func TestCategoryCRUD(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodGet, "/api/categories", nil)
	if status != http.StatusOK {
		t.Fatalf("list categories: %d %s", status, raw)
	}
	if got := len(decodeMap(t, raw)["categories"].([]any)); got != 1 {
		t.Fatalf("default categories = %d", got)
	}

	id := createCategory(t, ts, "开源")
	status, raw = request(t, ts, http.MethodPost, "/api/categories", map[string]any{"name": "开源"})
	if status != http.StatusConflict {
		t.Fatalf("duplicate category status = %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodPost, "/api/categories", map[string]any{"name": "  "})
	if status != http.StatusBadRequest {
		t.Fatalf("empty category status = %d %s", status, raw)
	}

	status, raw = request(t, ts, http.MethodPut, "/api/categories/"+id, map[string]any{
		"name":  "Open Source",
		"color": "#22c55e",
	})
	if status != http.StatusOK {
		t.Fatalf("rename category: %d %s", status, raw)
	}
	if got := decodeMap(t, raw)["category"].(map[string]any)["name"]; got != "Open Source" {
		t.Fatalf("renamed category = %v", got)
	}

	status, raw = request(t, ts, http.MethodPut, "/api/categories/"+id, map[string]any{"archived": true})
	if status != http.StatusOK {
		t.Fatalf("archive category: %d %s", status, raw)
	}
	status, _ = request(t, ts, http.MethodGet, "/api/weeks?category="+id, nil)
	if status != http.StatusNotFound {
		t.Fatalf("archived category weeks status = %d", status)
	}

	status, raw = request(t, ts, http.MethodPut, "/api/categories/"+id, map[string]any{"archived": false})
	if status != http.StatusOK {
		t.Fatalf("restore category: %d %s", status, raw)
	}
}

func TestCategoryIsolation(t *testing.T) {
	ts, _ := newTestServer(t)
	oss := createCategory(t, ts, "开源")

	putTasks(t, ts, "", "2026-W44", []map[string]any{{
		"id": "work-prev", "text": "工作上周", "status": "todo", "children": []any{},
	}})
	putTasks(t, ts, "", "2026-W45", []map[string]any{{
		"id": "work", "text": "工作", "status": "todo", "children": []any{},
	}})
	putTasks(t, ts, oss, "2026-W45", []map[string]any{{
		"id":          "oss",
		"text":        "开源",
		"status":      "done",
		"completedAt": "2026-11-05T10:00:00+08:00",
		"children":    []any{},
	}})

	status, raw := request(t, ts, http.MethodGet, "/api/weeks/2026-W45", nil)
	if status != http.StatusOK {
		t.Fatalf("default week: %d %s", status, raw)
	}
	if tasks := decodeMap(t, raw)["week"].(map[string]any)["tasks"].([]any); len(tasks) != 1 ||
		tasks[0].(map[string]any)["id"] != "work" {
		t.Fatalf("default tasks = %+v", tasks)
	}
	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45?category="+oss, nil)
	if status != http.StatusOK {
		t.Fatalf("oss week: %d %s", status, raw)
	}
	if tasks := decodeMap(t, raw)["week"].(map[string]any)["tasks"].([]any); len(tasks) != 1 ||
		tasks[0].(map[string]any)["id"] != "oss" {
		t.Fatalf("oss tasks = %+v", tasks)
	}

	// Carry-over must not pull default-category tasks into the open source space.
	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45/carry-over?category="+oss, nil)
	if status != http.StatusOK {
		t.Fatalf("oss carry preview: %d %s", status, raw)
	}
	if reason := decodeMap(t, raw)["reason"]; reason != "no_previous_week" {
		t.Fatalf("oss carry reason = %v", reason)
	}

	// Reports only include the active category.
	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45/report?mode=brief&category="+oss, nil)
	if status != http.StatusOK {
		t.Fatalf("oss report: %d %s", status, raw)
	}
	markdown := decodeMap(t, raw)["markdown"].(string)
	if strings.Contains(markdown, "工作") || !strings.Contains(markdown, "开源") {
		t.Fatalf("report mixed categories:\n%s", markdown)
	}

	// Moving a task in one category must not touch the other.
	status, raw = request(t, ts, http.MethodPost, "/api/weeks/2026-W45/move?category="+oss, map[string]any{
		"taskId": "oss", "toWeekId": "2026-W46",
	})
	if status != http.StatusOK {
		t.Fatalf("move oss: %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45", nil)
	if status != http.StatusOK || len(decodeMap(t, raw)["week"].(map[string]any)["tasks"].([]any)) != 1 {
		t.Fatalf("default week changed after oss move: %d %s", status, raw)
	}
}

func putTasks(t *testing.T, ts *httptest.Server, category, weekID string, tasks []map[string]any) {
	t.Helper()
	path := "/api/weeks/" + weekID
	if category != "" {
		path += "?category=" + category
	}
	status, raw := request(t, ts, http.MethodPut, path, map[string]any{"tasks": tasks})
	if status != http.StatusOK {
		t.Fatalf("save %s %s: %d %s", category, weekID, status, raw)
	}
}

func TestCategorySettings(t *testing.T) {
	ts, _ := newTestServer(t)
	oss := createCategory(t, ts, "开源")
	status, raw := request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"activeCategory": oss,
		"categoryWeeks":  map[string]string{oss: "2026-W44", "default": "2026-W45"},
	})
	if status != http.StatusOK {
		t.Fatalf("save category state: %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodGet, "/api/settings", nil)
	if status != http.StatusOK {
		t.Fatalf("get settings: %d %s", status, raw)
	}
	settings := decodeMap(t, raw)
	if settings["activeCategory"] != oss {
		t.Fatalf("activeCategory = %v", settings["activeCategory"])
	}
	weeks := settings["categoryWeeks"].(map[string]any)
	if weeks[oss] != "2026-W44" || weeks["default"] != "2026-W45" {
		t.Fatalf("categoryWeeks = %+v", weeks)
	}

	status, _ = request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"activeCategory": "missing",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("missing active category status = %d", status)
	}
}
