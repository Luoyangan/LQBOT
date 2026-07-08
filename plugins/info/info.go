// Package info demonstrates querying QQ API for guild, channel, and member info.
package info

import (
	"fmt"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds the server, whoami, and avatar commands.
// Demonstrates: GetGuild, GetChannel, GetGuildMember, AppID(), scene-aware API selection.
func Register(r contract.CommandRegister, api contract.QQAPI) {
	// /server — shows current guild/channel info
	r.Register(contract.Command{
		Name:        "server",
		Description: "查看当前频道/群聊信息",
		Usage:       "server",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.Scene() == contract.SceneGroup {
				return ctx.Reply(fmt.Sprintf("当前群 ID: %s", ctx.GroupID()))
			}

			// Fetch guild (server) info
			guild, err := api.GetGuild(ctx.GuildID())
			if err != nil {
				return ctx.Reply("获取频道信息失败: " + err.Error())
			}

			// Fetch current channel info
			channel, err := api.GetChannel(ctx.ChannelID())
			if err != nil {
				return ctx.Reply("获取子频道信息失败: " + err.Error())
			}

			reply := fmt.Sprintf("📗 服务器: %s\n👃 成员: %d\n📌 当前频道: %s",
				guild.Name, guild.MemberCount, channel.Name)
			return ctx.Reply(reply)
		},
	})

	// /whoami — shows your member info in the current guild
	r.Register(contract.Command{
		Name:        "whoami",
		Description: "查看你在当前频道的信息",
		Usage:       "whoami",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.Scene() != contract.SceneGuild {
				return ctx.Reply("该命令仅支持文字子频道场景。")
			}

			member, err := api.GetGuildMember(ctx.GuildID(), ctx.AuthorID())
			if err != nil {
				return ctx.Reply("获取成员信息失败: " + err.Error())
			}

			reply := fmt.Sprintf("🙁 昵称: %s\n↘ 用户 ID: %s\n📮 加入时间: %s",
				member.Nick, member.User.ID, member.JoinedAt)
			return ctx.Reply(reply)
		},
	})

	// /avatar — shows your QQ avatar via https://q.qlogo.cn/qqapp/{appid}/{openid}/640
	// This endpoint returns the user's QQ headshot thumbnail (640x640).
	r.Register(contract.Command{
		Name:        "avatar",
		Description: "获取你的 QQ 头像",
		Usage:       "avatar",
		Handler: func(ctx contract.CommandContext) error {
			// Build the avatar URL: https://q.qlogo.cn/qqapp/{appid}/{openid}/640
			//   - appid: bot's App ID from config
			//   - openid: user's QQ OpenID (ctx.AuthorID())
			//   - 640: image size in pixels
			appID := api.AppID()
			openID := ctx.AuthorID()
			avatarURL := fmt.Sprintf("https://q.qlogo.cn/qqapp/%s/%s/640", appID, openID)

			switch ctx.Scene() {
			case contract.SceneGroup:
				// 群聊: 通过 RichMedia 发送头像图片（需在消息 URL 配置中注册 q.qlogo.cn）
				return api.SendGroupRichMedia(ctx.GroupID(), &contract.RichMedia{
					FileType: 1,
					URL:      avatarURL,
					Content:  "你的 QQ 头像",
					MsgID:    ctx.MessageID(),
				})
			case contract.SceneC2C:
				return api.SendC2CRichMedia(ctx.AuthorID(), &contract.RichMedia{
					FileType: 1,
					URL:      avatarURL,
					Content:  "你的 QQ 头像",
					MsgID:    ctx.MessageID(),
				})
			default:
				// 频道: 使用 ReplyImage 被动回复发送头像
				return api.ReplyImage(ctx.ChannelID(), ctx.MessageID(), avatarURL)
			}
		},
	})
}
