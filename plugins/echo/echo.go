// Package echo implements the /echo command that repeats user input.
package echo

import (
	"fmt"
	"strings"

	"github.com/Luoyangan/LQBOT/internal/contract"
)

const maxEchoLength = 200

// Register adds the echo command to the given registrar.
// Demonstrates: argument parsing, input validation, error feedback.
func Register(r contract.CommandRegister) {
	r.Register(contract.Command{
		Name:        "echo",
		Description: "重复你输入的消息",
		Usage:       "echo <消息内容>",
		Handler: func(ctx contract.CommandContext) error {
			// Validate: no arguments provided
			if ctx.ArgCount() == 0 {
				return ctx.Reply("请提供要重复的消息。用法: /echo <消息内容>")
			}

			msg := strings.Join(ctx.Args(), " ")

			// Validate: message too long
			if len(msg) > maxEchoLength {
				return ctx.Reply(fmt.Sprintf("消息过长，请控制在%d 字以内（当前 %d 字）",
					maxEchoLength, len(msg)))
			}

			return ctx.Reply(msg)
		},
	})
}
