package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// DebtRepo implements domain.DebtRepository with SQLite.
// Single-user: no UserID field — debtor is identified by Name.
type DebtRepo struct {
	store *SQLite
}

// NewDebtRepo creates a new DebtRepo.
func NewDebtRepo(store *SQLite) *DebtRepo {
	return &DebtRepo{store: store}
}

func (r *DebtRepo) Create(ctx context.Context, name string, totalAmount int64, description string) (int64, error) {
	result, err := r.store.Db.ExecContext(ctx,
		`INSERT INTO debts (name, total_amount, state, description) VALUES (?, ?, 'PENDING', ?)`,
		name, totalAmount, description,
	)
	if err != nil {
		return 0, fmt.Errorf("db: create debt: %w", err)
	}
	return result.LastInsertId()
}

func (r *DebtRepo) GetByID(ctx context.Context, id int64) (domain.Debt, error) {
	var d domain.Debt
	var totalAmountInt int64
	err := r.store.Db.QueryRowContext(ctx,
		`SELECT id, name, total_amount, state, description FROM debts WHERE id = ?`, id,
	).Scan(&d.ID, &d.Name, &totalAmountInt, &d.State, &d.Description)
	if err == sql.ErrNoRows {
		return d, fmt.Errorf("db: debt %d not found", id)
	}
	if err != nil {
		return d, fmt.Errorf("db: get debt %d: %w", id, err)
	}
	d.TotalAmount = domain.FromCents(totalAmountInt)
	return d, nil
}

func (r *DebtRepo) GetAll(ctx context.Context) ([]domain.Debt, error) {
	rows, err := r.store.Db.QueryContext(ctx,
		`SELECT id, name, total_amount, state, description FROM debts ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("db: get all debts: %w", err)
	}
	defer rows.Close()
	return scanDebts(rows)
}

func (r *DebtRepo) Update(ctx context.Context, id int64, name string, totalAmount int64, description string) error {
	result, err := r.store.Db.ExecContext(ctx,
		`UPDATE debts SET name = ?, total_amount = ?, description = ? WHERE id = ?`,
		name, totalAmount, description, id,
	)
	if err != nil {
		return fmt.Errorf("db: update debt %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: update debt %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: debt %d not found", id)
	}
	return nil
}

func (r *DebtRepo) UpdateState(ctx context.Context, id int64, state string) error {
	result, err := r.store.Db.ExecContext(ctx,
		`UPDATE debts SET state = ? WHERE id = ?`, state, id,
	)
	if err != nil {
		return fmt.Errorf("db: update debt state %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: update debt state %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: debt %d not found", id)
	}
	return nil
}

func (r *DebtRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.store.Db.ExecContext(ctx, `DELETE FROM debts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete debt %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: delete debt %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: debt %d not found", id)
	}
	return nil
}

func scanDebts(rows *sql.Rows) ([]domain.Debt, error) {
	var debts []domain.Debt
	for rows.Next() {
		var d domain.Debt
		var totalAmountInt int64
		if err := rows.Scan(&d.ID, &d.Name, &totalAmountInt, &d.State, &d.Description); err != nil {
			return nil, fmt.Errorf("db: scan debt: %w", err)
		}
		d.TotalAmount = domain.FromCents(totalAmountInt)
		debts = append(debts, d)
	}
	return debts, rows.Err()
}
