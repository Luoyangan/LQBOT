package script

import (
	"errors"
	"fmt"
	"time"

	"github.com/dop251/goja"
)

// runJS JavaScript 沙箱（goja）：
//   - 无标准库注入（无 fs/net/process），仅纯语言能力；
//   - 通过 Interrupt 强制中断超时脚本（基于语句粒度）。
func runJS(code string, env map[string]any, timeout time.Duration) (map[string]any, error) {
	vm := goja.New()

	envObj := vm.NewObject()
	for k, v := range env {
		envObj.Set(k, v)
	}
	vm.Set("env", envObj)

	timer := time.AfterFunc(timeout, func() {
		vm.Interrupt(fmt.Errorf("script: js 执行超时（%s）", timeout))
	})
	defer timer.Stop()

	if _, err := vm.RunString(code); err != nil {
		var interrupted *goja.InterruptedError
		if errors.As(err, &interrupted) {
			return nil, errors.New(interrupted.Error())
		}
		return nil, fmt.Errorf("script: js 执行失败: %w", err)
	}

	out := make(map[string]any, len(env)+4)
	for k, v := range env {
		out[k] = v
	}
	// 回读 env（脚本可能整体重赋值 env = {...}，从全局取最新引用）
	if o, ok := vm.Get("env").(*goja.Object); ok {
		for _, k := range o.Keys() {
			out[k] = o.Get(k).Export()
		}
	}
	return out, nil
}
