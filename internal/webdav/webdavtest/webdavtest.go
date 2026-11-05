// Package webdavtest provides an in-memory WebDAV server for tests.
package webdavtest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// Server is a small in-memory WebDAV endpoint. It supports the subset used by
// WeekTodo: PROPFIND, OPTIONS, MKCOL, GET and PUT.
type Server struct {
	mu             sync.Mutex
	files          map[string][]byte
	dirs           map[string]bool
	requests       []string
	user           string
	pass           string
	propfindStatus int
	optionsStatus  int
	http           *httptest.Server
}

// New starts an in-memory WebDAV server.
func New() *Server {
	s := &Server{
		files: map[string][]byte{},
		dirs:  map[string]bool{"/": true},
	}
	s.http = httptest.NewServer(http.HandlerFunc(s.serve))
	return s
}

// URL returns the server base URL.
func (s *Server) URL() string {
	return s.http.URL
}

// Close stops the server.
func (s *Server) Close() {
	s.http.Close()
}

// Auth makes the server require Basic Auth.
func (s *Server) Auth(user, pass string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.user = user
	s.pass = pass
}

// PropfindStatus forces PROPFIND responses to use the given status.
func (s *Server) PropfindStatus(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.propfindStatus = status
}

// OptionsStatus forces OPTIONS responses to use the given status.
func (s *Server) OptionsStatus(status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.optionsStatus = status
}

// Mkdir creates a directory and its parents.
func (s *Server) Mkdir(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mkdirLocked(normalize(path))
}

// SetFile stores a remote file, creating its parent directories.
func (s *Server) SetFile(path string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path = normalize(path)
	s.mkdirLocked(parent(path))
	s.files[path] = append([]byte(nil), data...)
}

// File returns a stored file.
func (s *Server) File(path string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.files[normalize(path)]
	return append([]byte(nil), data...), ok
}

// HasDir reports whether a directory exists.
func (s *Server) HasDir(path string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dirs[normalize(path)]
}

// Requests returns the recorded "METHOD /path" requests.
func (s *Server) Requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func normalize(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func parent(path string) string {
	index := strings.LastIndex(strings.TrimRight(path, "/"), "/")
	if index <= 0 {
		return "/"
	}
	return path[:index]
}

func (s *Server) mkdirLocked(path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	current := "/"
	s.dirs[current] = true
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = strings.TrimRight(current, "/") + "/" + part
		s.dirs[current] = true
	}
}

func (s *Server) serve(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, r.Method+" "+r.URL.Path)
	user, pass := s.user, s.pass
	propfindStatus, optionsStatus := s.propfindStatus, s.optionsStatus
	s.mu.Unlock()

	if user != "" {
		gotUser, gotPass, ok := r.BasicAuth()
		if !ok || gotUser != user || gotPass != pass {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	path := normalize(r.URL.Path)
	switch r.Method {
	case "PROPFIND":
		s.drain(r)
		if propfindStatus != 0 {
			w.WriteHeader(propfindStatus)
			return
		}
		s.mu.Lock()
		exists := s.dirs[path]
		s.mu.Unlock()
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusMultiStatus)
	case http.MethodOptions:
		s.drain(r)
		if optionsStatus != 0 {
			w.WriteHeader(optionsStatus)
			return
		}
		w.Header().Set("DAV", "1,2")
		w.WriteHeader(http.StatusOK)
	case "MKCOL":
		s.drain(r)
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.dirs[path] {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !s.dirs[parent(path)] {
			w.WriteHeader(http.StatusConflict)
			return
		}
		s.dirs[path] = true
		w.WriteHeader(http.StatusCreated)
	case http.MethodPut:
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.dirs[parent(path)] {
			w.WriteHeader(http.StatusConflict)
			return
		}
		s.files[path] = body
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet:
		s.mu.Lock()
		data, ok := s.files[path]
		s.mu.Unlock()
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) drain(r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
}
