// Package embed demonstrates sending embed (rich card) messages.
// Embed messages are only supported in text channels (not group/C2C).
package embed

import (
	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds the embed command.
// Demonstrates: SendEmbedMessage, scene-check guard.
func Register(r contract.CommandRegister, api contract.QQAPI) {
	r.Register(contract.Command{
		Name:        "embed",
		Description: "发送一张富卡片消息（仅频道）",
		Usage:       "embed",
		Handler: func(ctx contract.CommandContext) error {
			// Embed only works in text channels
			if ctx.Scene() != contract.SceneGuild {
				return ctx.Reply("Embed 消息仅支持文字子频道场景。")
			}

			return api.SendEmbedMessage(ctx.ChannelID(), &contract.MessageEmbed{
				Title:       "等级提升",
				Prompt:      "收到一条新通知",
				Description: "恭喜你达到黄金段位！",
				Thumbnail:   "https://example.com/badge.png",
				Fields: []contract.EmbedField{
					{Name: "当前段位：黄金 III"},
					{Name: "胜点：85 / 100"},
					{Name: "最近对局：3 胜 1 负"},
				},
			})
		},
	})
}
