package webdav

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/webdav/webdavtest"
)

func newTestClient(t *testing.T, server *webdavtest.Server, remoteDir string) *Client {
	t.Helper()
	return New(Config{
		URL:       server.URL(),
		Username:  "alice",
		Password:  "secret",
		RemoteDir: remoteDir,
	}, i18n.Zh)
}

func hasRequest(server *webdavtest.Server, want string) bool {
	for _, request := range server.Requests() {
		if request == want {
			return true
		}
	}
	return false
}

func TestValidate(t *testing.T) {
	cases := []struct {
		cfg Config
		ok  bool
	}{
		{Config{URL: "https://dav.example.com"}, true},
		{Config{URL: ""}, false},
		{Config{URL: "ftp://dav.example.com"}, false},
		{Config{URL: "not a url"}, false},
		{Config{URL: "https://dav.example.com", RemoteDir: "a/../b"}, false},
	}
	for _, tc := range cases {
		err := Validate(tc.cfg, i18n.Zh)
		if tc.ok && err != nil {
			t.Fatalf("Validate(%+v) = %v", tc.cfg, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("Validate(%+v) = nil", tc.cfg)
		}
	}
}

func TestTestConnection(t *testing.T) {
	server := webdavtest.New()
	defer server.Close()
	server.Auth("alice", "secret")
	server.Mkdir("/notes")

	client := newTestClient(t, server, "notes")
	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !hasRequest(server, "PROPFIND /notes") {
		t.Fatalf("requests = %v", server.Requests())
	}
}

func TestTestFallsBackToBaseURLWhenDirMissing(t *testing.T) {
	server := webdavtest.New()
	defer server.Close()

	client := newTestClient(t, server, "notes/new")
	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !hasRequest(server, "PROPFIND /notes/new") {
		t.Fatalf("expected directory probe, got:\n%v", server.Requests())
	}
	if !hasRequest(server, "PROPFIND /") {
		t.Fatalf("expected base URL probe, got:\n%v", server.Requests())
	}
}

func TestTestAuthFailure(t *testing.T) {
	server := webdavtest.New()
	defer server.Close()
	server.Auth("alice", "secret")

	client := New(Config{URL: server.URL(), Username: "alice", Password: "wrong"}, i18n.En)
	err := client.Test(context.Background())
	if err == nil || !strings.Contains(err.Error(), "authentication failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestTestFallsBackToOptions(t *testing.T) {
	server := webdavtest.New()
	defer server.Close()
	server.PropfindStatus(http.StatusMethodNotAllowed)

	client := New(Config{URL: server.URL()}, i18n.Zh)
	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if !hasRequest(server, "OPTIONS /") {
		t.Fatalf("expected OPTIONS fallback, got:\n%v", server.Requests())
	}
}

func TestGetMissingAndPutCreatesDir(t *testing.T) {
	server := webdavtest.New()
	defer server.Close()

	client := newTestClient(t, server, "notes/2026")
	if _, exists, err := client.Get(context.Background(), "weektodo.json"); err != nil || exists {
		t.Fatalf("Get missing = %v, %v", exists, err)
	}
	if err := client.Put(context.Background(), "weektodo.json", []byte(`{"version":1}`)); err != nil {
		t.Fatalf("Put: %v", err)
	}
	data, exists, err := client.Get(context.Background(), "weektodo.json")
	if err != nil || !exists {
		t.Fatalf("Get after put = %v, %v", exists, err)
	}
	if string(data) != `{"version":1}` {
		t.Fatalf("data = %s", data)
	}
	if !server.HasDir("/notes") || !server.HasDir("/notes/2026") {
		t.Fatalf("directories were not created: %v", server.Requests())
	}
	stored, ok := server.File("/notes/2026/weektodo.json")
	if !ok || string(stored) != `{"version":1}` {
		t.Fatalf("stored file = %s, %v", stored, ok)
	}
}

func TestEndpointEscapesSegmentsOnce(t *testing.T) {
	client := New(Config{URL: "https://dav.example.com/dav root", RemoteDir: "周报/子目录"}, i18n.Zh)
	endpoint, err := client.endpoint("weektodo.json")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(endpoint, "%25") {
		t.Fatalf("double escaped endpoint: %s", endpoint)
	}
	for _, want := range []string{"dav%20root", "%E5%91%A8%E6%8A%A5", "%E5%AD%90%E7%9B%AE%E5%BD%95", "weektodo.json"} {
		if !strings.Contains(endpoint, want) {
			t.Fatalf("endpoint %s missing %s", endpoint, want)
		}
	}
}
