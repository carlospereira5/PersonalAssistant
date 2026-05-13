package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// TaskRepo implements domain.TaskRepository with SQLite.
type TaskRepo struct {
	store *SQLite
}

// NewTaskRepo creates a new TaskRepo.
func NewTaskRepo(store *SQLite) *TaskRepo {
	return &TaskRepo{store: store}
}

func (r *TaskRepo) Create(ctx context.Context, name string, deadline string, description string) (int64, error) {
	result, err := r.store.Db.ExecContext(ctx,
		`INSERT INTO tasks (task_name, deadline, status, description) VALUES (?, ?, 'PENDING', ?)`,
		name, deadline, description,
	)
	if err != nil {
		return 0, fmt.Errorf("db: create task: %w", err)
	}
	return result.LastInsertId()
}

func (r *TaskRepo) GetByID(ctx context.Context, id int64) (domain.Task, error) {
	var t domain.Task
	var deadline string
	err := r.store.Db.QueryRowContext(ctx,
		`SELECT id, task_name, deadline, status, description FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &deadline, &t.Status, &t.Description)
	if err == sql.ErrNoRows {
		return t, fmt.Errorf("db: task %d not found", id)
	}
	if err != nil {
		return t, fmt.Errorf("db: get task %d: %w", id, err)
	}
	t.Deadline, _ = parseTime(deadline)
	return t, nil
}

func (r *TaskRepo) GetAll(ctx context.Context) ([]domain.Task, error) {
	rows, err := r.store.Db.QueryContext(ctx,
		`SELECT id, task_name, deadline, status, description FROM tasks ORDER BY deadline ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("db: get all tasks: %w", err)
	}
	defer rows.Close()
	return scanTasks(rows)
}

func (r *TaskRepo) UpdateStatus(ctx context.Context, id int64, status string) error {
	result, err := r.store.Db.ExecContext(ctx,
		`UPDATE tasks SET status = ? WHERE id = ?`, status, id,
	)
	if err != nil {
		return fmt.Errorf("db: update task status %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: update task status %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: task %d not found", id)
	}
	return nil
}

func (r *TaskRepo) Delete(ctx context.Context, id int64) error {
	result, err := r.store.Db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("db: delete task %d: %w", id, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("db: delete task %d rows affected: %w", id, err)
	}
	if n == 0 {
		return fmt.Errorf("db: task %d not found", id)
	}
	return nil
}

func scanTasks(rows *sql.Rows) ([]domain.Task, error) {
	var tasks []domain.Task
	for rows.Next() {
		var t domain.Task
		var deadline string
		if err := rows.Scan(&t.ID, &t.Name, &deadline, &t.Status, &t.Description); err != nil {
			return nil, fmt.Errorf("db: scan task: %w", err)
		}
		t.Deadline, _ = parseTime(deadline)
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}
