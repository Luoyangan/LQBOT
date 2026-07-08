// Package demo demonstrates the Plugin interface lifecycle pattern.
// It implements contract.Plugin and receives full PluginContext during Init().
package demo

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Demo implements contract.Plugin.
type Demo struct {
	config map[string]interface{}
}

// New creates a Demo plugin instance.
func New() *Demo {
	return &Demo{}
}

// Name returns the unique plugin identifier.
// Used to look up plugin-specific config from config.yaml: plugins.demo.*
func (d *Demo) Name() string { return "demo" }

// Init is called during bot startup with the full PluginContext.
func (d *Demo) Init(pc *contract.PluginContext) error {
	// 1. Read plugin-specific config from config.yaml
	if pc.PluginConfig != nil {
		if cfg, ok := pc.PluginConfig.(map[string]interface{}); ok {
			d.config = cfg
			pc.Logger.Info("demo plugin config loaded", "config", cfg)
		}
	}

	// 2. Register commands via PluginContext.Commands
	pc.Commands.Register(contract.Command{
		Name:        "demo",
		Aliases:     []string{"plugindemo"},
		Description: "Plugin 接口生命周期示例",
		Usage:       "demo [config|storage|ping]",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.ArgCount() == 0 {
				return ctx.Reply("demo 插件使用 Plugin 接口初始化\n用法: /demo config | /demo storage | /demo ping")
			}

			switch ctx.Arg(0) {
			case "config":
				return handleConfig(ctx, d.config)
			case "storage":
				return handleStorage(ctx, pc)
			case "ping":
				return ctx.Reply("demo 插件运行正常")
			default:
				return ctx.Reply("未知子命令: " + ctx.Arg(0))
			}
		},
	})

	// 3. Register HTTP routes if the HTTP server is available
	if pc.HTTPServer != nil {
		pc.HTTPServer.Handle("/demo", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte("demo plugin\n"))
		})
		pc.Logger.Info("demo: HTTP route registered", "path", "/demo")
	}

	// 4. Subscribe to non-message events
	pc.Listeners.Subscribe(contract.Listener{
		Event: "MESSAGE_DELETE",
		Handler: func(ctx contract.EventContext) error {
			pc.Logger.Debug("demo: message deleted", "msg_id", ctx.MessageID())
			return nil
		},
	})

	pc.Logger.Info("demo plugin initialized via Plugin interface")
	return nil
}

// handleConfig displays the plugin's config from config.yaml.
func handleConfig(ctx contract.CommandContext, cfg map[string]interface{}) error {
	if cfg == nil || len(cfg) == 0 {
		return ctx.Reply("demo 插件未配置 config.yaml 的 plugins.demo 段")
	}

	var b strings.Builder
	b.WriteString("📋 demo 插件配置\n")
	b.WriteString("━━━━━━━━━━━━━━\n")
	for k, v := range cfg {
		b.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
	}
	return ctx.Reply(b.String())
}

// handleStorage demonstrates direct Storage access via PluginContext.
func handleStorage(ctx contract.CommandContext, pc *contract.PluginContext) error {
	key := "demo:last_run"
	if err := pc.Storage.Set(key, "demo plugin was used"); err != nil {
		return ctx.Reply("写入 storage 失败: " + err.Error())
	}

	var val string
	_ = pc.Storage.Get(key, &val)
	return ctx.Reply("storage 操作成功\n  key: " + key + "\n  value: " + val)
}
