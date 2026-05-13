package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// SchedulerRepo implements domain.SchedulerRepository with SQLite.
type SchedulerRepo struct {
	store *SQLite
}

// NewSchedulerRepo creates a new SchedulerRepo.
func NewSchedulerRepo(store *SQLite) *SchedulerRepo {
	return &SchedulerRepo{store: store}
}

func (r *SchedulerRepo) Create(ctx context.Context, cronExpr, prompt string) (int64, error) {
	result, err := r.store.Db.ExecContext(ctx,
		`INSERT INTO scheduled_routines (cron_expr, prompt) VALUES (?, ?)`,
		cronExpr, prompt,
	)
	if err != nil {
		return 0, fmt.Errorf("db: create scheduled routine: %w", err)
	}
	return result.LastInsertId()
}

func (r *SchedulerRepo) GetAll(ctx context.Context) ([]domain.ScheduledRoutine, error) {
	rows, err := r.store.Db.QueryContext(ctx,
		`SELECT id, cron_expr, prompt, status, last_run_at, last_error, created_at
		 FROM scheduled_routines ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("db: get all scheduled routines: %w", err)
	}
	defer rows.Close()
	return scanScheduledRoutines(rows)
}

func (r *SchedulerRepo) GetByID(ctx context.Context, id int64) (domain.ScheduledRoutine, error) {
	var s domain.ScheduledRoutine
	var lastRunAt *string
	var createdAt string

	err := r.store.Db.QueryRowContext(ctx,
		`SELECT id, cron_expr, prompt, status, last_run_at, last_error, created_at
		 FROM scheduled_routines WHERE id = ?`, id,
	).Scan(&s.ID, &s.CronExpr, &s.Prompt, &s.Status, &lastRunAt, &s.LastError, &createdAt)
	if err == sql.ErrNoRows {
		return s, fmt.Errorf("db: scheduled routine %d not found", id)
	}
	if err != nil {
		return s, fmt.Errorf("db: get scheduled routine %d: %w", id, err)
	}

	s.LastRunAt = parseTimePtr(lastRunAt)
	s.CreatedAt, _ = parseTime(createdAt)
	return s, nil
}

func (r *SchedulerRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.store.Db.ExecContext(ctx,
		`DELETE FROM scheduled_routines WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("db: delete scheduled routine %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: delete scheduled routine %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: scheduled routine %d not found", id)
	}
	return nil
}

func (r *SchedulerRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	result, err := r.store.Db.ExecContext(ctx,
		`UPDATE scheduled_routines SET status = ? WHERE id = ?`, status, id,
	)
	if err != nil {
		return fmt.Errorf("db: update scheduled routine status %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: update scheduled routine status %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: scheduled routine %d not found", id)
	}
	return nil
}

func (r *SchedulerRepo) UpdateLastRun(ctx context.Context, id int64, lastError string) error {
	now := timeNow()
	_, err := r.store.Db.ExecContext(ctx,
		`UPDATE scheduled_routines SET last_run_at = ?, last_error = ? WHERE id = ?`,
		now.Format(timeFormat), lastError, id,
	)
	if err != nil {
		return fmt.Errorf("db: update scheduled routine last_run %d: %w", id, err)
	}
	return nil
}

func scanScheduledRoutines(rows *sql.Rows) ([]domain.ScheduledRoutine, error) {
	var routines []domain.ScheduledRoutine
	for rows.Next() {
		var s domain.ScheduledRoutine
		var lastRunAt *string
		var createdAt string
		if err := rows.Scan(&s.ID, &s.CronExpr, &s.Prompt, &s.Status, &lastRunAt, &s.LastError, &createdAt); err != nil {
			return nil, fmt.Errorf("db: scan scheduled routine: %w", err)
		}
		s.LastRunAt = parseTimePtr(lastRunAt)
		s.CreatedAt, _ = parseTime(createdAt)
		routines = append(routines, s)
	}
	return routines, rows.Err()
}
