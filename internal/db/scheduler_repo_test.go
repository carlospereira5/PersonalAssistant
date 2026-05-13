package db

import (
	"context"
	"testing"

	charm "github.com/charmbracelet/log"
)

func setupSchedulerRepo(t *testing.T) (*SchedulerRepo, func()) {
	t.Helper()

	db, err := NewDB(":memory:", charm.NewWithOptions(nil, charm.Options{
		Level: charm.FatalLevel,
	}))
	if err != nil {
		t.Fatalf("failed to create in-memory DB: %v", err)
	}

	ctx := context.Background()
	if err := db.MigrateContext(ctx); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	repo := NewSchedulerRepo(db)
	return repo, func() { db.Close() }
}

func TestSchedulerRepo_Create(t *testing.T) {
	repo, cleanup := setupSchedulerRepo(t)
	defer cleanup()
	ctx := context.Background()

	tests := []struct {
		name     string
		cronExpr string
		prompt   string
		wantErr  bool
	}{
		{
			name:     "valid routine",
			cronExpr: "0 8 * * *",
			prompt:   "Give me the daily briefing",
			wantErr:  false,
		},
		{
			name:     "different cron",
			cronExpr: "*/15 * * * *",
			prompt:   "Check system status",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := repo.Create(ctx, tt.cronExpr, tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if id <= 0 {
				t.Errorf("Create() got id = %d, want > 0", id)
			}
		})
	}
}

func TestSchedulerRepo_GetAll(t *testing.T) {
	repo, cleanup := setupSchedulerRepo(t)
	defer cleanup()
	ctx := context.Background()

	// Initially empty
	routines, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(routines) != 0 {
		t.Errorf("GetAll() got %d routines, want 0", len(routines))
	}

	// Create two routines
	id1, _ := repo.Create(ctx, "0 8 * * *", "Morning briefing")
	_, _ = repo.Create(ctx, "0 18 * * *", "Evening summary")

	routines, err = repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll() error = %v", err)
	}
	if len(routines) != 2 {
		t.Fatalf("GetAll() got %d routines, want 2", len(routines))
	}

	// Verify first routine
	if routines[0].ID != id1 {
		t.Errorf("routines[0].ID = %d, want %d", routines[0].ID, id1)
	}
	if routines[0].CronExpr != "0 8 * * *" {
		t.Errorf("routines[0].CronExpr = %q, want %q", routines[0].CronExpr, "0 8 * * *")
	}
	if routines[0].Status != "active" {
		t.Errorf("routines[0].Status = %q, want %q", routines[0].Status, "active")
	}
}

func TestSchedulerRepo_GetByID(t *testing.T) {
	repo, cleanup := setupSchedulerRepo(t)
	defer cleanup()
	ctx := context.Background()

	id, _ := repo.Create(ctx, "0 8 * * *", "Test routine")

	tests := []struct {
		name    string
		id      int64
		wantErr bool
	}{
		{
			name:    "existing routine",
			id:      id,
			wantErr: false,
		},
		{
			name:    "nonexistent routine",
			id:      999,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := repo.GetByID(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && r.CronExpr != "0 8 * * *" {
				t.Errorf("GetByID() got cron = %q, want %q", r.CronExpr, "0 8 * * *")
			}
		})
	}
}

func TestSchedulerRepo_Delete(t *testing.T) {
	repo, cleanup := setupSchedulerRepo(t)
	defer cleanup()
	ctx := context.Background()

	id, _ := repo.Create(ctx, "0 8 * * *", "To delete")

	// Delete existing
	if err := repo.Delete(ctx, id); err != nil {
		t.Errorf("Delete() existing error = %v", err)
	}

	// Verify gone
	_, err := repo.GetByID(ctx, id)
	if err == nil {
		t.Error("GetByID() should return error after delete")
	}

	// Delete nonexistent
	if err := repo.Delete(ctx, 999); err == nil {
		t.Error("Delete() nonexistent should return error")
	}
}

func TestSchedulerRepo_UpdateStatus(t *testing.T) {
	repo, cleanup := setupSchedulerRepo(t)
	defer cleanup()
	ctx := context.Background()

	id, _ := repo.Create(ctx, "0 8 * * *", "Test routine")

	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{"pause", "paused", false},
		{"resume", "active", false},
		{"invalid status", "invalid", true}, // CHECK constraint
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateStatus(ctx, id, tt.status)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSchedulerRepo_UpdateLastRun(t *testing.T) {
	repo, cleanup := setupSchedulerRepo(t)
	defer cleanup()
	ctx := context.Background()

	id, _ := repo.Create(ctx, "0 8 * * *", "Test routine")

	// Update with no error
	if err := repo.UpdateLastRun(ctx, id, ""); err != nil {
		t.Fatalf("UpdateLastRun() error = %v", err)
	}

	r, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if r.LastRunAt == nil {
		t.Error("LastRunAt should not be nil after UpdateLastRun")
	}
	if r.LastError != "" {
		t.Errorf("LastError = %q, want empty", r.LastError)
	}

	// Update with error
	if err := repo.UpdateLastRun(ctx, id, "something went wrong"); err != nil {
		t.Fatalf("UpdateLastRun() error = %v", err)
	}

	r, err = repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if r.LastError != "something went wrong" {
		t.Errorf("LastError = %q, want %q", r.LastError, "something went wrong")
	}

	// Update nonexistent
	if err := repo.UpdateLastRun(ctx, 999, ""); err != nil {
		// This is fine — UpdateLastRun doesn't check rows affected
	}
}

func TestSchedulerRepo_UpdateLastRun_NotFound(t *testing.T) {
	repo, cleanup := setupSchedulerRepo(t)
	defer cleanup()
	ctx := context.Background()

	// UpdateLastRun on nonexistent should not error (no rows affected check)
	if err := repo.UpdateLastRun(ctx, 999, ""); err != nil {
		t.Errorf("UpdateLastRun() nonexistent error = %v, want nil", err)
	}
}
