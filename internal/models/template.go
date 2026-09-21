package models

import "time"

type TaskTemplate struct {
	ID          string         `json:"id"`
	FamilyID    string         `json:"familyId,omitempty"`
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	Assignee    string         `json:"assignee"`
	CreatedBy   string         `json:"createdBy"`
	Rule        RecurrenceRule `json:"rule"`
	NextRunAt   time.Time      `json:"nextRunAt"`
	Active      bool           `json:"active"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// RecurrenceRule — компактное описание повторения.
// Хранится в БД как JSON-строка.
type RecurrenceRule struct {
	Type       string `json:"type"`                 // daily | weekly | monthly
	Interval   int    `json:"interval,omitempty"`   // для daily, по умолчанию 1
	Weekdays   []int  `json:"weekdays,omitempty"`   // для weekly, 0=вс, 1=пн...
	DayOfMonth int    `json:"dayOfMonth,omitempty"` // для monthly
	Time       string `json:"time"`                 // "HH:MM", локальное время сервера
}

type CreateTemplateRequest struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Assignee    string         `json:"assignee"`
	Rule        RecurrenceRule `json:"rule"`
}

type UpdateTemplateRequest struct {
	Title       *string         `json:"title,omitempty"`
	Description *string         `json:"description,omitempty"`
	Assignee    *string         `json:"assignee,omitempty"`
	Rule        *RecurrenceRule `json:"rule,omitempty"`
	Active      *bool           `json:"active,omitempty"`
}
