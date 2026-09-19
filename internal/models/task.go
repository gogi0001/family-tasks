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
	ID              string    `json:"id"`
	FamilyID        string    `json:"familyId,omitempty"`
	Title           string    `json:"title"`
	Description     string    `json:"description,omitempty"`
	Assignee        string    `json:"assignee"`
	CreatedBy       string    `json:"createdBy"`
	Status          Status    `json:"status"`
	StatusUpdatedBy string    `json:"statusUpdatedBy,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type TaskCreate struct {
	FamilyID    string
	Title       string
	Description string
	Assignee    string
	CreatedBy   string
}

// CreateTaskRequest — то, что приходит от клиента по HTTP.
// CreatedBy отсутствует намеренно: сервер берёт его из сессии.
type CreateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Assignee    string `json:"assignee"`
}

type UpdateTaskRequest struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	Assignee    *string `json:"assignee,omitempty"`
	Status      *Status `json:"status,omitempty"`
}
