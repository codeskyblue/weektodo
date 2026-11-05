package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type testDAV struct {
	mu    sync.Mutex
	files map[string][]byte
	dirs  map[string]bool
	user  string
	pass  string
}

func newTestDAV() *testDAV {
	return &testDAV{
		files: map[string][]byte{},
		dirs:  map[string]bool{"/": true},
	}
}

func (d *testDAV) file(path string) []byte {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.files[path]
}

func (d *testDAV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d.user != "" {
		user, pass, ok := r.BasicAuth()
		if !ok || user != d.user || pass != d.pass {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}
	switch r.Method {
	case "PROPFIND":
		w.WriteHeader(http.StatusMultiStatus)
	case "MKCOL":
		d.mu.Lock()
		defer d.mu.Unlock()
		if d.dirs[r.URL.Path] {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		d.dirs[r.URL.Path] = true
		w.WriteHeader(http.StatusCreated)
	case http.MethodPut:
		d.mu.Lock()
		defer d.mu.Unlock()
		parent := r.URL.Path[:strings.LastIndex(r.URL.Path, "/")]
		if !d.dirs[parent] {
			w.WriteHeader(http.StatusConflict)
			return
		}
		body, _ := io.ReadAll(r.Body)
		d.files[r.URL.Path] = body
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		d.mu.Lock()
		body, ok := d.files[r.URL.Path]
		d.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func TestWebDAVSettingsAreRedacted(t *testing.T) {
	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"webdav": map[string]any{
			"url":       "https://dav.example.com/dav",
			"username":  "alice",
			"password":  "secret-token",
			"remoteDir": "/weektodo/",
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
		webdav["remoteDir"] != "weektodo" ||
		webdav["hasPassword"] != true ||
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
	if webdav["username"] != "bob" || webdav["hasPassword"] != true {
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
	dav := newTestDAV()
	dav.user = "alice"
	dav.pass = "secret"
	davServer := httptest.NewServer(dav)
	defer davServer.Close()

	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"webdav": map[string]any{
			"url":       davServer.URL + "/dav",
			"username":  "alice",
			"password":  "secret",
			"remoteDir": "backup",
		},
	})
	if status != http.StatusOK {
		t.Fatalf("save settings: %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodPost, "/api/webdav/test", map[string]any{})
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
	dav := newTestDAV()
	dav.dirs["/dav"] = true
	remote := `{"version":1,"weeks":[{"id":"2026-W45","start":"2026-11-02","end":"2026-11-08","tasks":[{"id":"remote-1","text":"远端任务","status":"todo","updatedAt":"2026-11-05T11:00:00+08:00","children":[]}]}]}`
	dav.files["/dav/backup/weektodo.json"] = []byte(remote)
	davServer := httptest.NewServer(dav)
	defer davServer.Close()

	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"webdav": map[string]any{
			"url":       davServer.URL + "/dav",
			"remoteDir": "backup",
		},
	})
	if status != http.StatusOK {
		t.Fatalf("save settings: %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodPut, "/api/weeks/2026-W45", map[string]any{
		"tasks": []map[string]any{{
			"id":        "local-1",
			"text":      "本地任务",
			"status":    "todo",
			"updatedAt": "2026-11-05T10:00:00+08:00",
			"children":  []any{},
		}},
	})
	if status != http.StatusOK {
		t.Fatalf("seed week: %d %s", status, raw)
	}

	status, raw = request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("sync: %d %s", status, raw)
	}
	result := decodeMap(t, raw)
	if result["weeks"] != float64(1) || result["tasks"] != float64(2) || result["uploaded"] != true {
		t.Fatalf("sync result = %+v", result)
	}

	status, raw = request(t, ts, http.MethodGet, "/api/weeks/2026-W45", nil)
	if status != http.StatusOK {
		t.Fatalf("get week: %d %s", status, raw)
	}
	week := decodeMap(t, raw)["week"].(map[string]any)
	tasks := week["tasks"].([]any)
	if len(tasks) != 2 {
		t.Fatalf("local tasks = %+v", tasks)
	}
	ids := map[string]bool{}
	for _, item := range tasks {
		ids[item.(map[string]any)["id"].(string)] = true
	}
	if !ids["local-1"] || !ids["remote-1"] {
		t.Fatalf("local ids = %+v", ids)
	}

	var uploaded struct {
		Weeks []struct {
			Tasks []struct {
				ID string `json:"id"`
			} `json:"tasks"`
		} `json:"weeks"`
	}
	if err := json.Unmarshal(dav.file("/dav/backup/weektodo.json"), &uploaded); err != nil {
		t.Fatalf("remote payload: %v", err)
	}
	if len(uploaded.Weeks) != 1 || len(uploaded.Weeks[0].Tasks) != 2 {
		t.Fatalf("remote payload = %+v", uploaded)
	}
}

func TestWebDAVSyncRejectsBrokenRemote(t *testing.T) {
	dav := newTestDAV()
	dav.dirs["/dav"] = true
	dav.files["/dav/weektodo.json"] = []byte("{not json")
	davServer := httptest.NewServer(dav)
	defer davServer.Close()

	ts, _ := newTestServer(t)
	status, raw := request(t, ts, http.MethodPut, "/api/settings", map[string]any{
		"webdav": map[string]any{"url": davServer.URL + "/dav"},
	})
	if status != http.StatusOK {
		t.Fatalf("save settings: %d %s", status, raw)
	}
	status, raw = request(t, ts, http.MethodPost, "/api/webdav/sync", map[string]any{})
	if status != http.StatusUnprocessableEntity {
		t.Fatalf("sync broken remote = %d %s", status, raw)
	}
	if string(dav.file("/dav/weektodo.json")) != "{not json" {
		t.Fatal("remote file must not be overwritten when parsing fails")
	}
}
