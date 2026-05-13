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
// El JID se obtiene automáticamente de la DB (config table) — el bot lo guarda
// cuando recibe el primer mensaje de un número autorizado.
type Service struct {
	repo       domain.ReminderRepository
	configRepo domain.ConfigRepository
	messenger  agent.Messenger
	logger     *charm.Logger
}

// New crea un Service. El JID del admin se lee de configRepo en cada ciclo.
func New(repo domain.ReminderRepository, configRepo domain.ConfigRepository, messenger agent.Messenger, logger *charm.Logger) *Service {
	return &Service{repo: repo, configRepo: configRepo, messenger: messenger, logger: logger}
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
	adminJID, err := s.configRepo.Get(ctx, "admin_jid")
	if err != nil {
		s.logger.Warn("Admin JID no configurado — esperando primer mensaje del usuario")
		return
	}

	pending, err := s.repo.GetPending(ctx)
	if err != nil {
		s.logger.Error("Error obteniendo recordatorios pendientes", "err", err)
		return
	}
	for _, pr := range pending {
		msg := fmt.Sprintf("⏰ Recordatorio: *%s*", pr.TaskName)
		if err := s.messenger.Send(ctx, agent.TextMessage{To: adminJID, Content: msg}); err != nil {
			s.logger.Error("Error enviando recordatorio", "reminder_id", pr.ID, "task_id", pr.TaskID, "err", err)
			continue
		}
		if err := s.repo.MarkSent(ctx, pr.ID); err != nil {
			s.logger.Error("Error marcando recordatorio como enviado", "reminder_id", pr.ID, "err", err)
		}
	}
}
