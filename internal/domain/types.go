// Package domain defines the core business entities for the personal assistant.
package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// ToCents convierte decimal.Decimal a enteros (cents × 100) para almacenar en la DB.
func ToCents(d decimal.Decimal) int64 {
	return d.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
}

// FromCents convierte enteros (cents × 100) leídos de la DB a decimal.Decimal.
func FromCents(i int64) decimal.Decimal {
	return decimal.NewFromInt(i).Shift(-2)
}

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

// Debt represents money owed by/to someone. Single-user: Name identifies the debtor.
type Debt struct {
	ID          int64
	Name        string          // Who owes or is owed
	TotalAmount decimal.Decimal // Total amount of the debt
	State       string          // PENDING | PARTIAL | PAID
	Description string
}

// DebtPayment represents a partial payment toward a debt.
type DebtPayment struct {
	ID        int64
	DebtID    int64
	Amount    decimal.Decimal
	Notes     string
	PaidAt    time.Time
	CreatedAt time.Time
}

// ScheduledRoutine represents a recurring automated routine.
type ScheduledRoutine struct {
	ID        int64
	CronExpr  string
	Prompt    string
	Status    string // "active" or "paused"
	LastRunAt *time.Time
	LastError string
	CreatedAt time.Time
}
