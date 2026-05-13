// Package domain defines the core business entities for the personal assistant.
package domain

import "time"

// Task represents a unit of work.
type Task struct {
	ID          int64
	Name        string
	Deadline    time.Time
	Status      string // PENDING, IN_PROGRESS, DONE, CANCELLED
	Description string
}

// PendingReminder groups the data needed to fire a reminder notification.
type PendingReminder struct {
	ID       int64
	TaskID   int64
	TaskName string
}
