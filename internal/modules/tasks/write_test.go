package tasks

import (
	"context"
	"io"
	"testing"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockTaskRepo struct {
	createID  int64
	createErr error
}

func (m *mockTaskRepo) Create(_ context.Context, _, _, _ string) (int64, error) {
	return m.createID, m.createErr
}
func (m *mockTaskRepo) GetByID(_ context.Context, _ int64) (domain.Task, error) {
	return domain.Task{}, nil
}
func (m *mockTaskRepo) GetAll(_ context.Context) ([]domain.Task, error)         { return nil, nil }
func (m *mockTaskRepo) UpdateStatus(_ context.Context, _ int64, _ string) error { return nil }
func (m *mockTaskRepo) Delete(_ context.Context, _ int64) error                 { return nil }

type reminderCall struct {
	taskID   int64
	remindAt string
}

type mockReminderRepo struct {
	calls []reminderCall
}

func (m *mockReminderRepo) Create(_ context.Context, taskID int64, remindAt string) (int64, error) {
	m.calls = append(m.calls, reminderCall{taskID, remindAt})
	return int64(len(m.calls)), nil
}
func (m *mockReminderRepo) GetPending(_ context.Context) ([]domain.PendingReminder, error) {
	return nil, nil
}
func (m *mockReminderRepo) MarkSent(_ context.Context, _ int64) error { return nil }

func newModule(taskRepo *mockTaskRepo, reminderRepo *mockReminderRepo) *Module {
	return &Module{
		repo:      taskRepo,
		reminders: reminderRepo,
		logger:    charm.New(io.Discard),
	}
}

// ── compile-time check ──────────────────────────────────────────────────────

var _ agent.DataReader = (*Module)(nil)
var _ agent.DataWriter = (*Module)(nil)

// ── tests ────────────────────────────────────────────────────────────────────

func TestWriteCreateTask_WithoutReminders(t *testing.T) {
	m := newModule(&mockTaskRepo{createID: 42}, &mockReminderRepo{})

	result, err := m.Write(context.Background(), "create_task", map[string]any{
		"name":     "Comprar pan",
		"deadline": "2026-05-14T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("Write(create_task) error = %v, want nil", err)
	}
	if result["id"] != int64(42) {
		t.Errorf("Write(create_task) id = %v, want 42", result["id"])
	}
	if result["name"] != "Comprar pan" {
		t.Errorf("Write(create_task) name = %q, want %q", result["name"], "Comprar pan")
	}
}

func TestWriteCreateTask_WithReminders(t *testing.T) {
	reminderRepo := &mockReminderRepo{}
	m := newModule(&mockTaskRepo{createID: 7}, reminderRepo)

	_, err := m.Write(context.Background(), "create_task", map[string]any{
		"name":      "Reunión importante",
		"deadline":  "2026-05-01T00:00:00Z",
		"reminders": []any{"2026-04-24T09:00:00Z", "2026-04-25T09:00:00Z"},
	})
	if err != nil {
		t.Fatalf("Write(create_task) error = %v, want nil", err)
	}

	if len(reminderRepo.calls) != 2 {
		t.Fatalf("reminders.Create calls = %d, want 2", len(reminderRepo.calls))
	}
	wantReminders := []string{"2026-04-24T09:00:00Z", "2026-04-25T09:00:00Z"}
	for i, c := range reminderRepo.calls {
		if c.taskID != 7 {
			t.Errorf("reminders.Create[%d].taskID = %d, want 7", i, c.taskID)
		}
		if c.remindAt != wantReminders[i] {
			t.Errorf("reminders.Create[%d].remindAt = %q, want %q", i, c.remindAt, wantReminders[i])
		}
	}
}

func TestWriteCreateTask_WithDescription(t *testing.T) {
	m := newModule(&mockTaskRepo{createID: 10}, &mockReminderRepo{})

	result, err := m.Write(context.Background(), "create_task", map[string]any{
		"name":        "Preparar presentación",
		"deadline":    "2026-06-01T00:00:00Z",
		"description": "Incluir gráficos de ventas del Q1",
	})
	if err != nil {
		t.Fatalf("Write(create_task) error = %v, want nil", err)
	}
	if result["id"] != int64(10) {
		t.Errorf("Write(create_task) id = %v, want 10", result["id"])
	}
}

func TestWriteUpdateTaskStatus(t *testing.T) {
	m := newModule(&mockTaskRepo{}, &mockReminderRepo{})

	result, err := m.Write(context.Background(), "update_task_status", map[string]any{
		"id":     float64(1),
		"status": "DONE",
	})
	if err != nil {
		t.Fatalf("Write(update_task_status) error = %v, want nil", err)
	}
	if result["ok"] != true {
		t.Errorf("Write(update_task_status) ok = %v, want true", result["ok"])
	}
	if result["status"] != "DONE" {
		t.Errorf("Write(update_task_status) status = %q, want %q", result["status"], "DONE")
	}
}

func TestWriteDeleteTask(t *testing.T) {
	m := newModule(&mockTaskRepo{}, &mockReminderRepo{})

	result, err := m.Write(context.Background(), "delete_task", map[string]any{
		"id": float64(5),
	})
	if err != nil {
		t.Fatalf("Write(delete_task) error = %v, want nil", err)
	}
	if result["ok"] != true {
		t.Errorf("Write(delete_task) ok = %v, want true", result["ok"])
	}
}

func TestWriteUnknownTool_ReturnsError(t *testing.T) {
	m := newModule(&mockTaskRepo{}, &mockReminderRepo{})

	_, err := m.Write(context.Background(), "nonexistent_tool", nil)
	if err == nil {
		t.Fatal("Write(nonexistent) error = nil, want error")
	}
}
