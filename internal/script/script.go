// Package script 双语言脚本沙箱（Lua + JavaScript）：
// 仅开放基础库（禁 os/io/debug/package/网络），强制执行超时中断；
// env 作为脚本可读写的全局表，执行完毕后回读作为结果。
package script

import (
	"fmt"
	"time"
)

// 超时限制。
const (
	DefaultTimeout = 100 * time.Millisecond
	MaxTimeout     = 5 * time.Second
)

// Run 在沙箱中执行脚本：
//   - language：lua / js
//   - env：注入脚本的全局 env 表（脚本可读可写）
//
// 返回执行后的 env 快照；脚本执行错误（含语法错误与超时）返回 error。
func Run(language, code string, env map[string]any, timeout time.Duration) (map[string]any, error) {
	if code == "" {
		return nil, fmt.Errorf("script: 脚本内容为空")
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if timeout > MaxTimeout {
		timeout = MaxTimeout
	}
	switch language {
	case "lua":
		return runLua(code, env, timeout)
	case "js":
		return runJS(code, env, timeout)
	default:
		return nil, fmt.Errorf("script: 未知语言 %q（支持 lua/js）", language)
	}
}

// normalizeOut 保证返回非 nil map（脚本未改动 env 时返回输入快照）。
func normalizeOut(env map[string]any) map[string]any {
	out := make(map[string]any, len(env)+4)
	for k, v := range env {
		out[k] = v
	}
	return out
}
