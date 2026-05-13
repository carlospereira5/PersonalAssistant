package domain

import "context"

// TaskRepository defines persistence operations for tasks.
type TaskRepository interface {
	Create(ctx context.Context, name string, deadline string, description string) (int64, error)
	GetByID(ctx context.Context, id int64) (Task, error)
	GetAll(ctx context.Context) ([]Task, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	Delete(ctx context.Context, id int64) error
}

// ReminderRepository defines persistence operations for task reminders.
type ReminderRepository interface {
	Create(ctx context.Context, taskID int64, remindAt string) (int64, error)
	GetPending(ctx context.Context) ([]PendingReminder, error)
	MarkSent(ctx context.Context, id int64) error
}

// ConfigRepository defines persistence for key-value config.
type ConfigRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}
