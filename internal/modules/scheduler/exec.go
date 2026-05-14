package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/carlospereira5/PersonalAssistant/agent"
	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

const llmTimeout = 30 * time.Second

// Start implements backgroundModule — runs the background execution loop.
func (m *Module) Start(ctx context.Context) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	m.logger.Info("Scheduler daemon iniciado", "interval", "60s")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			m.executeDue(ctx)
		}
	}
}

func (m *Module) executeDue(ctx context.Context) {
	adminJID, err := m.configRepo.Get(ctx, "admin_jid")
	if err != nil {
		m.logger.Warn("Admin JID no configurado — saltando ejecución de rutinas")
		return
	}

	routines, err := m.repo.GetAll(ctx)
	if err != nil {
		m.logger.Error("Error obteniendo rutinas", "err", err)
		return
	}

	now := time.Now()

	for _, r := range routines {
		if r.Status != "active" {
			continue
		}

		schedule, err := m.cronParser.Parse(r.CronExpr)
		if err != nil {
			m.logger.Error("Error parseando cron", "routine_id", r.ID, "cron", r.CronExpr, "err", err)
			continue
		}

		if !m.isDue(r, schedule, now) {
			continue
		}

		m.logger.Info("Ejecutando rutina", "id", r.ID, "cron", r.CronExpr)

		result, execErr := m.executePrompt(ctx, r)
		if execErr != nil {
			m.logger.Error("Error ejecutando rutina", "routine_id", r.ID, "err", execErr)
			if updateErr := m.repo.UpdateLastRun(ctx, r.ID, execErr.Error()); updateErr != nil {
				m.logger.Error("Error actualizando last_error", "routine_id", r.ID, "err", updateErr)
			}
			continue
		}

		msg := fmt.Sprintf("🤖 *Rutina: %s*\n%s", r.Prompt, result)
		if sendErr := m.messenger.Send(ctx, agent.TextMessage{To: adminJID, Content: msg}); sendErr != nil {
			m.logger.Error("Error enviando resultado de rutina", "routine_id", r.ID, "err", sendErr)
		}

		if updateErr := m.repo.UpdateLastRun(ctx, r.ID, ""); updateErr != nil {
			m.logger.Error("Error actualizando last_run_at", "routine_id", r.ID, "err", updateErr)
		}
	}
}

// isDue determina si una rutina debe ejecutarse en este tick.
func (m *Module) isDue(r domain.ScheduledRoutine, schedule cron.Schedule, now time.Time) bool {
	if r.LastRunAt != nil {
		nextAfterLast := schedule.Next(*r.LastRunAt)
		return !nextAfterLast.After(now)
	}

	nextFromMinuteAgo := schedule.Next(now.Add(-1 * time.Minute))
	return !nextFromMinuteAgo.After(now)
}

// executePrompt llama al LLM con el prompt de la rutina y retorna la respuesta.
// Usa BackgroundLLM si está configurado, para no consumir cuota del LLM principal.
func (m *Module) executePrompt(ctx context.Context, r domain.ScheduledRoutine) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, llmTimeout)
	defer cancel()

	session, err := m.bgLLM.NewSession(ctx, "", nil)
	if err != nil {
		return "", fmt.Errorf("error creando sesión LLM para rutina %d: %w", r.ID, err)
	}

	// Sin tools para ejecución background — es un prompt directo.
	text, _, err := session.Send(ctx, r.Prompt)
	if err != nil {
		return "", fmt.Errorf("error ejecutando prompt de rutina %d: %w", r.ID, err)
	}

	return text, nil
}
