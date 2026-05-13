package tasks

import "github.com/carlospereira5/PersonalAssistant/internal/domain"

func marshalTask(t domain.Task) map[string]any {
	return map[string]any{
		"id":          t.ID,
		"name":        t.Name,
		"deadline":    t.Deadline.Format("2006-01-02"),
		"status":      t.Status,
		"description": t.Description,
	}
}

func marshalTasks(tasks []domain.Task) []map[string]any {
	out := make([]map[string]any, len(tasks))
	for i, t := range tasks {
		out[i] = marshalTask(t)
	}
	return out
}
