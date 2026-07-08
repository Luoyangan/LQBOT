// Package timer demonstrates scheduled task usage (cron + interval).
// It registers example tasks on startup and provides /timer management.
package timer

import (
	"fmt"
	"strings"
	"time"

	"github.com/Luoyangan/LQBOT/internal/contract"
	"github.com/Luoyangan/LQBOT/internal/log"
)

// taskRecord holds a registered task's info.
type taskRecord struct {
	name  string
	spec  string
	next  time.Time
}

var tasks []taskRecord

func Register(r contract.CommandRegister, sched contract.Scheduler, logger *log.Logger) {
	// --- Example 1: Interval task (every 10 minutes) ---
	_ = sched.Interval(10*time.Minute, func() {
		logger.Info("定时任务: 心跳", "msg", "bot is alive", "goroutines", "?")
	})
	tasks = append(tasks, taskRecord{name: "心跳", spec: "每10分钟", next: time.Now().Add(10 * time.Minute)})

	// --- Example 2: Cron task (every hour at :05) ---
	_ = sched.Every("0 5 * * *", func() {
		logger.Info("定时任务: 整点报告", "msg", "整点报告: bot 运行中", "time", time.Now().Format("15:04"))
	})
	tasks = append(tasks, taskRecord{name: "整点报告", spec: "0 5 * * * (每小时05分)", next: nextCron("0 5 * * *")})

	// --- Register /timer command ---
	r.Register(contract.Command{
		Name:        "timer",
		Aliases:     []string{"定时任务", "cron", "scheduler"},
		Description: "查看已注册的定时任务",
		Usage:       "timer",
		Handler: func(ctx contract.CommandContext) error {
			if len(tasks) == 0 {
				return ctx.Reply("当前没有运行中的定时任务")
			}

			var b strings.Builder
			b.WriteString("⏰ 定时任务列表\n")
			b.WriteString("━━━━━━━━━━━━━━\n")
			for _, t := range tasks {
				b.WriteString(fmt.Sprintf("• %s\n  调度: %s\n", t.name, t.spec))
			}
			b.WriteString("━━━━━━━━━━━━━━\n")
			b.WriteString("共 " + fmt.Sprintf("%d", len(tasks)) + " 个任务")
			return ctx.Reply(b.String())
		},
	})
}

// nextCron returns an approximate next run time for the given cron spec.
// This is a simplified version for display purposes.
func nextCron(spec string) time.Time {
	now := time.Now()
	// Simple parsing: "0 5 * * *" → next :05 of the next hour
	next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 5, 0, 0, now.Location())
	if next.Before(now) || next.Equal(now) {
		next = next.Add(1 * time.Hour)
	}
	return next
}
