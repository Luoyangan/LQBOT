// Package loglevel provides runtime log level switching.
package loglevel

import (
	"github.com/Luoyangan/LQBOT/internal/contract"
	framelog "github.com/Luoyangan/LQBOT/internal/log"
	"github.com/Luoyangan/LQBOT/internal/types"
)

func Register(r contract.CommandRegister, logger *framelog.Logger) {
	r.Register(contract.Command{
		Name:        "loglevel",
		Aliases:     []string{"ll", "log-level"},
		Description: "查看或修改日志级别（trace/debug/info/warn/error）",
		Usage:       "loglevel [trace|debug|info|warn|error]",
		Permission:  "admin",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.ArgCount() == 0 {
				return ctx.Reply("当前日志级别: " + logger.Level())
			}

			level := types.LogLevel(ctx.Arg(0))
			switch level {
			case types.LogLevelTrace, types.LogLevelDebug, types.LogLevelInfo,
				types.LogLevelWarn, types.LogLevelError:
				prev := logger.SetLevel(level)
				return ctx.Reply("日志级别已从 " + prev + " 切换为 " + string(level))
			default:
				return ctx.Reply("无效级别，可用: trace / debug / info / warn / error")
			}
		},
	})
}
