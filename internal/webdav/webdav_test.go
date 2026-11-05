package webdav

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/uiautodev/weektodo/internal/i18n"
)

type fakeDAV struct {
	mu            sync.Mutex
	files         map[string][]byte
	dirs          map[string]bool
	user          string
	password      string
	propfindCode  int
	optionsCode   int
	seenPaths     []string
	seenAuthUsers []string
}

func newFakeDAV() *fakeDAV {
	return &fakeDAV{
		files: map[string][]byte{},
		dirs:  map[string]bool{"/": true},
	}
}

func (f *fakeDAV) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.seenPaths = append(f.seenPaths, r.Method+" "+r.URL.Path)
	f.seenAuthUsers = append(f.seenAuthUsers, r.Header.Get("Authorization"))
	f.mu.Unlock()

	if f.user != "" {
		user, password, ok := r.BasicAuth()
		if !ok || user != f.user || password != f.password {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	switch r.Method {
	case "PROPFIND":
		if f.propfindCode != 0 {
			w.WriteHeader(f.propfindCode)
			return
		}
		f.mu.Lock()
		exists := f.dirs[r.URL.Path]
		f.mu.Unlock()
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusMultiStatus)
	case http.MethodOptions:
		if f.optionsCode != 0 {
			w.WriteHeader(f.optionsCode)
			return
		}
		w.WriteHeader(http.StatusOK)
	case "MKCOL":
		f.mu.Lock()
		defer f.mu.Unlock()
		if f.dirs[r.URL.Path] {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		f.dirs[r.URL.Path] = true
		w.WriteHeader(http.StatusCreated)
	case http.MethodPut:
		f.mu.Lock()
		defer f.mu.Unlock()
		parent := r.URL.Path[:strings.LastIndex(r.URL.Path, "/")]
		if !f.dirs[parent] {
			w.WriteHeader(http.StatusConflict)
			return
		}
		body, _ := io.ReadAll(r.Body)
		f.files[r.URL.Path] = body
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		f.mu.Lock()
		data, ok := f.files[r.URL.Path]
		f.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(data)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func newTestClient(t *testing.T, server *httptest.Server, remoteDir string) *Client {
	t.Helper()
	return New(Config{
		URL:       server.URL + "/dav",
		Username:  "alice",
		Password:  "secret",
		RemoteDir: remoteDir,
	}, i18n.Zh)
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
	fake := newFakeDAV()
	fake.dirs["/dav/notes"] = true
	server := httptest.NewServer(fake)
	defer server.Close()

	client := newTestClient(t, server, "notes")
	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	if len(fake.seenAuthUsers) == 0 || fake.seenAuthUsers[0] == "" {
		t.Fatal("expected basic auth header")
	}
}

func TestTestFallsBackToBaseURLWhenDirMissing(t *testing.T) {
	fake := newFakeDAV()
	fake.dirs["/dav"] = true
	server := httptest.NewServer(fake)
	defer server.Close()

	client := newTestClient(t, server, "notes/new")
	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	joined := strings.Join(fake.seenPaths, "\n")
	if !strings.Contains(joined, "PROPFIND /dav/notes/new") {
		t.Fatalf("expected directory probe, got:\n%s", joined)
	}
	if !strings.Contains(joined, "PROPFIND /dav") {
		t.Fatalf("expected base URL probe, got:\n%s", joined)
	}
}

func TestTestAuthFailure(t *testing.T) {
	fake := newFakeDAV()
	fake.user = "alice"
	fake.password = "secret"
	server := httptest.NewServer(fake)
	defer server.Close()

	client := New(Config{URL: server.URL + "/dav", Username: "alice", Password: "wrong"}, i18n.En)
	err := client.Test(context.Background())
	if err == nil || !strings.Contains(err.Error(), "authentication failed") {
		t.Fatalf("err = %v", err)
	}
}

func TestTestFallsBackToOptions(t *testing.T) {
	fake := newFakeDAV()
	fake.propfindCode = http.StatusMethodNotAllowed
	server := httptest.NewServer(fake)
	defer server.Close()

	client := New(Config{URL: server.URL + "/dav"}, i18n.Zh)
	if err := client.Test(context.Background()); err != nil {
		t.Fatalf("Test: %v", err)
	}
	joined := strings.Join(fake.seenPaths, "\n")
	if !strings.Contains(joined, "OPTIONS /dav") {
		t.Fatalf("expected OPTIONS fallback, got:\n%s", joined)
	}
}

func TestGetMissingAndPutCreatesDir(t *testing.T) {
	fake := newFakeDAV()
	server := httptest.NewServer(fake)
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
	if !fake.dirs["/dav/notes"] || !fake.dirs["/dav/notes/2026"] {
		t.Fatalf("dirs = %+v", fake.dirs)
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
