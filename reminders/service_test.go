package reminders

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

// ── mocks ────────────────────────────────────────────────────────────────────

type mockReminderRepo struct {
	pending    []domain.PendingReminder
	pendingErr error
	markCalls  []int64
}

func (m *mockReminderRepo) Create(_ context.Context, _ int64, _ string) (int64, error) {
	return 0, nil
}
func (m *mockReminderRepo) GetPending(_ context.Context) ([]domain.PendingReminder, error) {
	return m.pending, m.pendingErr
}
func (m *mockReminderRepo) MarkSent(_ context.Context, id int64) error {
	m.markCalls = append(m.markCalls, id)
	return nil
}

type mockMessenger struct {
	sent []agent.Message
	err  error
}

func (m *mockMessenger) Send(_ context.Context, msg agent.Message) error {
	m.sent = append(m.sent, msg)
	return m.err
}

const testJID = "56912345678@s.whatsapp.net"

func newTestService(repo domain.ReminderRepository, msg agent.Messenger) *Service {
	return New(repo, msg, charm.New(io.Discard), testJID)
}

// ── tests ────────────────────────────────────────────────────────────────────

func TestDispatch_SendsAndMarks(t *testing.T) {
	repo := &mockReminderRepo{
		pending: []domain.PendingReminder{
			{ID: 5, TaskID: 10, TaskName: "Revisar caja"},
		},
	}
	messenger := &mockMessenger{}

	newTestService(repo, messenger).dispatch(context.Background())

	if len(messenger.sent) != 1 {
		t.Fatalf("messenger.Send calls = %d, want 1", len(messenger.sent))
	}
	if messenger.sent[0].Recipient() != testJID {
		t.Errorf("Send recipient = %q, want %q", messenger.sent[0].Recipient(), testJID)
	}
	content, _ := messenger.sent[0].Payload().(string)
	if !strings.Contains(content, "Revisar caja") {
		t.Errorf("message content %q missing task name", content)
	}
	if len(repo.markCalls) != 1 || repo.markCalls[0] != 5 {
		t.Errorf("MarkSent calls = %v, want [5]", repo.markCalls)
	}
}

func TestDispatch_NoReminders_NothingSent(t *testing.T) {
	repo := &mockReminderRepo{pending: nil}
	messenger := &mockMessenger{}

	newTestService(repo, messenger).dispatch(context.Background())

	if len(messenger.sent) != 0 {
		t.Errorf("messenger.Send calls = %d, want 0 (no pending reminders)", len(messenger.sent))
	}
}

func TestDispatch_SendError_MarkSentNotCalled(t *testing.T) {
	repo := &mockReminderRepo{
		pending: []domain.PendingReminder{
			{ID: 3, TaskID: 7, TaskName: "Inventario"},
		},
	}
	messenger := &mockMessenger{err: errors.New("whatsapp offline")}

	newTestService(repo, messenger).dispatch(context.Background())

	if len(repo.markCalls) != 0 {
		t.Errorf("MarkSent calls = %v, want none (send failed)", repo.markCalls)
	}
}

func TestDispatch_MultipleReminders_SendsAll(t *testing.T) {
	repo := &mockReminderRepo{
		pending: []domain.PendingReminder{
			{ID: 1, TaskID: 1, TaskName: "Tarea A"},
			{ID: 2, TaskID: 2, TaskName: "Tarea B"},
		},
	}
	messenger := &mockMessenger{}

	newTestService(repo, messenger).dispatch(context.Background())

	if len(messenger.sent) != 2 {
		t.Fatalf("messenger.Send calls = %d, want 2", len(messenger.sent))
	}
	if len(repo.markCalls) != 2 {
		t.Fatalf("MarkSent calls = %d, want 2", len(repo.markCalls))
	}
	if repo.markCalls[0] != 1 || repo.markCalls[1] != 2 {
		t.Errorf("MarkSent calls = %v, want [1 2]", repo.markCalls)
	}
}
