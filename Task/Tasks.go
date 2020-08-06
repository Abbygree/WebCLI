package Task

import "time"

type Task struct {
	TaskID      string    `json:"task_id"`
	GroupID     int       `json:"group_id"`
	Task        string    `json:"task"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at"`
}
