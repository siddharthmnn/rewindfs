package models

import "time"

type Snapshot struct {
	ID        int       `json:"id"`
	File      string    `json:"file"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
