package models

import "time"

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

func (s Status) Valid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusDone:
		return true
	}
	return false
}

type Task struct {
	ID              string       `json:"id"`
	FamilyID        string       `json:"familyId,omitempty"`
	Title           string       `json:"title"`
	Description     string       `json:"description,omitempty"`
	Assignee        string       `json:"assignee"`
	CreatedBy       string       `json:"createdBy"`
	Status          Status       `json:"status"`
	StatusUpdatedBy string       `json:"statusUpdatedBy,omitempty"`
	StatusUpdatedAt *time.Time   `json:"statusUpdatedAt,omitempty"`
	DueAt           *time.Time   `json:"dueAt,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	Attachments     []Attachment `json:"attachments,omitempty"`
}

type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Assignee    string `json:"assignee"`
	DueAt       string `json:"dueAt,omitempty"` // RFC3339 или пусто (бессрочно)
}

type TaskCreate struct {
	FamilyID    string
	Title       string
	Description string
	Assignee    string
	CreatedBy   string
	DueAt       *time.Time
}

type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Assignee    *string `json:"assignee,omitempty"`
	Status      *Status `json:"status,omitempty"`
	// DueAt: nil — не менять, "" — очистить, иначе RFC3339.
	DueAt *string `json:"dueAt,omitempty"`
}

type DueTask struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Assignee  string    `json:"assignee"`
	CreatedBy string    `json:"createdBy"`
	DueAt     time.Time `json:"dueAt"`
}
