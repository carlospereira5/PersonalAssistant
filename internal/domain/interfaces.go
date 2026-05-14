package domain

import (
	"context"

	"github.com/shopspring/decimal"
)

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

// DebtRepository defines persistence operations for debts.
// Single-user: no GetByUser, Create uses name string (debtor name).
type DebtRepository interface {
	Create(ctx context.Context, name string, totalAmount int64, description string) (int64, error)
	GetByID(ctx context.Context, id int64) (Debt, error)
	GetAll(ctx context.Context) ([]Debt, error)
	Update(ctx context.Context, id int64, name string, totalAmount int64, description string) error
	UpdateState(ctx context.Context, id int64, state string) error
	Delete(ctx context.Context, id int64) error
}

// DebtPaymentRepository defines persistence operations for debt payments.
type DebtPaymentRepository interface {
	Create(ctx context.Context, debtID int64, amount int64, notes string, paidAt string) (int64, error)
	GetByID(ctx context.Context, id int64) (DebtPayment, error)
	GetByDebt(ctx context.Context, debtID int64) ([]DebtPayment, error)
	GetTotalPaid(ctx context.Context, debtID int64) (decimal.Decimal, error)
	Update(ctx context.Context, id int64, amount int64, notes string, paidAt string) error
	Delete(ctx context.Context, id int64) error
}

// SchedulerRepository defines persistence operations for scheduled routines.
type SchedulerRepository interface {
	Create(ctx context.Context, cronExpr, prompt string) (int64, error)
	GetAll(ctx context.Context) ([]ScheduledRoutine, error)
	GetByID(ctx context.Context, id int64) (ScheduledRoutine, error)
	Delete(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateLastRun(ctx context.Context, id int64, lastError string) error
}
