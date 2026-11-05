package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/uiautodev/weektodo/internal/ai"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/store"
	"github.com/uiautodev/weektodo/internal/update"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := New(st, ai.New(), Options{Version: "test", Commit: "abc1234"})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, st
}

func request(t *testing.T, ts *httptest.Server, method, path string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, ts.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, raw
}

func decodeMap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return value
}

func TestMetaAndHealth(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodGet, "/api/health", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	status, raw = request(t, ts, http.MethodGet, "/api/meta", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	meta := decodeMap(t, raw)
	if meta["version"] != "test" || meta["commit"] != "abc1234" || meta["currentWeekId"] == "" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestWeekLifecycle(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodPost, "/api/weeks/2026-W45", nil)
	if status != http.StatusCreated {
		t.Fatalf("create status = %d %s", status, raw)
	}
	tasks := []*model.Task{
		{ID: "p", Text: "parent", Status: model.StatusTodo, Day: "2026-11-05", Children: []*model.Task{
			{ID: "a", Text: "child a", Status: model.StatusDone, Children: []*model.Task{}},
			{ID: "b", Text: "child b", Status: model.StatusTodo, Children: []*model.Task{}},
		}},
	}
	status, raw = request(t, ts, http.MethodPut, "/api/weeks/2026-W45", map[string]any{"tasks": tasks})
	if status != http.StatusOK {
		t.Fatalf("save status = %d %s", status, raw)
	}
	saved := decodeMap(t, raw)["week"].(map[string]any)
	savedTasks := saved["tasks"].([]any)
	parent := savedTasks[0].(map[string]any)
	if parent["status"] != "todo" {
		t.Fatalf("parent status = %v", parent["status"])
	}

	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45/report?mode=brief", nil)
	if status != http.StatusOK {
		t.Fatalf("report status = %d %s", status, raw)
	}
	if markdown, _ := decodeMap(t, raw)["markdown"].(string); !strings.Contains(markdown, "周报") {
		t.Fatalf("markdown = %s", markdown)
	}

	if status, _ := request(t, ts, http.MethodGet, "/api/weeks/not-a-week", nil); status != http.StatusBadRequest {
		t.Fatalf("invalid week status = %d", status)
	}
	if status, _ := request(t, ts, http.MethodGet, "/api/weeks/2025-W01", nil); status != http.StatusNotFound {
		t.Fatalf("missing week status = %d", status)
	}
}

func TestCarryOverAPI(t *testing.T) {
	ts, _ := newTestServer(t)
	for _, id := range []string{"2026-W44", "2026-W45"} {
		if status, raw := request(t, ts, http.MethodPost, "/api/weeks/"+id, nil); status != http.StatusCreated {
			t.Fatalf("create %s: %d %s", id, status, raw)
		}
	}
	sourceTasks := []*model.Task{
		{ID: "s1", Text: "unfinished", Status: model.StatusTodo, Children: []*model.Task{
			{ID: "s2", Text: "done child", Status: model.StatusDone, Children: []*model.Task{}},
			{ID: "s3", Text: "todo child", Status: model.StatusTodo, Children: []*model.Task{}},
		}},
		{ID: "s4", Text: "dropped", Status: model.StatusDropped, Children: []*model.Task{}},
	}
	if status, raw := request(t, ts, http.MethodPut, "/api/weeks/2026-W44", map[string]any{"tasks": sourceTasks}); status != http.StatusOK {
		t.Fatalf("save source: %d %s", status, raw)
	}

	status, raw := request(t, ts, http.MethodGet, "/api/weeks/2026-W45/carry-over", nil)
	if status != http.StatusOK {
		t.Fatalf("preview status = %d %s", status, raw)
	}
	preview := decodeMap(t, raw)
	if preview["count"].(float64) != 2 {
		t.Fatalf("preview = %+v", preview)
	}

	status, raw = request(t, ts, http.MethodPost, "/api/weeks/2026-W45/carry-over", nil)
	if status != http.StatusOK {
		t.Fatalf("carry status = %d %s", status, raw)
	}
	result := decodeMap(t, raw)
	if result["created"].(float64) != 2 {
		t.Fatalf("result = %+v", result)
	}
	week := result["week"].(map[string]any)
	if len(week["tasks"].([]any)) != 1 {
		t.Fatalf("target tasks = %+v", week["tasks"])
	}

	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45/carry-over", nil)
	if status != http.StatusOK {
		t.Fatalf("second preview status = %d", status)
	}
	if count := decodeMap(t, raw)["count"].(float64); count != 0 {
		t.Fatalf("second preview count = %v", count)
	}
}

func TestSettingsMaskAndClearKey(t *testing.T) {
	ts, _ := newTestServer(t)
	aiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failed", http.StatusInternalServerError)
	}))
	defer aiServer.Close()
	if status, raw := request(t, ts, http.MethodPost, "/api/weeks/2026-W45", nil); status != http.StatusCreated {
		t.Fatalf("create: %d %s", status, raw)
	}
	if status, raw := request(t, ts, http.MethodPost, "/api/weeks/2026-W45/report/polish", map[string]any{"mode": "brief"}); status != http.StatusBadRequest {
		t.Fatalf("polish unconfigured status = %d %s", status, raw)
	}
	status, raw := request(t, ts, http.MethodGet, "/api/settings", nil)
	if status != http.StatusOK {
		t.Fatalf("get settings: %d %s", status, raw)
	}
	if decodeMap(t, raw)["ai"].(map[string]any)["configured"].(bool) {
		t.Fatal("should not be configured")
	}

	status, raw = request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"ai": map[string]any{
			"baseUrl": " " + aiServer.URL + "/v1/ ",
			"model":   " model ",
			"apiKey":  "secret-key-1234",
		},
	})
	if status != http.StatusOK {
		t.Fatalf("save settings: %d %s", status, raw)
	}
	if strings.Contains(string(raw), "secret-key-1234") {
		t.Fatalf("response leaked the API key: %s", raw)
	}
	aiSettings := decodeMap(t, raw)["ai"].(map[string]any)
	if !aiSettings["configured"].(bool) || aiSettings["apiKeyHint"] != "••••1234" {
		t.Fatalf("ai = %+v", aiSettings)
	}

	if status, raw := request(t, ts, http.MethodPost, "/api/weeks/2026-W45/report/polish", map[string]any{"mode": "brief"}); status != http.StatusBadGateway {
		t.Fatalf("polish with failing endpoint status = %d %s", status, raw)
	}

	status, raw = request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"ai": map[string]any{"clearApiKey": true},
	})
	if status != http.StatusOK {
		t.Fatalf("clear key: %d %s", status, raw)
	}
	if decodeMap(t, raw)["ai"].(map[string]any)["configured"].(bool) {
		t.Fatal("key should be cleared")
	}
}

func TestStaticAndUnknownAPI(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodGet, "/", nil)
	if status != http.StatusOK || !strings.Contains(string(raw), "<!doctype html") {
		t.Fatalf("index status = %d len=%d", status, len(raw))
	}
	if status, _ := request(t, ts, http.MethodGet, "/api/unknown", nil); status != http.StatusNotFound {
		t.Fatalf("unknown api status = %d", status)
	}
}

func TestUpdateEndpoints(t *testing.T) {
	version := "0.2.0"
	manifestSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/update.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version": version,
			"tag":     "v" + version,
			"notes":   "update notes",
			"assets":  []any{},
		})
	}))
	defer manifestSrv.Close()

	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	executable := t.TempDir() + "/weektodo"
	updater := update.New(update.Options{
		ManifestURL: manifestSrv.URL + "/update.json",
		LatestURL:   manifestSrv.URL + "/latest",
		FeedURL:     manifestSrv.URL + "/feed",
		APIURL:      manifestSrv.URL + "/release",
		ReleaseBase: manifestSrv.URL,
		DataDir:     t.TempDir(),
		Executable:  executable,
		GOOS:        "darwin",
		GOARCH:      "arm64",
		HTTPClient:  manifestSrv.Client(),
	})
	server := New(st, ai.New(), Options{Version: "0.1.0", Updater: updater})
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	status, raw := request(t, ts, http.MethodGet, "/api/update/check", nil)
	if status != http.StatusOK {
		t.Fatalf("check status = %d %s", status, raw)
	}
	info := decodeMap(t, raw)
	if info["available"] != true || info["latest"] != version || info["source"] != "github" {
		t.Fatalf("info = %+v", info)
	}

	status, raw = request(t, ts, http.MethodGet, "/api/update/status", nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d %s", status, raw)
	}
	if active := decodeMap(t, raw)["active"]; active != false {
		t.Fatalf("progress = %+v", active)
	}

	upToDate := update.New(update.Options{
		ManifestURL: manifestSrv.URL + "/update.json",
		LatestURL:   manifestSrv.URL + "/latest",
		FeedURL:     manifestSrv.URL + "/feed",
		APIURL:      manifestSrv.URL + "/release",
		ReleaseBase: manifestSrv.URL,
		DataDir:     t.TempDir(),
		Executable:  executable,
		GOOS:        "darwin",
		GOARCH:      "arm64",
		HTTPClient:  manifestSrv.Client(),
	})
	server2 := New(st, ai.New(), Options{Version: version, Updater: upToDate})
	ts2 := httptest.NewServer(server2.Handler())
	defer ts2.Close()
	if status, _ := request(t, ts2, http.MethodPost, "/api/update/apply", map[string]any{}); status != http.StatusConflict {
		t.Fatalf("apply up-to-date status = %d", status)
	}
}

func TestAIModelsEndpoint(t *testing.T) {
	modelsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			http.Error(w, "bad key", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{{"id": "model-b"}, {"id": "model-a"}},
		})
	}))
	defer modelsSrv.Close()

	ts, st := newTestServer(t)
	if status, _ := request(t, ts, http.MethodPost, "/api/settings/ai/models", map[string]any{}); status != http.StatusBadRequest {
		t.Fatalf("unconfigured status = %d", status)
	}
	if err := st.UpdateSettings(model.Settings{AI: model.AIConfig{
		BaseURL: modelsSrv.URL + "/v1",
		APIKey:  "test-key",
		Model:   "model-a",
	}}); err != nil {
		t.Fatal(err)
	}
	status, raw := request(t, ts, http.MethodPost, "/api/settings/ai/models", map[string]any{
		"baseUrl": modelsSrv.URL + "/v1",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d %s", status, raw)
	}
	models := decodeMap(t, raw)["models"].([]any)
	if len(models) != 2 || models[0] != "model-a" || models[1] != "model-b" {
		t.Fatalf("models = %+v", models)
	}

}
