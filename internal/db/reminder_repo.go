package db

import (
	"context"
	"fmt"

	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// ReminderRepo implements domain.ReminderRepository with SQLite.
type ReminderRepo struct {
	store *SQLite
}

// NewReminderRepo creates a new ReminderRepo.
func NewReminderRepo(store *SQLite) *ReminderRepo {
	return &ReminderRepo{store: store}
}

func (r *ReminderRepo) Create(ctx context.Context, taskID int64, remindAt string) (int64, error) {
	result, err := r.store.Db.ExecContext(ctx,
		`INSERT INTO task_reminders (task_id, remind_at) VALUES (?, ?)`,
		taskID, remindAt,
	)
	if err != nil {
		return 0, fmt.Errorf("db: create reminder: %w", err)
	}
	return result.LastInsertId()
}

// GetPending retorna los recordatorios no enviados cuya hora ya llegó.
func (r *ReminderRepo) GetPending(ctx context.Context) ([]domain.PendingReminder, error) {
	rows, err := r.store.Db.QueryContext(ctx, `
		SELECT tr.id, tr.task_id, t.task_name
		FROM task_reminders tr
		JOIN tasks t ON tr.task_id = t.id
		WHERE tr.sent = 0
		  AND tr.remind_at <= datetime('now')
	`)
	if err != nil {
		return nil, fmt.Errorf("db: get pending reminders: %w", err)
	}
	defer rows.Close()

	var out []domain.PendingReminder
	for rows.Next() {
		var pr domain.PendingReminder
		if err := rows.Scan(&pr.ID, &pr.TaskID, &pr.TaskName); err != nil {
			return nil, fmt.Errorf("db: scan pending reminder: %w", err)
		}
		out = append(out, pr)
	}
	return out, rows.Err()
}

func (r *ReminderRepo) MarkSent(ctx context.Context, id int64) error {
	_, err := r.store.Db.ExecContext(ctx,
		`UPDATE task_reminders SET sent = 1 WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("db: mark reminder sent %d: %w", id, err)
	}
	return nil
}
