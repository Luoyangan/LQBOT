// Package ark demonstrates sending ark template messages.
// Ark messages use pre-registered templates on QQ Open Platform.
// Default built-in templates: 23 (link+text list), 24 (text+thumbnail), 37 (big image).
//
// NOTE: Replace TemplateID with your actual registered template ID.
package ark

import (
	"fmt"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds the ark command with sub-commands for each scene.
// Demonstrates: SendArkMessage, SendGroupArkMessage, SendC2CArkMessage,
// flat KV and nested ObjKV.
func Register(r contract.CommandRegister, api contract.QQAPI) {
	r.Register(contract.Command{
		Name:        "ark",
		Description: "发送 Ark 模板消息（支持频道/群聊/C2C）",
		Usage:       "ark <channel|group|c2c> [<用户ID>]",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.ArgCount() == 0 {
				return ctx.Reply("用法:\n" +
					"  /ark channel           → 在当前频道发 Ark\n" +
					"  /ark group             → 在当前群发 Ark\n" +
					"  /ark c2c <用户ID>       → 给指定用户发 Ark（私聊）\n\n" +
					"模板 ID 默认为 23（链接+文本列表），可在管理端注册自定义模板后修改。")
			}

			scene := ctx.Arg(0)

			// 构建 Ark 消息（template_id 23: 链接+文本列表模板）
			// 官方文档: https://bot.q.qq.com/wiki/develop/api-v2/server-inter/message/type/template/template_23.html
			// 可用键: #DESC#(描述), #PROMPT#(提示), #LIST#(数组, 含 desc/link)
			ark := &contract.MessageArk{
				TemplateID: 23,
				KV: []contract.ArkKV{
					{Key: "#DESC#", Value: "机器人订阅消息"},
					{Key: "#PROMPT#", Value: "LQBOT 机器人"},
					{
						Key: "#LIST#",
						Obj: []contract.ArkObj{
							{ObjKV: []contract.ArkObjKV{
								{Key: "desc", Value: "需求标题：UI问题解决"},
							}},
							{ObjKV: []contract.ArkObjKV{
								{Key: "desc", Value: "当前状态：体验中"},
							}},
							{ObjKV: []contract.ArkObjKV{
								{Key: "desc", Value: "已评估"},
								{Key: "link", Value: "https://qun.qq.com"},
							}},
							{ObjKV: []contract.ArkObjKV{
								{Key: "desc", Value: "已排期"},
								{Key: "link", Value: "https://qun.qq.com"},
							}},
							{ObjKV: []contract.ArkObjKV{
								{Key: "desc", Value: "开发中"},
								{Key: "link", Value: "https://qun.qq.com"},
							}},
						},
					},
				},
			}

			switch scene {
			case "channel":
				if ctx.Scene() != contract.SceneGuild {
					return ctx.Reply("频道场景才能使用 /ark channel，当前不在此场景。")
				}
				if err := ctx.ReplyArk(ark); err != nil {
					return ctx.Reply(fmt.Sprintf("频道 Ark 发送失败: %s", err))
				}
				return ctx.Reply("频道 Ark 消息已发送！")

			case "group":
				if ctx.Scene() != contract.SceneGroup {
					return ctx.Reply("群聊场景才能使用 /ark group。")
				}
				if err := ctx.ReplyArk(ark); err != nil {
					return ctx.Reply(fmt.Sprintf("群聊 Ark 发送失败: %s", err))
				}
				return ctx.Reply("群聊 Ark 消息已发送！")

			case "c2c":
				if ctx.ArgCount() < 2 {
					return ctx.Reply("请提供用户 ID。用法: /ark c2c <用户ID>")
				}
				userID := ctx.Arg(1)
				if err := api.SendC2CArkMessage(userID, ark); err != nil {
					return ctx.Reply(fmt.Sprintf("C2C Ark 发送失败: %s", err))
				}
				return ctx.Reply(fmt.Sprintf("已向 %s 发送 C2C Ark 消息！", userID))

			default:
				return ctx.Reply(fmt.Sprintf("未知场景: %s，请用 channel / group / c2c。", scene))
			}
		},
	})
}
