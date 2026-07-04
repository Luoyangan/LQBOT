// Package ping implements the /ping command for testing bot responsiveness.
package ping

import (
	"fmt"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds the ping command to the given registrar.
// Demonstrates: basic command with alias, scene detection, usage info.
func Register(r contract.CommandRegister) {
	r.Register(contract.Command{
		Name:        "ping",
		Aliases:     []string{"p"},
		Description: "测试机器人是否在线",
		Usage:       "ping",
		Handler: func(ctx contract.CommandContext) error {
			// Build a response with scene context using the framework's Scene() and AuthorID()
			reply := fmt.Sprintf("Pong! (场景: %s, 发送者: %s)",
				sceneLabel(ctx.Scene()),
				ctx.AuthorID())
			return ctx.Reply(reply)
		},
	})
}

// sceneLabel returns a human-readable label for the message scene.
func sceneLabel(s contract.MessageScene) string {
	switch s {
	case contract.SceneGuild:
		return "频道"
	case contract.SceneGroup:
		return "群聊"
	case contract.SceneC2C:
		return "私聊"
	default:
		return "未知"
	}
}
