// Package markdown demonstrates markdown message sending in two modes:
//   - Custom markdown: 原生 markdown 文本
//   - Template markdown: 使用管理端注册的 markdown 模板
//
// 2026/04 起，单聊/群聊的自定义 markdown 已对所有机器人开放（无需申请）。
// 频道场景需内邀开通。
package markdown

import (
	"fmt"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// Register adds the md command for demonstrating custom, template,
// and interactive markdown.
// Demonstrates: SendMarkdown, SendGroupMarkdown, SendC2CMarkdown,
// SendMarkdownTemplate, SendGroupMarkdownTemplate, SendC2CMarkdownTemplate.
func Register(r contract.CommandRegister, api contract.QQAPI) {
	r.Register(contract.Command{
		Name:        "md",
		Description: "发送 Markdown 消息（自定义/模板/交互）",
		Usage:       "md <custom|template|interactive> [channel|group|c2c] [<用户ID>]",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.ArgCount() < 2 {
				return ctx.Reply("用法:\n" +
					"  /md custom channel           → 频道自定义 Markdown\n" +
					"  /md custom group             → 群聊自定义 Markdown\n" +
					"  /md custom c2c <用户ID>       → C2C 自定义 Markdown\n" +
					"  /md interactive channel       → 频道参数指令 Markdown\n" +
					"  /md interactive group         → 群聊参数指令 Markdown\n" +
					"  /md interactive c2c <用户ID>  → C2C 指令操作 Markdown\n" +
					"  /md template channel          → 频道模板 Markdown\n" +
					"  /md template group            → 群聊模板 Markdown\n" +
					"  /md template c2c <用户ID>     → C2C 模板 Markdown\n\n" +
					"模板 markdown 需先在管理端注册 custom_template_id。\n" +
					"参数指令(CmdInput)所有场景支持，回车指令(CmdEnter)仅 C2C。")
			}

			mode := ctx.Arg(0)
			scene := ctx.Arg(1)

			switch mode {
			case "custom":
				return sendCustomMD(ctx, api, scene)
			case "interactive":
				return sendInteractiveMD(ctx, api, scene)
			case "template":
				return sendTemplateMD(ctx, api, scene)
			default:
				return ctx.Reply(fmt.Sprintf("未知模式: %s，请用 custom / interactive / template。", mode))
			}
		},
	})

	r.Register(contract.Command{
		Name:        "mda",
		Description: "发送 Markdown 代码消息",
		Usage:       "mda",
		Handler: func(ctx contract.CommandContext) error {
			md := "## 代码演示\n\n" +
				"```markdown\n" +
				"# 标题\n" +
				"**加粗文字**\n" +
				"- 列表项\n" +
				"```\n" +
				"```python\n" +
				"print('hello world')\n" +
				"```\n"
			return ctx.ReplyMarkdown(md)
		},
	})
}

func sendCustomMD(ctx contract.CommandContext, api contract.QQAPI, scene string) error {
	mention := contract.MentionUser(ctx.AuthorID())

	switch scene {
	case "channel":
		if ctx.Scene() != contract.SceneGuild {
			return ctx.Reply("频道场景才能使用 /md custom channel。")
		}
		md := "# LQBOT 机器人\n\n" +
			"**你好！** 频道自定义 Markdown。\n\n" +
			"## 图片演示\n" +
			"![示例图片](https://wd.lilei007.cn/Luoyangan/1.png)\n" +
			"> 图片 URL 需在 QQ 开放平台管理端素材管理中报备\n\n" +
			"## 文本链交互演示\n" +
			"1. @某人: " + mention + "\n" +
			"2. 参数指令: " + contract.CmdInput("/echo 你好", "点我输入", false) + "\n" +
			"3. 跳转子频道: " + contract.ChannelLink(ctx.ChannelID()) + "\n" +
			"4. 表情: " + contract.Emoji("4") + " " + contract.Emoji("14") + "\n\n" +
			"## 功能列表\n" +
			"1. 文本消息\n2. 图片消息\n3. Markdown 消息\n4. 按钮交互\n\n" +
			"> LQBOT 是基于 Go 开发的 QQ 机器人\n\n" +
			"[🔗 GitHub](https://github.com/Luoyangan/LQBOT)"
		if err := api.ReplyMarkdown(ctx.ChannelID(), ctx.MessageID(), md); err != nil {
			return ctx.Reply(fmt.Sprintf("频道 Markdown 发送失败: %s", err))
		}
		return ctx.Reply("频道自定义 Markdown 已发送！")

	case "group":
		if ctx.Scene() != contract.SceneGroup {
			return ctx.Reply("群聊场景才能使用 /md custom group。")
		}
		md := "# LQBOT 机器人\n\n" +
			"**你好！** 群聊自定义 Markdown。\n\n" +
			"## 图片演示\n" +
			"![示例图片](https://wd.lilei007.cn/Luoyangan/1.png)\n" +
			"> 图片 URL 需在 QQ 开放平台管理端素材管理中报备\n\n" +
			"## 文本链交互演示\n" +
			"1. @某人: " + mention + "\n" +
			"2. 参数指令: " + contract.CmdInput("/echo 你好", "点我输入", false) + "\n\n" +
			"## 功能列表\n" +
			"1. 文本消息\n2. 图片消息\n3. Markdown 消息\n4. 按钮交互\n\n" +
			"> LQBOT 是基于 Go 开发的 QQ 机器人\n\n" +
			"[🔗 GitHub](https://github.com/Luoyangan/LQBOT)"
		if err := ctx.ReplyMarkdown(md); err != nil {
			return ctx.Reply(fmt.Sprintf("群聊 Markdown 发送失败: %s", err))
		}
		return ctx.Reply("群聊自定义 Markdown 已发送！")

	case "c2c":
		if ctx.ArgCount() < 3 {
			return ctx.Reply("请提供用户 ID。用法: /md custom c2c <用户ID>")
		}
		userID := ctx.Arg(2)
		md := "# LQBOT 机器人\n\n" +
			"**你好！** C2C 自定义 Markdown。\n\n" +
			"## 图片演示\n" +
			"![示例图片](https://wd.lilei007.cn/Luoyangan/1.png)\n" +
			"> 图片 URL 需在 QQ 开放平台管理端素材管理中报备\n\n" +
			"## 文本链交互演示\n" +
			"1. @某人: " + mention + "\n" +
			"2. 回车指令: " + contract.CmdEnter("/ping") + "\n" +
			"3. 参数指令: " + contract.CmdInput("/echo 你好", "点我输入", false) + "\n\n" +
			"## 功能列表\n" +
			"1. 文本消息\n2. 图片消息\n3. Markdown 消息\n4. 按钮交互\n\n" +
			"[🔗 GitHub](https://github.com/Luoyangan/LQBOT)"
		if err := api.SendC2CMarkdown(userID, md); err != nil {
			return ctx.Reply(fmt.Sprintf("C2C Markdown 发送失败: %s", err))
		}
		return ctx.Reply(fmt.Sprintf("已向 %s 发送自定义 Markdown！", userID))

	default:
		return ctx.Reply(fmt.Sprintf("未知场景: %s，请用 channel / group / c2c。", scene))
	}
}

// sendInteractiveMD sends a markdown message demonstrating interactive elements.
// CmdInput（参数指令）: all scenes supported.
// CmdEnter（回车指令）: C2C only; group/text channel NOT supported per docs.
func sendInteractiveMD(ctx contract.CommandContext, api contract.QQAPI, scene string) error {
	switch scene {
	case "channel":
		if ctx.Scene() != contract.SceneGuild {
			return ctx.Reply("频道场景才能使用 /md interactive channel。")
		}
		cmdEcho := contract.CmdInput("/echo 你好", "点我输入", false)
		cmdRef := contract.CmdInput("/echo 收到", "带引用回复", true)
		md := "# LQBOT 参数指令演示\n\n" +
			"点击下面的标签即可体验：\n\n" +
			"## 参数指令\n" +
			"点击插入输入框:\n" + cmdEcho + "\n\n" +
			"## 带引用参数指令\n" +
			cmdRef + "\n\n" +
			"> 参数指令(CmdInput)所有场景均支持。回车指令(CmdEnter)仅 C2C 场景。"
		if err := api.ReplyMarkdown(ctx.ChannelID(), ctx.MessageID(), md); err != nil {
			return ctx.Reply(fmt.Sprintf("频道交互 Markdown 发送失败: %s", err))
		}
		return ctx.Reply("频道参数指令 Markdown 已发送！")

	case "group":
		if ctx.Scene() != contract.SceneGroup {
			return ctx.Reply("群聊场景才能使用 /md interactive group。")
		}
		cmdEcho := contract.CmdInput("/echo 你好", "点我输入", false)
		cmdRef := contract.CmdInput("/echo 收到", "带引用回复", true)
		md := "# LQBOT 参数指令演示\n\n" +
			"点击下面的标签即可体验：\n\n" +
			"## 参数指令\n" +
			"点击插入输入框:\n" + cmdEcho + "\n\n" +
			"## 带引用参数指令\n" +
			cmdRef + "\n\n" +
			"> 参数指令(CmdInput)所有场景均支持。回车指令(CmdEnter)仅 C2C 场景。"
		if err := ctx.ReplyMarkdown(md); err != nil {
			return ctx.Reply(fmt.Sprintf("群聊交互 Markdown 发送失败: %s", err))
		}
		return ctx.Reply("群聊参数指令 Markdown 已发送！")

	case "c2c":
		if ctx.ArgCount() < 3 {
			return ctx.Reply("请提供用户 ID。用法: /md interactive c2c <用户ID>")
		}
		userID := ctx.Arg(2)
		cmdPing := contract.CmdEnter("/ping")
		cmdEcho := contract.CmdInput("/echo 你好", "点我输入", false)
		cmdRef := contract.CmdInput("/echo 收到", "带引用回复", true)
		md := "# LQBOT 指令操作演示\n\n" +
			"点击下面的标签即可体验：\n\n" +
			"## 回车指令\n" +
			"点击直接发送 `/ping`:\n" + cmdPing + "\n\n" +
			"## 参数指令\n" +
			"点击插入输入框:\n" + cmdEcho + "\n\n" +
			"## 带引用参数指令\n" +
			cmdRef + "\n\n" +
			"> 回车指令仅 C2C 支持，参数指令所有场景均支持。"
		if err := api.SendC2CMarkdown(userID, md); err != nil {
			return ctx.Reply(fmt.Sprintf("C2C 交互 Markdown 发送失败: %s", err))
		}
		return ctx.Reply(fmt.Sprintf("已向 %s 发送指令操作 Markdown！", userID))

	default:
		return ctx.Reply(fmt.Sprintf("未知场景: %s，请用 channel / group / c2c。", scene))
	}
}

func sendTemplateMD(ctx contract.CommandContext, api contract.QQAPI, scene string) error {
	templateID := "YOUR_TEMPLATE_ID"
	params := []contract.MarkdownParam{
		{Key: "title", Values: []string{"LQBOT 通知"}},
		{Key: "desc", Values: []string{"这是一条模板 Markdown 消息"}},
		{Key: "link", Values: []string{"https://github.com/Luoyangan/LQBOT"}},
	}

	switch scene {
	case "channel":
		if ctx.Scene() != contract.SceneGuild {
			return ctx.Reply("频道场景才能使用 /md template channel。")
		}
		if err := ctx.ReplyMarkdownTemplate(templateID, params); err != nil {
			return ctx.Reply(fmt.Sprintf("频道模板 Markdown 发送失败: %s", err))
		}
		return ctx.Reply("频道模板 Markdown 已发送！")

	case "group":
		if ctx.Scene() != contract.SceneGroup {
			return ctx.Reply("群聊场景才能使用 /md template group。")
		}
		if err := ctx.ReplyMarkdownTemplate(templateID, params); err != nil {
			return ctx.Reply(fmt.Sprintf("群聊模板 Markdown 发送失败: %s", err))
		}
		return ctx.Reply("群聊模板 Markdown 已发送！")

	case "c2c":
		if ctx.ArgCount() < 3 {
			return ctx.Reply("请提供用户 ID。用法: /md template c2c <用户ID>")
		}
		userID := ctx.Arg(2)
		if err := api.SendC2CMarkdownTemplate(userID, templateID, params); err != nil {
			return ctx.Reply(fmt.Sprintf("C2C 模板 Markdown 发送失败: %s", err))
		}
		return ctx.Reply(fmt.Sprintf("已向 %s 发送模板 Markdown！", userID))

	default:
		return ctx.Reply(fmt.Sprintf("未知场景: %s，请用 channel / group / c2c。", scene))
	}
}
