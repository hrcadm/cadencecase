package internal

import "time"

type User struct {
	ID    string `json:"id"`
	Token string `json:"token"`
	Name  string `json:"name"`
}

type SleepLog struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Quality       int       `json:"quality"` // 1–10 scale
	Reason        string    `json:"reason,omitempty"`
	Interruptions []string  `json:"interruptions,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type Goal struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Type      string    `json:"type"` // duration, consistency, quality
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}
