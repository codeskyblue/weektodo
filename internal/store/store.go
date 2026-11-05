package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/todo"
)

const (
	dataFileName     = "data.json"
	settingsFileName = "settings.json"
	backupDirName    = "backups"
	backupKeep       = 14
)

// Store owns the JSON documents and serializes all access to them.
type Store struct {
	mu       sync.RWMutex
	dir      string
	data     model.Data
	settings model.Settings
}

// DefaultDir returns the platform data directory.
func DefaultDir() string {
	if value := strings.TrimSpace(os.Getenv("WEEKTODO_DATA_DIR")); value != "" {
		return value
	}
	base, err := os.UserConfigDir()
	if err != nil {
		base = "."
	}
	return filepath.Join(base, "WeekTodo")
}

// New opens or creates the store in dir.
func New(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		dir = DefaultDir()
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	s := &Store{dir: dir}
	if err := s.loadData(); err != nil {
		return nil, err
	}
	if err := s.loadSettings(); err != nil {
		return nil, err
	}
	return s, nil
}

// Dir returns the store directory.
func (s *Store) Dir() string {
	return s.dir
}

// Snapshot returns a deep copy of all weeks.
func (s *Store) Snapshot() model.Data {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return todo.CloneData(s.data)
}

// ReplaceData atomically replaces all weeks, normalizes them and persists once.
func (s *Store) ReplaceData(data model.Data) (model.Data, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := todo.CloneData(data)
	if next.Version == 0 {
		next.Version = 1
	}
	if next.Weeks == nil {
		next.Weeks = []*model.Week{}
	}
	for _, week := range next.Weeks {
		if week == nil {
			continue
		}
		if err := todo.NormalizeWeek(week, time.Now()); err != nil {
			return model.Data{}, err
		}
	}
	sort.Slice(next.Weeks, func(i, j int) bool { return next.Weeks[i].Start < next.Weeks[j].Start })
	s.data = next
	if err := s.saveDataLocked(); err != nil {
		return model.Data{}, err
	}
	return todo.CloneData(next), nil
}

// Week returns one week by id.
func (s *Store) Week(id string) (*model.Week, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.weekLocked(id)
}

func (s *Store) weekLocked(id string) (*model.Week, bool) {
	for _, week := range s.data.Weeks {
		if week.ID == id {
			return todo.CloneWeek(week), true
		}
	}
	return nil, false
}

// ListWeeks returns summaries sorted by start date.
func (s *Store) ListWeeks() []model.WeekSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.WeekSummary, 0, len(s.data.Weeks))
	for _, week := range s.data.Weeks {
		stats := todo.WeekStats(week)
		out = append(out, model.WeekSummary{
			ID:        week.ID,
			Start:     week.Start,
			End:       week.End,
			Total:     stats.Total,
			Done:      stats.Done,
			Dropped:   stats.Dropped,
			Unplanned: stats.Unplanned,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start < out[j].Start })
	return out
}

// SaveWeek validates and persists a week.
func (s *Store) SaveWeek(week *model.Week) (*model.Week, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	clone := todo.CloneWeek(week)
	if err := todo.NormalizeWeek(clone, time.Now()); err != nil {
		return nil, err
	}
	replaced := false
	for i, existing := range s.data.Weeks {
		if existing.ID == clone.ID {
			s.data.Weeks[i] = clone
			replaced = true
			break
		}
	}
	if !replaced {
		s.data.Weeks = append(s.data.Weeks, clone)
	}
	sort.Slice(s.data.Weeks, func(i, j int) bool { return s.data.Weeks[i].Start < s.data.Weeks[j].Start })
	if err := s.saveDataLocked(); err != nil {
		return nil, err
	}
	return todo.CloneWeek(clone), nil
}

// DeleteWeek removes a week.
func (s *Store) DeleteWeek(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data.Weeks[:0]
	found := false
	for _, week := range s.data.Weeks {
		if week.ID == id {
			found = true
			continue
		}
		next = append(next, week)
	}
	if !found {
		return nil
	}
	s.data.Weeks = next
	return s.saveDataLocked()
}

// TodayCount returns the number of unfinished tasks scheduled for now's date.
func (s *Store) TodayCount(now time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	week, ok := s.weekLocked(todo.WeekIDFor(now))
	if !ok {
		return 0
	}
	return todo.TodayCount(week, todo.DateString(now))
}

// Settings returns the persisted settings, including the API key.
func (s *Store) Settings() model.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// UpdateSettings persists normalized settings.
func (s *Store) UpdateSettings(next model.Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next.AI.BaseURL = strings.TrimRight(strings.TrimSpace(next.AI.BaseURL), "/")
	next.AI.Model = strings.TrimSpace(next.AI.Model)
	next.AI.APIKey = strings.TrimSpace(next.AI.APIKey)
	next.WebDAV.URL = strings.TrimRight(strings.TrimSpace(next.WebDAV.URL), "/")
	next.WebDAV.Username = strings.TrimSpace(next.WebDAV.Username)
	next.WebDAV.Password = strings.TrimSpace(next.WebDAV.Password)
	next.WebDAV.RemoteDir = strings.Trim(strings.TrimSpace(next.WebDAV.RemoteDir), "/")
	s.settings = next
	return s.saveSettingsLocked()
}

func (s *Store) loadData() error {
	path := filepath.Join(s.dir, dataFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.data = model.Data{Version: 1, Weeks: []*model.Week{}}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var parsed model.Data
	if err := json.Unmarshal(data, &parsed); err != nil {
		if recovered, ok := s.loadLatestBackup(); ok {
			_ = os.Rename(path, path+".corrupt-"+time.Now().Format("20060102-150405"))
			s.data = recovered
			_ = s.saveDataLocked()
			return nil
		}
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if parsed.Version == 0 {
		parsed.Version = 1
	}
	if parsed.Weeks == nil {
		parsed.Weeks = []*model.Week{}
	}
	seen := map[string]bool{}
	for _, week := range parsed.Weeks {
		if week == nil {
			return fmt.Errorf("parse %s: null week", path)
		}
		if seen[week.ID] {
			return fmt.Errorf("parse %s: duplicate week %s", path, week.ID)
		}
		seen[week.ID] = true
		if err := todo.NormalizeWeek(week, time.Now()); err != nil {
			return fmt.Errorf("normalize week %s: %w", week.ID, err)
		}
	}
	sort.Slice(parsed.Weeks, func(i, j int) bool { return parsed.Weeks[i].Start < parsed.Weeks[j].Start })
	s.data = parsed
	return nil
}

func (s *Store) loadSettings() error {
	path := filepath.Join(s.dir, settingsFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.settings = model.Settings{}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var parsed model.Settings
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	s.settings = parsed
	return nil
}

func (s *Store) loadLatestBackup() (model.Data, bool) {
	entries, err := os.ReadDir(filepath.Join(s.dir, backupDirName))
	if err != nil {
		return model.Data{}, false
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "data-") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for _, name := range names {
		raw, err := os.ReadFile(filepath.Join(s.dir, backupDirName, name))
		if err != nil {
			continue
		}
		var parsed model.Data
		if err := json.Unmarshal(raw, &parsed); err != nil {
			continue
		}
		if parsed.Weeks == nil {
			parsed.Weeks = []*model.Week{}
		}
		if parsed.Version == 0 {
			parsed.Version = 1
		}
		for _, week := range parsed.Weeks {
			if week == nil {
				continue
			}
			if err := todo.NormalizeWeek(week, time.Now()); err != nil {
				return model.Data{}, false
			}
		}
		return parsed, true
	}
	return model.Data{}, false
}

func (s *Store) saveDataLocked() error {
	if err := s.backupLocked(time.Now()); err != nil {
		// A failed backup must not stop the user's save.
		_ = err
	}
	data, err := json.MarshalIndent(s.data, "", "\t")
	if err != nil {
		return fmt.Errorf("encode data: %w", err)
	}
	return writeAtomic(filepath.Join(s.dir, dataFileName), data)
}

func (s *Store) saveSettingsLocked() error {
	data, err := json.MarshalIndent(s.settings, "", "\t")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	return writeAtomic(filepath.Join(s.dir, settingsFileName), data)
}

func (s *Store) backupLocked(now time.Time) error {
	source := filepath.Join(s.dir, dataFileName)
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Size() == 0 {
		return nil
	}
	dir := filepath.Join(s.dir, backupDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	target := filepath.Join(dir, "data-"+now.Format("2006-01-02")+".json")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	pruneBackups(dir)
	return nil
}

func pruneBackups(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "data-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		names = append(names, name)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	for i, name := range names {
		if i < backupKeep {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".weektodo-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	_ = os.Chmod(path, 0o600)
	if dirHandle, err := os.Open(dir); err == nil {
		_ = dirHandle.Sync()
		_ = dirHandle.Close()
	}
	return nil
}
