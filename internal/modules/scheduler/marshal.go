package scheduler

import (
	"time"

	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

func marshalRoutine(r domain.ScheduledRoutine) map[string]any {
	m := map[string]any{
		"id":         r.ID,
		"cron":       r.CronExpr,
		"prompt":     r.Prompt,
		"status":     r.Status,
		"last_error": r.LastError,
		"created_at": r.CreatedAt.Format(time.RFC3339),
	}
	if r.LastRunAt != nil {
		m["last_run_at"] = r.LastRunAt.Format(time.RFC3339)
	}
	return m
}

func marshalRoutines(routines []domain.ScheduledRoutine) []map[string]any {
	out := make([]map[string]any, len(routines))
	for i, r := range routines {
		out[i] = marshalRoutine(r)
	}
	return out
}
