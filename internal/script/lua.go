package script

import (
	"context"
	"fmt"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// runLua Lua 沙箱：
//   - 仅开放 base/string/table/math 库，禁 os/io/debug/package/coroutine；
//   - 移除 base 库中的 dofile/loadfile/print/load/loadstring；
//   - 通过 SetContext 逐指令超时中断（官方支持机制）。
func runLua(code string, env map[string]any, timeout time.Duration) (map[string]any, error) {
	L := lua.NewState(lua.Options{SkipOpenLibs: true})
	defer L.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	L.SetContext(ctx)

	for _, open := range []func(*lua.LState) int{
		lua.OpenBase, lua.OpenString, lua.OpenTable, lua.OpenMath,
	} {
		open(L)
	}
	// base 库残留的危险/干扰函数显式移除
	for _, name := range []string{"dofile", "loadfile", "print", "load", "loadstring", "collectgarbage"} {
		L.SetGlobal(name, lua.LNil)
	}

	// 注入 env 表（脚本可读写）
	envTbl := L.NewTable()
	for k, v := range env {
		L.SetField(envTbl, k, toLua(L, v))
	}
	L.SetGlobal("env", envTbl)

	if err := L.DoString(code); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("script: lua 执行超时（%s）", timeout)
		}
		return nil, fmt.Errorf("script: lua 执行失败: %w", err)
	}
	out := make(map[string]any, len(env)+4)
	for k, v := range env {
		out[k] = v
	}
	envTbl.ForEach(func(k, v lua.LValue) {
		out[k.String()] = luaToGo(v)
	})
	return out, nil
}

// toLua Go 值 → Lua 值（仅支持沙箱所需的标量与容器）。
func toLua(L *lua.LState, v any) lua.LValue {
	switch x := v.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(x)
	case string:
		return lua.LString(x)
	case float64:
		return lua.LNumber(x)
	case int:
		return lua.LNumber(x)
	case int64:
		return lua.LNumber(x)
	case []any:
		tbl := L.NewTable()
		for i, e := range x {
			tbl.RawSetInt(i+1, toLua(L, e))
		}
		return tbl
	case map[string]any:
		tbl := L.NewTable()
		for k, e := range x {
			tbl.RawSetString(k, toLua(L, e))
		}
		return tbl
	default:
		return lua.LString(fmt.Sprintf("%v", x))
	}
}

func luaToGo(v lua.LValue) any {
	switch x := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(x)
	case lua.LString:
		return string(x)
	case lua.LNumber:
		return float64(x)
	case *lua.LTable:
		// 连续数组表 → []any，其余按 map 返回
		if isArrayTable(x) {
			arr := make([]any, 0, x.Len())
			for i := 1; i <= x.Len(); i++ {
				arr = append(arr, luaToGo(x.RawGetInt(i)))
			}
			return arr
		}
		m := make(map[string]any)
		x.ForEach(func(k, v lua.LValue) {
			m[k.String()] = luaToGo(v)
		})
		return m
	default:
		return x.String()
	}
}

// isArrayTable 判断是否为连续数组表（键为 1..n 且无字符串键）。
func isArrayTable(t *lua.LTable) bool {
	n := t.Len()
	if n == 0 {
		return false
	}
	isArray := true
	for i := 1; i <= n; i++ {
		if t.RawGetInt(i) == lua.LNil {
			return false
		}
	}
	t.ForEach(func(k, _ lua.LValue) {
		if _, ok := k.(lua.LString); ok {
			isArray = false
		}
	})
	return isArray
}
