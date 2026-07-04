// Package manage demonstrates message management operations:
// delete, pin, reactions, and C2C message sending.
package manage

import (
	"fmt"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds management commands for message operations.
// Demonstrates: DeleteMessage, PinMessage, CreateReaction.
func Register(r contract.CommandRegister, api contract.QQAPI) {
	// /delete — deletes/recalls a message
	r.Register(contract.Command{
		Name:        "delete",
		Description: "撤回一条消息",
		Usage:       "delete <消息ID>",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.ArgCount() == 0 {
				return ctx.Reply("用法: /delete <消息ID>")
			}
			msgID := ctx.Arg(0)

			var err error
			switch ctx.Scene() {
			case contract.SceneGuild:
				err = api.DeleteMessage(ctx.ChannelID(), msgID)
			case contract.SceneGroup:
				err = api.DeleteGroupMessage(ctx.GroupID(), msgID)
			case contract.SceneC2C:
				err = api.DeleteC2CMessage(ctx.AuthorID(), msgID)
			}
			if err != nil {
				return ctx.Reply("撤回失败: " + err.Error())
			}
			return ctx.Reply("消息已撤回。")
		},
	})

	// /react — add a reaction to the replied message
	r.Register(contract.Command{
		Name:        "react",
		Description: "给消息添加表情反应（仅频道）",
		Usage:       "react <表情ID>",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.Scene() != contract.SceneGuild {
				return ctx.Reply("表情反应仅支持文字子频道场景。")
			}
			if ctx.ArgCount() == 0 {
				return ctx.Reply("用法: /react <表情ID>\n例如: /react 1:4 或 /react 2:❤️")
			}

			emoji := ctx.Arg(0)
			// React to the message that the user is replying to
			// Since we don't have a replied message ID readily available,
			// react to the current message as a demo
			if err := api.CreateReaction(ctx.ChannelID(), ctx.MessageID(), emoji); err != nil {
				return ctx.Reply(fmt.Sprintf("添加表情失败: %s", err))
			}
			return ctx.Reply(fmt.Sprintf("已添加表情 %s", emoji))
		},
	})

	// /pin — pin a message (精华消息)
	r.Register(contract.Command{
		Name:        "pin",
		Description: "将消息设为精华（仅频道）",
		Usage:       "pin <消息ID>",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.Scene() != contract.SceneGuild {
				return ctx.Reply("精华消息仅支持文字子频道场景。")
			}
			if ctx.ArgCount() == 0 {
				return ctx.Reply("用法: /pin <消息ID>")
			}

			if err := api.PinMessage(ctx.ChannelID(), ctx.Arg(0)); err != nil {
				return ctx.Reply(fmt.Sprintf("设置精华失败: %s", err))
			}
			return ctx.Reply("已设为精华消息。")
		},
	})

	// /unpin — remove pin from a message
	r.Register(contract.Command{
		Name:        "unpin",
		Description: "取消精华消息（仅频道）",
		Usage:       "unpin <消息ID>",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.Scene() != contract.SceneGuild {
				return ctx.Reply("精华消息仅支持文字子频道场景。")
			}
			if ctx.ArgCount() == 0 {
				return ctx.Reply("用法: /unpin <消息ID>")
			}

			if err := api.UnpinMessage(ctx.ChannelID(), ctx.Arg(0)); err != nil {
				return ctx.Reply(fmt.Sprintf("取消精华失败: %s", err))
			}
			return ctx.Reply("已取消精华消息。")
		},
	})
}
