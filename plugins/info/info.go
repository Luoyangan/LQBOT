// Package info demonstrates querying QQ API for guild, channel, and member info.
package info

import (
	"fmt"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds the server and whoami commands.
// Demonstrates: GetGuild, GetChannel, GetGuildMember, scene-aware API selection.
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
}
