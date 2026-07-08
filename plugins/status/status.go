// Package status provides runtime bot diagnostics (/status command).
package status

import (
	"fmt"
	"runtime"
	"time"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// StatusFunc returns a formatted status string.
type StatusFunc func() string

func Register(r contract.CommandRegister, fn StatusFunc) {
	r.Register(contract.Command{
		Name:        "status",
		Aliases:     []string{"stats", "botinfo"},
		Description: "查看机器人运行状态",
		Usage:       "status",
		Handler: func(ctx contract.CommandContext) error {
			return ctx.Reply(fn())
		},
	})
}

// BuildStatus creates a status function from runtime stats.
func BuildStatus(startTime time.Time, version string, logLevelFn func() string) StatusFunc {
	return func() string {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		uptime := time.Since(startTime)
		days := int(uptime.Hours()) / 24
		hours := int(uptime.Hours()) % 24
		mins := int(uptime.Minutes()) % 60

		uptimeStr := fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, mins)
		if days == 0 && hours == 0 {
			uptimeStr = fmt.Sprintf("%d分钟", mins)
		}

		return fmt.Sprintf(
			"🤖 LQBOT 运行状态\n"+
				"━━━━━━━━━━━━━\n"+
				"版本: %s\n"+
				"运行时间: %s\n"+
				"协程数: %d\n"+
				"内存使用: %.1f MB\n"+
				"日志级别: %s\n"+
				"━━━━━━━━━━━━━",
			version,
			uptimeStr,
			runtime.NumGoroutine(),
			float64(m.Alloc)/1024/1024,
			logLevelFn(),
		)
	}
}
