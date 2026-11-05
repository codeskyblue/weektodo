package model

// Status is the lifecycle state of a task.
type Status string

const (
	StatusTodo    Status = "todo"
	StatusDone    Status = "done"
	StatusDropped Status = "dropped"
)

// CarriedFrom records the week and task a task was copied from.
type CarriedFrom struct {
	WeekID string `json:"weekId"`
	TaskID string `json:"taskId"`
}

// Task is one node in a week's task tree.
type Task struct {
	ID          string       `json:"id"`
	Text        string       `json:"text"`
	Status      Status       `json:"status"`
	Day         string       `json:"day,omitempty"` // YYYY-MM-DD inside the week, empty means unplanned
	CreatedAt   string       `json:"createdAt,omitempty"`
	UpdatedAt   string       `json:"updatedAt,omitempty"`
	CompletedAt string       `json:"completedAt,omitempty"`
	DroppedAt   string       `json:"droppedAt,omitempty"`
	CarriedFrom *CarriedFrom `json:"carriedFrom,omitempty"`
	Children    []*Task      `json:"children"`
}

// DefaultCategoryID is the category used by data created before categories.
const DefaultCategoryID = "default"

// Category is one independent weekly workspace.
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Color     string `json:"color,omitempty"`
	Order     int    `json:"order,omitempty"`
	Archived  bool   `json:"archived"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// Week is one ISO week of tasks inside a category.
type Week struct {
	ID        string  `json:"id"`
	Category  string  `json:"category,omitempty"`
	Start     string  `json:"start"` // YYYY-MM-DD, Monday
	End       string  `json:"end"`   // YYYY-MM-DD, Sunday
	CreatedAt string  `json:"createdAt,omitempty"`
	Tasks     []*Task `json:"tasks"`
}

// Data is the complete persisted document.
type Data struct {
	Version    int         `json:"version"`
	Categories []*Category `json:"categories,omitempty"`
	Weeks      []*Week     `json:"weeks"`
}

// AIConfig is the persisted OpenAI-compatible endpoint configuration.
type AIConfig struct {
	BaseURL string `json:"baseUrl"`
	APIKey  string `json:"apiKey"`
	Model   string `json:"model"`
}

// WebDAVConfig is the persisted WebDAV endpoint configuration.
type WebDAVConfig struct {
	URL       string `json:"url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	RemoteDir string `json:"remoteDir"`
	AutoSync  bool   `json:"autoSync,omitempty"`
}

// Settings is stored separately from task data.
type Settings struct {
	AI             AIConfig          `json:"ai"`
	WebDAV         WebDAVConfig      `json:"webdav"`
	WeekSpan       int               `json:"weekSpan,omitempty"`
	Language       string            `json:"language,omitempty"`
	ActiveCategory string            `json:"activeCategory,omitempty"`
	CategoryWeeks  map[string]string `json:"categoryWeeks,omitempty"`
}

// WeekSummary is a lightweight week list entry.
type WeekSummary struct {
	ID        string `json:"id"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Total     int    `json:"total"`
	Done      int    `json:"done"`
	Dropped   int    `json:"dropped"`
	Unplanned int    `json:"unplanned"`
}

// Stats counts leaf tasks in a week.
type Stats struct {
	Total     int `json:"total"`
	Done      int `json:"done"`
	Dropped   int `json:"dropped"`
	Unplanned int `json:"unplanned"`
}
