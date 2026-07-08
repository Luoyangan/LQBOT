// Package blacklist provides runtime blacklist management commands.
package blacklist

import (
	"fmt"

	"github.com/Luoyangan/LQBOT/internal/contract"
	fwblacklist "github.com/Luoyangan/LQBOT/internal/blacklist"
)

func Register(r contract.CommandRegister, mgr *fwblacklist.Manager) {
	// /blacklist — list all blacklisted users/groups
	r.Register(contract.Command{
		Name:        "blacklist",
		Aliases:     []string{"bl"},
		Description: "管理黑名单（用户/群）",
		Usage:       "blacklist [add|remove|list] [user|group] [id]",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.ArgCount() == 0 || ctx.Arg(0) == "list" {
				return ctx.Reply(mgr.StringList())
			}

			action := ctx.Arg(0)
			switch action {
			case "add":
				return handleAdd(mgr, ctx)
			case "remove":
				return handleRemove(mgr, ctx)
			default:
				return ctx.Reply("用法: blacklist [add|remove|list] [user|group] [id]")
			}
		},
	})
}

func handleAdd(mgr *fwblacklist.Manager, ctx contract.CommandContext) error {
	if ctx.ArgCount() < 3 {
		return ctx.Reply("用法: blacklist add [user|group] [id]")
	}

	target := ctx.Arg(1)
	id := ctx.Arg(2)

	switch target {
	case "user":
		if err := mgr.AddUser(id); err != nil {
			return ctx.Reply("添加用户黑名单失败: " + err.Error())
		}
		return ctx.Reply(fmt.Sprintf("用户 %s 已加入黑名单", id))
	case "group":
		if err := mgr.AddGroup(id); err != nil {
			return ctx.Reply("添加群黑名单失败: " + err.Error())
		}
		return ctx.Reply(fmt.Sprintf("群 %s 已加入黑名单", id))
	default:
		return ctx.Reply("类型错误，请使用 user 或 group")
	}
}

func handleRemove(mgr *fwblacklist.Manager, ctx contract.CommandContext) error {
	if ctx.ArgCount() < 3 {
		return ctx.Reply("用法: blacklist remove [user|group] [id]")
	}

	target := ctx.Arg(1)
	id := ctx.Arg(2)

	switch target {
	case "user":
		if err := mgr.RemoveUser(id); err != nil {
			return ctx.Reply("移除用户黑名单失败: " + err.Error())
		}
		return ctx.Reply(fmt.Sprintf("用户 %s 已移出黑名单", id))
	case "group":
		if err := mgr.RemoveGroup(id); err != nil {
			return ctx.Reply("移除群黑名单失败: " + err.Error())
		}
		return ctx.Reply(fmt.Sprintf("群 %s 已移出黑名单", id))
	default:
		return ctx.Reply("类型错误，请使用 user 或 group")
	}
}
