package scheduler

import (
	"testing"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/carlospereira5/PersonalAssistant/internal/domain"
)

func newParser() cron.Parser {
	return cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
}

func TestIsDue(t *testing.T) {
	parser := newParser()
	now := time.Date(2026, 5, 14, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		routine domain.ScheduledRoutine
		now     time.Time
		want    bool
	}{
		{
			name: "never run, cron matches now",
			routine: domain.ScheduledRoutine{
				ID:       1,
				CronExpr: "0 8 * * *",
				Status:   "active",
			},
			now:  now,
			want: true,
		},
		{
			name: "never run, cron does not match now",
			routine: domain.ScheduledRoutine{
				ID:       2,
				CronExpr: "0 9 * * *",
				Status:   "active",
			},
			now:  now,
			want: false,
		},
		{
			name: "ran at scheduled time, next run in future",
			routine: domain.ScheduledRoutine{
				ID:        3,
				CronExpr:  "0 8 * * *",
				Status:    "active",
				LastRunAt: timePtr(time.Date(2026, 5, 14, 8, 0, 0, 0, time.UTC)),
			},
			now:  now.Add(30 * time.Second), // 30s later, same tick window
			want: false,
		},
		{
			name: "ran yesterday, next run is now",
			routine: domain.ScheduledRoutine{
				ID:        4,
				CronExpr:  "0 8 * * *",
				Status:    "active",
				LastRunAt: timePtr(time.Date(2026, 5, 13, 8, 0, 0, 0, time.UTC)),
			},
			now:  now,
			want: true,
		},
		{
			name: "paused, should not run",
			routine: domain.ScheduledRoutine{
				ID:       5,
				CronExpr: "0 8 * * *",
				Status:   "paused",
			},
			now:  now,
			want: true, // isDue doesn't check status — that's done in executeDue
		},
	}

	// We need a Module to call isDue. Create a minimal one.
	m := &Module{
		cronParser: parser,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schedule, err := m.cronParser.Parse(tt.routine.CronExpr)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			got := m.isDue(tt.routine, schedule, tt.now)
			if got != tt.want {
				t.Errorf("isDue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
