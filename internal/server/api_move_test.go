package server

import (
	"net/http"
	"strings"
	"testing"
)

func TestSettingsWeekSpan(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodGet, "/api/settings", nil)
	if status != http.StatusOK {
		t.Fatalf("get settings: %d %s", status, raw)
	}
	if got := decodeMap(t, raw)["weekSpan"]; got != float64(1) {
		t.Fatalf("default weekSpan = %v", got)
	}

	for _, span := range []int{2, 3, 1} {
		status, raw = request(t, ts, http.MethodPut, "/api/settings", map[string]any{"weekSpan": span})
		if status != http.StatusOK {
			t.Fatalf("save weekSpan %d: %d %s", span, status, raw)
		}
		if got := decodeMap(t, raw)["weekSpan"]; got != float64(span) {
			t.Fatalf("weekSpan = %v, want %d", got, span)
		}
	}

	status, _ = request(t, ts, http.MethodPut, "/api/settings", map[string]any{"weekSpan": 4})
	if status != http.StatusBadRequest {
		t.Fatalf("invalid weekSpan status = %d", status)
	}
}

func TestMoveTaskEndpoint(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodPut, "/api/weeks/2026-W45", map[string]any{
		"tasks": []map[string]any{{
			"id":     "parent",
			"text":   "parent",
			"status": "todo",
			"day":    "2026-11-05",
			"children": []map[string]any{{
				"id":       "child",
				"text":     "child",
				"status":   "todo",
				"day":      "2026-11-06",
				"children": []any{},
			}},
		}},
	})
	if status != http.StatusOK {
		t.Fatalf("seed week: %d %s", status, raw)
	}

	status, raw = request(t, ts, http.MethodPost, "/api/weeks/2026-W45/move", map[string]any{
		"taskId":   "parent",
		"toWeekId": "2026-W46",
	})
	if status != http.StatusOK {
		t.Fatalf("move: %d %s", status, raw)
	}
	result := decodeMap(t, raw)
	from := result["from"].(map[string]any)
	to := result["to"].(map[string]any)
	if len(from["tasks"].([]any)) != 0 {
		t.Fatalf("from = %+v", from)
	}
	tasks := to["tasks"].([]any)
	if len(tasks) != 1 {
		t.Fatalf("to = %+v", to)
	}
	moved := tasks[0].(map[string]any)
	if moved["id"] != "parent" || moved["day"] != nil {
		t.Fatalf("moved = %+v", moved)
	}
	child := moved["children"].([]any)[0].(map[string]any)
	if child["day"] != nil {
		t.Fatalf("child = %+v", child)
	}

	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W46", nil)
	if status != http.StatusOK {
		t.Fatalf("target week: %d %s", status, raw)
	}
	if len(decodeMap(t, raw)["week"].(map[string]any)["tasks"].([]any)) != 1 {
		t.Fatalf("target not persisted: %s", raw)
	}

	status, _ = request(t, ts, http.MethodPost, "/api/weeks/2026-W46/move", map[string]any{
		"taskId":   "parent",
		"toWeekId": "2026-W46",
	})
	if status != http.StatusBadRequest {
		t.Fatalf("same week status = %d", status)
	}
	status, _ = request(t, ts, http.MethodPost, "/api/weeks/2026-W46/move", map[string]any{
		"taskId":   "missing",
		"toWeekId": "2026-W45",
	})
	if status != http.StatusNotFound {
		t.Fatalf("missing task status = %d", status)
	}
}

func TestLocalizedMoveError(t *testing.T) {
	ts, _ := newTestServer(t)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/weeks/2026-W45/move",
		strings.NewReader(`{"taskId":"a","toWeekId":"2026-W46"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestOptionalMissingWeek(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodGet, "/api/weeks/2026-W50?optional=1", nil)
	if status != http.StatusOK {
		t.Fatalf("optional missing week: %d %s", status, raw)
	}
	if week := decodeMap(t, raw)["week"]; week != nil {
		t.Fatalf("week = %+v", week)
	}
	status, _ = request(t, ts, http.MethodGet, "/api/weeks/2026-W50", nil)
	if status != http.StatusNotFound {
		t.Fatalf("plain missing week = %d", status)
	}
}
