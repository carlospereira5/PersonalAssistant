package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/carlospereira5/PersonalAssistant/internal/domain"
	"github.com/shopspring/decimal"
)

// DebtPaymentRepo implements domain.DebtPaymentRepository with SQLite.
type DebtPaymentRepo struct {
	store *SQLite
}

// NewDebtPaymentRepo creates a new DebtPaymentRepo.
func NewDebtPaymentRepo(store *SQLite) *DebtPaymentRepo {
	return &DebtPaymentRepo{store: store}
}

func (r *DebtPaymentRepo) Create(ctx context.Context, debtID int64, amount int64, notes string, paidAt string) (int64, error) {
	result, err := r.store.Db.ExecContext(ctx,
		`INSERT INTO debts_payments (debt_id, amount_int, notes, paid_at, created_at) VALUES (?, ?, ?, ?, datetime('now'))`,
		debtID, amount, notes, paidAt,
	)
	if err != nil {
		return 0, fmt.Errorf("db: create debt payment: %w", err)
	}
	return result.LastInsertId()
}

func (r *DebtPaymentRepo) GetByID(ctx context.Context, id int64) (domain.DebtPayment, error) {
	var p domain.DebtPayment
	var amountInt int64
	var paidAt, createdAt string
	err := r.store.Db.QueryRowContext(ctx,
		`SELECT id, debt_id, amount_int, notes, paid_at, created_at FROM debts_payments WHERE id = ?`, id,
	).Scan(&p.ID, &p.DebtID, &amountInt, &p.Notes, &paidAt, &createdAt)
	if err == sql.ErrNoRows {
		return p, fmt.Errorf("db: payment %d not found", id)
	}
	if err != nil {
		return p, fmt.Errorf("db: get payment %d: %w", id, err)
	}
	p.Amount = domain.FromCents(amountInt)
	p.PaidAt, _ = parseTime(paidAt)
	p.CreatedAt, _ = parseTime(createdAt)
	return p, nil
}

func (r *DebtPaymentRepo) GetByDebt(ctx context.Context, debtID int64) ([]domain.DebtPayment, error) {
	rows, err := r.store.Db.QueryContext(ctx,
		`SELECT id, debt_id, amount_int, notes, paid_at, created_at FROM debts_payments WHERE debt_id = ? ORDER BY paid_at DESC`,
		debtID,
	)
	if err != nil {
		return nil, fmt.Errorf("db: get payments for debt %d: %w", debtID, err)
	}
	defer rows.Close()

	var payments []domain.DebtPayment
	for rows.Next() {
		var p domain.DebtPayment
		var amountInt int64
		var paidAt, createdAt string
		if err := rows.Scan(&p.ID, &p.DebtID, &amountInt, &p.Notes, &paidAt, &createdAt); err != nil {
			return nil, fmt.Errorf("db: scan debt payment: %w", err)
		}
		p.Amount = domain.FromCents(amountInt)
		p.PaidAt, _ = parseTime(paidAt)
		p.CreatedAt, _ = parseTime(createdAt)
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

func (r *DebtPaymentRepo) GetTotalPaid(ctx context.Context, debtID int64) (decimal.Decimal, error) {
	var total int64
	err := r.store.Db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount_int), 0) FROM debts_payments WHERE debt_id = ?`, debtID,
	).Scan(&total)
	if err != nil {
		return decimal.Zero, fmt.Errorf("db: get total paid for debt %d: %w", debtID, err)
	}
	return domain.FromCents(total), nil
}

func (r *DebtPaymentRepo) Update(ctx context.Context, id int64, amount int64, notes string, paidAt string) error {
	result, err := r.store.Db.ExecContext(ctx,
		`UPDATE debts_payments SET amount_int = ?, notes = ?, paid_at = ? WHERE id = ?`,
		amount, notes, paidAt, id,
	)
	if err != nil {
		return fmt.Errorf("db: update payment %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: update payment %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: payment %d not found", id)
	}
	return nil
}

func (r *DebtPaymentRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.store.Db.ExecContext(ctx, `DELETE FROM debts_payments WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete payment %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: delete payment %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: payment %d not found", id)
	}
	return nil
}
