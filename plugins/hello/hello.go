// Package hello implements greeting responses and interactive button demos.
// Demonstrates: event listener, command routing, button interaction, scene-aware API.
package hello

import (
	"strings"

	"github.com/Luoyangan/LQBOT/internal/contract"
	"github.com/Luoyangan/LQBOT/internal/types"
)

// Register adds the "你好" listener, button command, and button interaction handler.
func Register(r contract.CommandRegister, l contract.ListenerRegister, api contract.QQAPI) {
	// ── Event listener: responds to "你好" in all scenes (channel / group / C2C) ──
	l.Subscribe(contract.Listener{
		Event: types.EventMessageCreate,
		Handler: func(ctx contract.EventContext) error {
			if ctx.Content() == "你好" {
				return ctx.Reply("你好！我是 LQBOT")
			}
			return nil
		},
	})
	l.Subscribe(contract.Listener{
		Event: types.EventGroupMessageCreate,
		Handler: func(ctx contract.EventContext) error {
			if ctx.Content() == "你好" {
				return ctx.Reply("你好！我是 LQBOT")
			}
			return nil
		},
	})
	l.Subscribe(contract.Listener{
		Event: types.EventC2CMessageCreate,
		Handler: func(ctx contract.EventContext) error {
			if ctx.Content() == "你好" {
				return ctx.Reply("你好！我是 LQBOT")
			}
			return nil
		},
	})

	// ── 按钮交互演示 ──
	r.Register(contract.Command{
		Name:        "button",
		Description: "发送一个按钮交互消息",
		Usage:       "button",
		Handler: func(ctx contract.CommandContext) error {
			buttons := []contract.MessageButton{
				{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},
			}
			// 存储原始命令 msg_id 供按钮回调被动回复使用
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtons("请点击按钮：", buttons)
		},
	})

	// ── 按钮交互回调 ──
	l.Subscribe(contract.Listener{
		Event: types.EventInteractionCreate,
		Handler: func(ctx contract.EventContext) error {
			ic := ctx.(contract.InteractionContext)
			// 先确认交互（防止超时），再用 Reply 发送群聊可见消息
			_ = ic.DeferReply()
			switch ic.ButtonID() {
			case "btn_hello":
				return ic.Reply("你好！我是 LQBOT，很高兴认识你")
			case "btn_ping":
				return ic.Reply("Pong! 机器人运行正常 ✓")
			case "btn_info":
				return ic.Reply("LQBOT - 基于 Go 的 QQ 机器人\n技术栈: Go + botgo SDK + SQLite\n支持: 文本 / Markdown / Ark / 按钮交互")
			default:
				return ic.Reply("按钮已点击 (ID: " + ic.ButtonID() + ")")
			}
		},
	})

	r.Register(contract.Command{
		Name:        "button2",
		Description: "演示不同样式和行为的多个按钮",
		Usage:       "button2",
		Handler: func(ctx contract.CommandContext) error {
			buttons := []contract.MessageButton{
				{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},       // 蓝色主按钮
				{ID: "btn_ping", Label: "Ping", Data: "btn_ping", Style: 0, ActionType: 1},          // 灰色次要按钮
				{ID: "btn_info", Label: "机器人信息", Data: "btn_info", Style: 0, ActionType: 1},     // 灰色次要按钮
			}
			// 存储原始命令 msg_id 供按钮回调被动回复使用
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtons("请点击按钮：", buttons)
		},
	})

	// ── 命令：/buttons 演示多行按钮布局 ──
	// 展示 1 行 1 个、1 行 2 个、1 行 3 个共 3 种布局
	r.Register(contract.Command{
		Name:        "buttons",
		Description: "演示多行按钮布局（1个/2个/3个按钮每行）",
		Usage:       "buttons",
		Handler: func(ctx contract.CommandContext) error {
			rows := [][]contract.MessageButton{
				// 第 1 行：1 个按钮（蓝色主按钮）
				{
					{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},
				},
				// 第 2 行：2 个按钮（灰色次要按钮）
				{
					{ID: "btn_ping", Label: "Ping", Data: "btn_ping", Style: 0, ActionType: 1},
					{ID: "btn_info", Label: "机器人信息", Data: "btn_info", Style: 4, ActionType: 1},
				},
				// 第 3 行：3 个按钮（混合样式）
				{
					{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},
					{ID: "btn_ping", Label: "Ping", Data: "btn_ping", Style: 2, ActionType: 1},
					{ID: "btn_info", Label: "信息", Data: "btn_info", Style: 3, ActionType: 1},
				},
				// 第 4 行：跳转按钮 (使用 URL 字段替代 ActionType: 0，避免 Go 零值歧义)
				{
					{ID: "btn_qun", Label: "加群", Style: 1,
						URL: "https://qm.qq.com/q/9Rvq6VylQA"},
					{ID: "btn_aaaaa", Label: "加入频道", Style: 0,
						URL: "https://pd.qq.com/s/7zeumh7of?b=9"},
				},
			}
			// 存储原始命令 msg_id 供按钮回调被动回复使用
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtonRows("多行按钮布局演示：", rows)
		},
	})

	// ── 命令：/buttonaction 演示不同 action type ──
	// ActionType: 1=回调, 2=指令 (跳转按钮请使用 URL 字段)
	// Permission: 0=指定用户, 1=管理员, 2=所有人(默认), 3=指定身份组
	// 注意: Enter (auto-send) 和 Anchor (选图器) 仅 C2C 单聊可用，群聊/频道不可用
	r.Register(contract.Command{
		Name:        "buttonaction",
		Description: "演示按钮的 action type（跳转/回调/指令）和权限控制",
		Usage:       "buttonaction",
		Handler: func(ctx contract.CommandContext) error {
			// Enter 仅 C2C 有效，群聊/频道中使用会导致"无权限"
			isC2C := ctx.Scene() == contract.SceneC2C
			rows := [][]contract.MessageButton{
				// 第 1 行：跳转按钮 (使用 URL 字段)
				{
					{
						ID: "btn_github", Label: "GitHub", Style: 1,
						URL: "https://github.com/Luoyangan/LQBOT",
						UnsupportTips: "请使用最新版手机QQ",
					},
				},
				// 第 2 行：指令按钮 (ActionType=2)
				// Enter=true 仅 C2C，群聊/频道中强制为 false
				{
					{
						ID: "btn_cmd_ping", Label: "发 Ping", Style: 3,
						Data: "/ping", ActionType: 2,
						Enter: isC2C, UnsupportTips: "请升级客户端",
					},
					{
						ID: "btn_cmd_info", Label: "发 info", Style: 4,
						Data: "/info", ActionType: 2,
						UnsupportTips: "请升级客户端",
					},
				},
				// 第 3 行：回调按钮 + 指令按钮(带引用回复)
				{
					{
						ID: "btn_hello3", Label: "你好(回调)", Style: 1,
						Data: "btn_hello", ActionType: 1,
					},
					{
						ID: "btn_cmd_reply", Label: "发 Ping(引用)", Style: 0,
						Data: "/ping", ActionType: 2, Reply: true,
						UnsupportTips: "请升级客户端",
					},
				},
				{
					{
						ID: "btn_hello3a", Label: "管理权限", Style: 0,
						Data: "你好", ActionType: 2,
						Enter: isC2C, Permission: 1,
					},
				},
			}
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtonRows(
				"按钮 Action 类型演示：[跳转] [指令] [回调]",
				rows,
			)
		},
	})

	// ── 命令：/at 演示 @ 提及用户 ──
	// <qqbot-at-user> 仅在 Markdown 消息中生效。
	// 使用文本链新格式 <qqbot-at-user id="xxx" />（旧格式 <@userid> 即将弃用）
	// 详见: https://bot.q.qq.com/wiki/develop/api-v2/server-inter/message/trans/text-chain.html
	r.Register(contract.Command{
		Name:        "at",
		Description: "艾特(提及)消息发送者演示",
		Usage:       "at <消息>",
		Handler: func(ctx contract.CommandContext) error {
			content := "你被提到了！"
			if ctx.ArgCount() > 0 {
				content = strings.Join(ctx.Args(), " ")
			}
			msg := contract.MentionUser(ctx.AuthorID()) + " " + content
			return ctx.ReplyMarkdown(msg)
		},
	})

	// ── 命令：/textchain 演示文本链交互元素 ──
	// 文档: https://bot.q.qq.com/wiki/develop/api-v2/server-inter/message/trans/text-chain.html
	//   CmdInput（参数指令）→ 所有场景支持
	//   CmdEnter（回车指令）→ 仅 C2C（群聊/文字子频道不支持）
	//   @某人               → 群聊/文字子频道可用
	//   ChannelLink/Emoji  → 仅频道可用
	r.Register(contract.Command{
		Name:        "textchain",
		Description: "演示文本链交互元素（Markdown）",
		Usage:       "textchain",
		Handler: func(ctx contract.CommandContext) error {
			switch ctx.Scene() {
			case contract.SceneGuild:
				msg := "# 文本链演示（频道）\n" +
					"1. @某人: " + contract.MentionUser(ctx.AuthorID()) + "\n" +
					"2. 参数指令: " + contract.CmdInput("/ping", "点我发 ping", false) + "\n" +
					"3. 跳转子频道: " + contract.ChannelLink(ctx.ChannelID()) + "\n" +
					"4. 表情: " + contract.Emoji("4") + " " + contract.Emoji("14") + "\n\n" +
					"回车指令仅 C2C，请用 /md interactive c2c"
				return ctx.ReplyMarkdown(msg)

			case contract.SceneGroup:
				msg := "# 文本链演示（群聊）\n" +
					"1. @某人: " + contract.MentionUser(ctx.AuthorID()) + "\n" +
					"2. 参数指令: " + contract.CmdInput("/ping", "点我发 ping", false)
				return ctx.ReplyMarkdown(msg)

			default: // C2C
				return ctx.Reply("C2C 场景请用: /md interactive c2c <用户ID>")
			}
		},
	})
}
