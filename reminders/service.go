// Package reminders — servicio de notificaciones programadas de tareas.
package reminders

import (
	"context"
	"fmt"
	"time"

	charm "github.com/charmbracelet/log"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

const tickInterval = 30 * time.Second

// Service revisa periódicamente los recordatorios pendientes y los envía
// por el Messenger configurado al JID del administrador.
type Service struct {
	repo      domain.ReminderRepository
	messenger agent.Messenger
	logger    *charm.Logger
	adminJID  string // JID de WhatsApp del administrador
}

// New crea un Service que envía recordatorios al JID indicado.
func New(repo domain.ReminderRepository, messenger agent.Messenger, logger *charm.Logger, adminJID string) *Service {
	return &Service{repo: repo, messenger: messenger, logger: logger, adminJID: adminJID}
}

// Start bloquea hasta que ctx sea cancelado. Debe llamarse en una goroutine.
func (s *Service) Start(ctx context.Context) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.dispatch(ctx)
		}
	}
}

func (s *Service) dispatch(ctx context.Context) {
	pending, err := s.repo.GetPending(ctx)
	if err != nil {
		s.logger.Error("Error obteniendo recordatorios pendientes", "err", err)
		return
	}
	for _, pr := range pending {
		msg := fmt.Sprintf("⏰ Recordatorio: *%s*", pr.TaskName)
		if err := s.messenger.Send(ctx, agent.TextMessage{To: s.adminJID, Content: msg}); err != nil {
			s.logger.Error("Error enviando recordatorio", "reminder_id", pr.ID, "task_id", pr.TaskID, "err", err)
			continue
		}
		if err := s.repo.MarkSent(ctx, pr.ID); err != nil {
			s.logger.Error("Error marcando recordatorio como enviado", "reminder_id", pr.ID, "err", err)
		}
	}
}
