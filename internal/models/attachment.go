package models

import "time"

type Attachment struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	Filename  string    `json:"filename"`
	Mime      string    `json:"mime"`
	Size      int64     `json:"size"`
	CreatedBy string    `json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	URL       string    `json:"url"` // вычисляется на API-слое

	StoredName string `json:"-"` // внутреннее имя файла на диске
}
