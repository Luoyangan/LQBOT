package menu

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

// CommandsFunc returns all registered commands.
type CommandsFunc func() []contract.Command

func Register(r contract.CommandRegister, cmdsFn ...CommandsFunc) {
	r.Register(contract.Command{
		Name:        "菜单",
		Aliases:     []string{"menu", "help", "帮助", "命令"},
		Description: "显示帮助菜单",
		Usage:       "menu [指令名]",
		Handler: func(ctx contract.CommandContext) error {
			// If a command name is provided, show detailed help for that command
			if ctx.ArgCount() > 0 && len(cmdsFn) > 0 {
				cmdName := ctx.Arg(0)
				for _, cmd := range cmdsFn[0]() {
					if cmd.Name == cmdName {
						return showCommandDetail(ctx, &cmd)
					}
					for _, alias := range cmd.Aliases {
						if alias == cmdName {
							return showCommandDetail(ctx, &cmd)
						}
					}
				}
				return ctx.Reply("未找到指令: " + cmdName)
			}

			// Show all commands
			if len(cmdsFn) > 0 {
				return showAllCommands(ctx, cmdsFn[0])
			}
			return showDefaultMenu(ctx)
		},
	})
}

func showAllCommands(ctx contract.CommandContext, cmdsFn CommandsFunc) error {
	cmds := cmdsFn()
	if len(cmds) == 0 {
		return showDefaultMenu(ctx)
	}

	// Sort by name
	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name < cmds[j].Name
	})

	var b strings.Builder
	b.WriteString("📋 LQBOT 指令列表\n")
	b.WriteString("━━━━━━━━━━━━━━\n")

	for _, cmd := range cmds {
		if cmd.Description == "" {
			continue
		}
		usage := cmd.Usage
		if usage == "" {
			usage = cmd.Name
		}

		// Show aliases if any
		aliases := ""
		if len(cmd.Aliases) > 0 {
			aliases = " [" + strings.Join(cmd.Aliases, "/") + "]"
		}

		b.WriteString(fmt.Sprintf("/%s%s — %s\n", usage, aliases, cmd.Description))
	}

	b.WriteString("━━━━━━━━━━━━━━\n")
	b.WriteString("发送 /menu <指令名> 查看详情")

	return ctx.Reply(b.String())
}

func showCommandDetail(ctx contract.CommandContext, cmd *contract.Command) error {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📖 %s\n", cmd.Name))
	b.WriteString("━━━━━━━━━━━━━━\n")
	if cmd.Description != "" {
		b.WriteString(fmt.Sprintf("说明: %s\n", cmd.Description))
	}
	b.WriteString(fmt.Sprintf("用法: /%s\n", cmd.Usage))
	if len(cmd.Aliases) > 0 {
		b.WriteString("别称: " + strings.Join(cmd.Aliases, ", ") + "\n")
	}
	return ctx.Reply(b.String())
}

func showDefaultMenu(ctx contract.CommandContext) error {
	return ctx.Reply("LQBOT 帮助菜单\n" +
		"示例指令：\n" +
		"echo <消息> | 重复用户发送的消息\n" +
		"button | 发送一个按钮交互消息\n" +
		"你好 | 与 LQBOT 交互，显示欢迎消息\n" +
		"button2 | 演示不同样式和行为的多个按钮\n" +
		"buttons | 演示多行按钮布局\n" +
		"buttonaction | 演示按钮的 action type（跳转/回调/指令）和权限控制\n" +
		"at | 演示 @ 提醒用户\n" +
		"textchain | 演示文本链交互元素（Markdown）\n" +
		"menu | 显示帮助菜单\n" +
		"embed | 发送一张富卡片消息（仅频道）\n" +
		"ping | 测试机器人是否在线\n" +
		"image | 发送一张图片消息\n" +
		"video | 发送一个视频消息\n" +
		"server | 查看当前频道/群聊信息\n" +
		"whoami | 查看你在当前频道的信息\n" +
		"delete <消息ID> | 撤回一条消息\n" +
		"react <表情ID> | 给消息添加表情反应（仅频道）\n" +
		"pin <消息ID> | 将消息设为精华（仅频道）\n" +
		"unpin <消息ID> | 取消精华消息（仅频道）\n" +
		"md | 发送 Markdown 消息（自定义/模板/交互）\n" +
		"mda | 发送 Markdown 代码消息\n" +
		"buttonstate | 演示按钮 3 种状态：Normal / Press / Loading\n" +
		"deletes <消息ID> | 撤回消息（仅群主/管理员可用）",
	)
}
