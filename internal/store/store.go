package store

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/uiautodev/weektodo/internal/i18n"
	"github.com/uiautodev/weektodo/internal/model"
	"github.com/uiautodev/weektodo/internal/todo"
)

const (
	dataFileName     = "data.json"
	settingsFileName = "settings.json"
	backupDirName    = "backups"
	backupKeep       = 14
)

var (
	// ErrWeekNotFound is returned when a week does not exist.
	ErrWeekNotFound = errors.New("week not found")
	// ErrTaskNotFound is returned when a task does not exist in the source week.
	ErrTaskNotFound = errors.New("task not found")
	// ErrSameWeek is returned when source and target week are identical.
	ErrSameWeek = errors.New("source and target week are the same")
	// ErrCategoryNotFound is returned when a category does not exist.
	ErrCategoryNotFound = errors.New("category not found")
	// ErrCategoryNameTaken is returned when a category name is already used.
	ErrCategoryNameTaken = errors.New("category name already exists")
)

// Store owns the JSON documents and serializes all access to them.
type Store struct {
	mu             sync.RWMutex
	dir            string
	data           model.Data
	settings       model.Settings
	needsMigration bool
}

// DefaultDir returns the platform data directory. WEEKTODO_DATA_DIR overrides it.
func DefaultDir() string {
	if value := strings.TrimSpace(os.Getenv("WEEKTODO_DATA_DIR")); value != "" {
		return value
	}
	return configDir()
}

// configDir returns ~/.config/weektodo on Unix and the per-user application
// data directory on Windows.
func configDir() string {
	if runtime.GOOS == "windows" {
		if base, err := os.UserConfigDir(); err == nil {
			return filepath.Join(base, "weektodo")
		}
		return "weektodo"
	}
	if base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); base != "" {
		return filepath.Join(base, "weektodo")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "weektodo")
	}
	if base, err := os.UserConfigDir(); err == nil {
		return filepath.Join(base, "weektodo")
	}
	return "weektodo"
}

// New opens or creates the store in dir.
func New(dir string) (*Store, error) {
	if strings.TrimSpace(dir) == "" {
		dir = DefaultDir()
		if samePath(dir, configDir()) {
			if err := migrateLegacy(dir); err != nil {
				return nil, err
			}
		}
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
	changed, err := s.ensureDefaults()
	if err != nil {
		return nil, err
	}
	if changed {
		if err := s.saveDataLocked(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *Store) ensureDefaults() (bool, error) {
	changed := s.needsMigration
	s.needsMigration = false
	now := time.Now()
	if s.data.Categories == nil {
		s.data.Categories = []*model.Category{}
		changed = true
	}
	for _, category := range s.data.Categories {
		if err := todo.NormalizeCategory(category, now); err != nil {
			return false, err
		}
	}
	name := "默认"
	if i18n.Normalize(s.settings.Language) == i18n.En {
		name = "Inbox"
	}
	if todo.EnsureDefaultCategory(&s.data, name, now) {
		changed = true
	}
	if s.data.Version != 2 {
		s.data.Version = 2
		changed = true
	}
	todo.SortCategories(s.data.Categories)
	return changed, nil
}

// legacyDirs lists directories used by earlier versions.
func legacyDirs() []string {
	dirs := []string{}
	if base, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, filepath.Join(base, "WeekTodo"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".weektodo-dev"), filepath.Join(home, ".weektodo"))
	}
	return dirs
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	return errA == nil && errB == nil && absA == absB
}

// migrateLegacy copies data from a previous default directory the first time
// the new directory is used. Old files are kept so nothing is lost if the
// migration is interrupted.
func migrateLegacy(target string) error {
	if _, err := os.Stat(filepath.Join(target, dataFileName)); err == nil {
		return nil
	}
	for _, legacy := range legacyDirs() {
		if samePath(legacy, target) {
			continue
		}
		if _, err := os.Stat(filepath.Join(legacy, dataFileName)); err != nil {
			continue
		}
		if err := copyLegacyDir(legacy, target); err != nil {
			return fmt.Errorf("migrate data from %s: %w", legacy, err)
		}
		return nil
	}
	return nil
}

func copyLegacyDir(src, dst string) error {
	for _, name := range []string{dataFileName, settingsFileName} {
		if err := copyFileIfMissing(filepath.Join(src, name), filepath.Join(dst, name)); err != nil {
			return err
		}
	}
	backups := filepath.Join(src, backupDirName)
	return filepath.WalkDir(backups, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		return copyFileIfMissing(path, filepath.Join(dst, rel))
	})
}

func copyFileIfMissing(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return writeAtomic(dst, data)
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
	now := time.Now()
	next := todo.CloneData(data)
	next.Version = 2
	if next.Categories == nil {
		next.Categories = []*model.Category{}
	}
	for _, category := range next.Categories {
		if category == nil {
			continue
		}
		if err := todo.NormalizeCategory(category, now); err != nil {
			return model.Data{}, err
		}
	}
	if todo.EnsureDefaultCategory(&next, "默认", now) {
		// ReplaceData is also used after WebDAV merges, where the default
		// category should already exist. Keep a safe fallback.
		sort.Slice(next.Categories, func(i, j int) bool { return next.Categories[i].Order < next.Categories[j].Order })
	}
	if next.Weeks == nil {
		next.Weeks = []*model.Week{}
	}
	for _, week := range next.Weeks {
		if week == nil {
			continue
		}
		if week.Category == "" {
			week.Category = model.DefaultCategoryID
		}
		if err := todo.NormalizeWeek(week, now); err != nil {
			return model.Data{}, err
		}
	}
	sort.Slice(next.Categories, func(i, j int) bool {
		if next.Categories[i].Order != next.Categories[j].Order {
			return next.Categories[i].Order < next.Categories[j].Order
		}
		return next.Categories[i].Name < next.Categories[j].Name
	})
	sort.Slice(next.Weeks, func(i, j int) bool {
		if next.Weeks[i].Category != next.Weeks[j].Category {
			return next.Weeks[i].Category < next.Weeks[j].Category
		}
		return next.Weeks[i].Start < next.Weeks[j].Start
	})
	s.data = next
	if err := s.saveDataLocked(); err != nil {
		return model.Data{}, err
	}
	return todo.CloneData(next), nil
}

// Categories returns all categories, including archived ones.
func (s *Store) Categories() []model.Category {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Category, 0, len(s.data.Categories))
	for _, category := range s.data.Categories {
		if category == nil {
			continue
		}
		out = append(out, *category)
	}
	return out
}

// Category returns one category by id.
func (s *Store) Category(id string) (model.Category, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	category := s.categoryLocked(id)
	if category == nil {
		return model.Category{}, false
	}
	return *category, true
}

func (s *Store) categoryLocked(id string) *model.Category {
	for _, category := range s.data.Categories {
		if category != nil && category.ID == id {
			return category
		}
	}
	return nil
}

// SaveCategory creates a category when ID is empty, otherwise updates it.
func (s *Store) SaveCategory(next model.Category) (model.Category, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next.ID = strings.TrimSpace(next.ID)
	next.Name = strings.TrimSpace(next.Name)
	if next.Name == "" {
		return model.Category{}, fmt.Errorf("category name cannot be empty")
	}
	now := time.Now()
	index := -1
	for i, category := range s.data.Categories {
		if category != nil && category.ID == next.ID {
			index = i
			break
		}
	}
	if next.ID != "" && index < 0 {
		return model.Category{}, ErrCategoryNotFound
	}
	if index < 0 {
		ids := map[string]bool{}
		for _, category := range s.data.Categories {
			ids[category.ID] = true
		}
		next.ID = newCategoryID(ids)
		if next.Order == 0 {
			next.Order = len(s.data.Categories)
		}
		next.CreatedAt = now.Format(time.RFC3339)
	} else {
		next.CreatedAt = s.data.Categories[index].CreatedAt
	}
	next.UpdatedAt = now.Format(time.RFC3339)
	for i, category := range s.data.Categories {
		if i == index || category == nil {
			continue
		}
		if strings.EqualFold(category.Name, next.Name) {
			return model.Category{}, ErrCategoryNameTaken
		}
	}
	if err := todo.NormalizeCategory(&next, now); err != nil {
		return model.Category{}, err
	}
	if index >= 0 {
		s.data.Categories[index] = &next
	} else {
		s.data.Categories = append(s.data.Categories, &next)
	}
	todo.SortCategories(s.data.Categories)
	if err := s.saveDataLocked(); err != nil {
		return model.Category{}, err
	}
	return next, nil
}

func newCategoryID(existing map[string]bool) string {
	for {
		var buffer [6]byte
		if _, err := rand.Read(buffer[:]); err != nil {
			return fmt.Sprintf("c_%d", time.Now().UnixNano())
		}
		id := "c_" + hex.EncodeToString(buffer[:])
		if !existing[id] {
			return id
		}
	}
}

// Week returns one week by id in the default category.
func (s *Store) Week(id string) (*model.Week, bool) {
	return s.WeekInCategory(model.DefaultCategoryID, id)
}

// WeekInCategory returns one week by category and id.
func (s *Store) WeekInCategory(category, id string) (*model.Week, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.weekLocked(category, id)
}

func (s *Store) weekLocked(category, id string) (*model.Week, bool) {
	for _, week := range s.data.Weeks {
		if week != nil && week.Category == category && week.ID == id {
			return todo.CloneWeek(week), true
		}
	}
	return nil, false
}

// MoveTask moves a task between weeks in the default category.
func (s *Store) MoveTask(fromID, taskID, toID string, now time.Time) (*model.Week, *model.Week, error) {
	return s.MoveTaskInCategory(model.DefaultCategoryID, fromID, taskID, toID, now)
}

// MoveTaskInCategory moves one task subtree to another week in the same
// category and persists both changes in a single write.
func (s *Store) MoveTaskInCategory(category, fromID, taskID, toID string, now time.Time) (*model.Week, *model.Week, error) {
	if fromID == toID {
		return nil, nil, ErrSameWeek
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	category = strings.TrimSpace(category)
	if category == "" {
		category = model.DefaultCategoryID
	}
	if s.categoryLocked(category) == nil {
		return nil, nil, ErrCategoryNotFound
	}
	fromIndex := s.weekIndexLocked(category, fromID)
	if fromIndex < 0 {
		return nil, nil, ErrWeekNotFound
	}
	toIndex := s.weekIndexLocked(category, toID)

	from := todo.CloneWeek(s.data.Weeks[fromIndex])
	var to *model.Week
	if toIndex >= 0 {
		to = todo.CloneWeek(s.data.Weeks[toIndex])
	} else {
		created, err := todo.NewWeek(toID, now)
		if err != nil {
			return nil, nil, err
		}
		to = created
	}
	from.Category = category
	to.Category = category

	remaining, task, ok := todo.TakeTask(from.Tasks, taskID)
	if !ok {
		return nil, nil, ErrTaskNotFound
	}
	from.Tasks = remaining
	todo.PrepareMovedTask(task, now)
	to.Tasks = append(to.Tasks, task)
	if err := todo.NormalizeWeek(from, now); err != nil {
		return nil, nil, err
	}
	if err := todo.NormalizeWeek(to, now); err != nil {
		return nil, nil, err
	}

	s.data.Weeks[fromIndex] = from
	if toIndex >= 0 {
		s.data.Weeks[toIndex] = to
	} else {
		s.data.Weeks = append(s.data.Weeks, to)
	}
	s.sortWeeksLocked()
	if err := s.saveDataLocked(); err != nil {
		return nil, nil, err
	}
	return todo.CloneWeek(from), todo.CloneWeek(to), nil
}

func (s *Store) weekIndexLocked(category, id string) int {
	for index, week := range s.data.Weeks {
		if week != nil && week.Category == category && week.ID == id {
			return index
		}
	}
	return -1
}

func (s *Store) sortWeeksLocked() {
	sort.Slice(s.data.Weeks, func(i, j int) bool {
		if s.data.Weeks[i].Category != s.data.Weeks[j].Category {
			return s.data.Weeks[i].Category < s.data.Weeks[j].Category
		}
		return s.data.Weeks[i].Start < s.data.Weeks[j].Start
	})
}

// ListWeeks returns default-category summaries sorted by start date.
func (s *Store) ListWeeks() []model.WeekSummary {
	return s.ListWeeksInCategory(model.DefaultCategoryID)
}

// ListWeeksInCategory returns summaries for one category.
func (s *Store) ListWeeksInCategory(category string) []model.WeekSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.WeekSummary, 0)
	for _, week := range s.data.Weeks {
		if week == nil || week.Category != category {
			continue
		}
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

// SaveWeek validates and persists a week in the default category.
func (s *Store) SaveWeek(week *model.Week) (*model.Week, error) {
	return s.SaveWeekInCategory(model.DefaultCategoryID, week)
}

// SaveWeekInCategory validates and persists a week inside a category.
func (s *Store) SaveWeekInCategory(category string, week *model.Week) (*model.Week, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	category = strings.TrimSpace(category)
	if category == "" {
		category = model.DefaultCategoryID
	}
	if s.categoryLocked(category) == nil {
		return nil, ErrCategoryNotFound
	}
	clone := todo.CloneWeek(week)
	clone.Category = category
	if err := todo.NormalizeWeek(clone, time.Now()); err != nil {
		return nil, err
	}
	replaced := false
	for i, existing := range s.data.Weeks {
		if existing.Category == category && existing.ID == clone.ID {
			s.data.Weeks[i] = clone
			replaced = true
			break
		}
	}
	if !replaced {
		s.data.Weeks = append(s.data.Weeks, clone)
	}
	s.sortWeeksLocked()
	if err := s.saveDataLocked(); err != nil {
		return nil, err
	}
	return todo.CloneWeek(clone), nil
}

// DeleteWeek removes a default-category week.
func (s *Store) DeleteWeek(id string) error {
	return s.DeleteWeekInCategory(model.DefaultCategoryID, id)
}

// DeleteWeekInCategory removes one week from a category.
func (s *Store) DeleteWeekInCategory(category, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := make([]*model.Week, 0, len(s.data.Weeks))
	found := false
	for _, week := range s.data.Weeks {
		if week.Category == category && week.ID == id {
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

// TodayCount returns unfinished tasks for today across all categories.
func (s *Store) TodayCount(now time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	for _, category := range s.data.Categories {
		week, ok := s.weekLocked(category.ID, todo.WeekIDFor(now))
		if ok {
			total += todo.TodayCount(week, todo.DateString(now))
		}
	}
	return total
}

// TodayCountInCategory returns unfinished tasks for today in one category.
func (s *Store) TodayCountInCategory(category string, now time.Time) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	week, ok := s.weekLocked(category, todo.WeekIDFor(now))
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
	next.WebDAV.RemoteDir = normalizeRemoteDir(next.WebDAV.RemoteDir)
	next.WeekSpan = normalizeWeekSpan(next.WeekSpan)
	next.ActiveCategory = strings.TrimSpace(next.ActiveCategory)
	if next.CategoryWeeks != nil {
		weeks := map[string]string{}
		for id, weekID := range next.CategoryWeeks {
			id = strings.TrimSpace(id)
			weekID = strings.TrimSpace(weekID)
			if id != "" && weekID != "" {
				weeks[id] = weekID
			}
		}
		next.CategoryWeeks = weeks
	}
	s.settings = next
	return s.saveSettingsLocked()
}

func (s *Store) loadData() error {
	path := filepath.Join(s.dir, dataFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.data = model.Data{Version: 2, Categories: []*model.Category{}, Weeks: []*model.Week{}}
		s.needsMigration = true
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
	migrated := parsed.Version < 2
	if parsed.Version == 0 {
		parsed.Version = 1
	}
	if parsed.Categories == nil {
		parsed.Categories = []*model.Category{}
		migrated = true
	}
	seenCategories := map[string]bool{}
	for _, category := range parsed.Categories {
		if category == nil {
			return fmt.Errorf("parse %s: null category", path)
		}
		if err := todo.NormalizeCategory(category, time.Now()); err != nil {
			return fmt.Errorf("normalize category %s: %w", category.ID, err)
		}
		if seenCategories[category.ID] {
			return fmt.Errorf("parse %s: duplicate category %s", path, category.ID)
		}
		seenCategories[category.ID] = true
	}
	if parsed.Weeks == nil {
		parsed.Weeks = []*model.Week{}
	}
	seen := map[string]bool{}
	for _, week := range parsed.Weeks {
		if week == nil {
			return fmt.Errorf("parse %s: null week", path)
		}
		week.Category = strings.TrimSpace(week.Category)
		if week.Category == "" {
			week.Category = model.DefaultCategoryID
			migrated = true
		}
		key := week.Category + "\x00" + week.ID
		if seen[key] {
			return fmt.Errorf("parse %s: duplicate week %s in category %s", path, week.ID, week.Category)
		}
		seen[key] = true
		if err := todo.NormalizeWeek(week, time.Now()); err != nil {
			return fmt.Errorf("normalize week %s: %w", week.ID, err)
		}
	}
	parsed.Version = 2
	sort.Slice(parsed.Weeks, func(i, j int) bool {
		if parsed.Weeks[i].Category != parsed.Weeks[j].Category {
			return parsed.Weeks[i].Category < parsed.Weeks[j].Category
		}
		return parsed.Weeks[i].Start < parsed.Weeks[j].Start
	})
	s.data = parsed
	s.needsMigration = migrated
	return nil
}

func (s *Store) loadSettings() error {
	path := filepath.Join(s.dir, settingsFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		s.settings = model.Settings{WeekSpan: 1}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var parsed model.Settings
	if err := json.Unmarshal(data, &parsed); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	parsed.WeekSpan = normalizeWeekSpan(parsed.WeekSpan)
	parsed.WebDAV.RemoteDir = normalizeRemoteDir(parsed.WebDAV.RemoteDir)
	s.settings = parsed
	return nil
}

func normalizeWeekSpan(value int) int {
	if value == 2 || value == 3 {
		return value
	}
	return 1
}

// normalizeRemoteDir keeps one leading slash for display while the WebDAV
// client still treats the path as relative to the configured URL.
func normalizeRemoteDir(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "/")
	if value == "" {
		return ""
	}
	return "/" + value
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
