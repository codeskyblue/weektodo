package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/webdav/webdavtest"
)

func newDAV(t *testing.T) *webdavtest.Server {
	t.Helper()
	dav := webdavtest.New()
	t.Cleanup(dav.Close)
	return dav
}

func saveDAVSettings(t *testing.T, ts *httptest.Server, davURL, remoteDir, user, pass string) {
	t.Helper()
	body := map[string]any{
		"webdav": map[string]any{
			"url":       davURL,
			"remoteDir": remoteDir,
		},
	}
	if user != "" {
		webdav := body["webdav"].(map[string]any)
		webdav["username"] = user
		webdav["password"] = pass
	}
	status, raw := request(t, ts, http.MethodPut, "/api/settings", body)
	if status != http.StatusOK {
		t.Fatalf("save settings: %d %s", status, raw)
	}
}

func saveWeek(t *testing.T, ts *httptest.Server, weekID string, tasks []map[string]any) {
	t.Helper()
	status, raw := request(t, ts, http.MethodPut, "/api/weeks/"+weekID, map[string]any{"tasks": tasks})
	if status != http.StatusOK {
		t.Fatalf("seed week %s: %d %s", weekID, status, raw)
	}
}

func remoteWeek(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var document modelData
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode remote document: %v", err)
	}
	if len(document.Weeks) == 0 {
		return nil
	}
	return document.Weeks[0]
}

// modelData mirrors the parts of model.Data used by these tests.
type modelData struct {
	Weeks []map[string]any `json:"weeks"`
}

func TestWebDAVSettingsAreRedacted(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"webdav": map[string]any{
			"url":       "https://dav.example.com/dav",
			"username":  "alice",
			"password":  "secret-token",
			"remoteDir": "/weektodo/",
			"autoSync":  true,
		},
	})
	if status != http.StatusOK {
		t.Fatalf("save webdav: %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodGet, "/api/settings", nil)
	if status != http.StatusOK {
		t.Fatalf("get settings: %d %s", status, raw)
	}
	if strings.Contains(string(raw), "secret-token") {
		t.Fatalf("settings leaked the password: %s", raw)
	}
	settings := decodeMap(t, raw)
	webdav := settings["webdav"].(map[string]any)
	if webdav["url"] != "https://dav.example.com/dav" ||
		webdav["username"] != "alice" ||
		webdav["remoteDir"] != "/weektodo" ||
		webdav["hasPassword"] != true ||
		webdav["autoSync"] != true ||
		webdav["passwordHint"] != "••••oken" {
		t.Fatalf("webdav = %+v", webdav)
	}

	// Saving without a password keeps the stored one.
	status, raw = request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"webdav": map[string]any{"username": "bob", "password": ""},
	})
	if status != http.StatusOK {
		t.Fatalf("save webdav again: %d %s", status, raw)
	}
	webdav = decodeMap(t, raw)["webdav"].(map[string]any)
	if webdav["username"] != "bob" || webdav["hasPassword"] != true || webdav["autoSync"] != true {
		t.Fatalf("webdav after keep = %+v", webdav)
	}

	// Invalid URLs are rejected.
	status, _ = request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"webdav": map[string]any{"url": "ftp://dav.example.com"},
	})
	if status != http.StatusBadRequest {
		t.Fatalf("invalid url status = %d", status)
	}
}

func TestWebDAVTestEndpoint(t *testing.T) {
	dav := newDAV(t)
	dav.Auth("alice", "secret")
	dav.Mkdir("/backup")

	ts, _ := newTestServer(t)
	saveDAVSettings(t, ts, dav.URL(), "backup", "alice", "secret")
	status, raw := request(t, ts, http.MethodPost, "/api/webdav/test", map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("test webdav: %d %s", status, raw)
	}

	status, raw = request(t, ts, http.MethodPost, "/api/webdav/test", map[string]any{"password": "wrong"})
	if status != http.StatusBadGateway {
		t.Fatalf("wrong password status = %d %s", status, raw)
	}
	if !strings.Contains(string(raw), "认证失败") {
		t.Fatalf("expected localized auth error: %s", raw)
	}
}

func TestWebDAVSyncMergesBothSides(t *testing.T) {
	dav := newDAV(t)
	dav.SetFile("/backup/weektodo.json", []byte(`{"version":1,"weeks":[{"id":"2026-W45","start":"2026-11-02","end":"2026-11-08","tasks":[{"id":"remote-1","text":"远端任务","status":"todo","updatedAt":"2026-11-05T11:00:00+08:00","children":[]}]}]}`))

	ts, _ := newTestServer(t)
	saveDAVSettings(t, ts, dav.URL(), "backup", "", "")
	saveWeek(t, ts, "2026-W45", []map[string]any{{
		"id":        "local-1",
		"text":      "本地任务",
		"status":    "todo",
		"updatedAt": "2026-11-05T10:00:00+08:00",
		"children":  []any{},
	}})

	status, raw := request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("sync: %d %s", status, raw)
	}
	result := decodeMap(t, raw)
	if result["weeks"] != float64(1) || result["tasks"] != float64(2) || result["uploaded"] != true {
		t.Fatalf("sync result = %+v", result)
	}
	if result["remotePath"] != "backup/weektodo.json" {
		t.Fatalf("remotePath = %v", result["remotePath"])
	}

	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45", nil)
	if status != http.StatusOK {
		t.Fatalf("get week: %d %s", status, raw)
	}
	tasks := decodeMap(t, raw)["week"].(map[string]any)["tasks"].([]any)
	ids := map[string]bool{}
	for _, item := range tasks {
		ids[item.(map[string]any)["id"].(string)] = true
	}
	if !ids["local-1"] || !ids["remote-1"] {
		t.Fatalf("local ids = %+v", ids)
	}

	remote, ok := dav.File("/backup/weektodo.json")
	if !ok {
		t.Fatal("remote file missing")
	}
	week := remoteWeek(t, remote)
	if week == nil || len(week["tasks"].([]any)) != 2 {
		t.Fatalf("remote week = %+v", week)
	}
}

func TestWebDAVSyncUploadsWhenRemoteMissing(t *testing.T) {
	dav := newDAV(t)
	ts, _ := newTestServer(t)
	saveDAVSettings(t, ts, dav.URL(), "backup", "", "")
	saveWeek(t, ts, "2026-W45", []map[string]any{{
		"id":       "local-1",
		"text":     "本地任务",
		"status":   "todo",
		"children": []any{},
	}})

	status, raw := request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("sync: %d %s", status, raw)
	}
	if !dav.HasDir("/backup") {
		t.Fatalf("remote directory was not created: %v", dav.Requests())
	}
	remote, ok := dav.File("/backup/weektodo.json")
	if !ok {
		t.Fatal("remote file missing")
	}
	week := remoteWeek(t, remote)
	if week == nil || len(week["tasks"].([]any)) != 1 {
		t.Fatalf("remote week = %+v", week)
	}
}

func TestWebDAVSyncPullsRemoteOnlyWeek(t *testing.T) {
	dav := newDAV(t)
	dav.SetFile("/backup/weektodo.json", []byte(`{"version":1,"weeks":[{"id":"2026-W44","start":"2026-10-26","end":"2026-11-01","tasks":[{"id":"remote-only","text":"只有远端","status":"todo","children":[]}]}]}`))

	ts, _ := newTestServer(t)
	saveDAVSettings(t, ts, dav.URL(), "backup", "", "")
	status, raw := request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("sync: %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W44", nil)
	if status != http.StatusOK {
		t.Fatalf("remote-only week was not pulled: %d %s", status, raw)
	}
	if len(decodeMap(t, raw)["week"].(map[string]any)["tasks"].([]any)) != 1 {
		t.Fatalf("remote-only week = %s", raw)
	}
}

func TestWebDAVSyncUsesNewestTask(t *testing.T) {
	cases := []struct {
		name      string
		localText string
		localAt   string
		remote    string
		want      string
	}{
		{
			name:      "remote newer",
			localText: "本地旧",
			localAt:   "2026-11-05T10:00:00+08:00",
			remote:    "远端新",
			want:      "远端新",
		},
		{
			name:      "local newer",
			localText: "本地新",
			localAt:   "2026-11-05T12:00:00+08:00",
			remote:    "远端旧",
			want:      "本地新",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dav := newDAV(t)
			dav.SetFile("/backup/weektodo.json", []byte(`{"version":1,"weeks":[{"id":"2026-W45","start":"2026-11-02","end":"2026-11-08","tasks":[{"id":"same","text":"`+tc.remote+`","status":"todo","createdAt":"2026-11-05T09:00:00+08:00","updatedAt":"2026-11-05T11:00:00+08:00","children":[]}]}]}`))

			ts, _ := newTestServer(t)
			saveDAVSettings(t, ts, dav.URL(), "backup", "", "")
			saveWeek(t, ts, "2026-W45", []map[string]any{{
				"id":        "same",
				"text":      tc.localText,
				"status":    "todo",
				"createdAt": "2026-11-05T09:00:00+08:00",
				"updatedAt": tc.localAt,
				"children":  []any{},
			}})

			status, raw := request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
			if status != http.StatusOK {
				t.Fatalf("sync: %d %s", status, raw)
			}
			status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45", nil)
			if status != http.StatusOK {
				t.Fatalf("get week: %d %s", status, raw)
			}
			tasks := decodeMap(t, raw)["week"].(map[string]any)["tasks"].([]any)
			if got := tasks[0].(map[string]any)["text"]; got != tc.want {
				t.Fatalf("task text = %v, want %s", got, tc.want)
			}
		})
	}
}

func TestWebDAVSyncRejectsBrokenRemote(t *testing.T) {
	dav := newDAV(t)
	dav.SetFile("/weektodo.json", []byte("{not json"))

	ts, _ := newTestServer(t)
	saveDAVSettings(t, ts, dav.URL(), "", "", "")
	status, raw := request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("sync broken remote = %d %s", status, raw)
	}
	stored, ok := dav.File("/weektodo.json")
	if !ok || string(stored) != "{not json" {
		t.Fatal("remote file must not be overwritten when parsing fails")
	}
}

func TestWebDAVSyncKeepsCategoriesSeparate(t *testing.T) {
	dav := newDAV(t)
	ts, _ := newTestServer(t)
	oss := createCategory(t, ts, "开源")
	saveDAVSettings(t, ts, dav.URL(), "backup", "", "")
	saveWeek(t, ts, "2026-W45", []map[string]any{{
		"id": "work", "text": "工作", "status": "todo", "children": []any{},
	}})
	putTasks(t, ts, oss, "2026-W45", []map[string]any{{
		"id": "oss", "text": "开源", "status": "todo", "children": []any{},
	}})

	status, raw := request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("sync: %d %s", status, raw)
	}
	result := decodeMap(t, raw)
	if result["categories"] != float64(2) || result["weeks"] != float64(2) || result["tasks"] != float64(2) {
		t.Fatalf("sync result = %+v", result)
	}

	stored, ok := dav.File("/backup/weektodo.json")
	if !ok {
		t.Fatal("remote file missing")
	}
	var document struct {
		Categories []map[string]any `json:"categories"`
		Weeks      []map[string]any `json:"weeks"`
	}
	if err := json.Unmarshal(stored, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Categories) != 2 || len(document.Weeks) != 2 {
		t.Fatalf("remote document = %+v", document)
	}
	categories := map[string]bool{}
	for _, week := range document.Weeks {
		categories[week["category"].(string)] = true
	}
	if !categories[model.DefaultCategoryID] || !categories[oss] {
		t.Fatalf("remote week categories = %+v", categories)
	}
}
