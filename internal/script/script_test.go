package script

import (
	"strings"
	"testing"
	"time"
)

func TestLuaBasic(t *testing.T) {
	env := map[string]any{"name": "LQBOT", "n": 2}
	out, err := Run("lua", `
		env.greeting = "hello " .. env.name
		env.doubled = env.n * 2
		env.list = {"a", "b"}
	`, env, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out["greeting"] != "hello LQBOT" {
		t.Fatalf("greeting = %#v", out["greeting"])
	}
	if out["doubled"] != float64(4) {
		t.Fatalf("doubled = %#v", out["doubled"])
	}
	if arr, ok := out["list"].([]any); !ok || len(arr) != 2 || arr[0] != "a" {
		t.Fatalf("list = %#v", out["list"])
	}
}

func TestLuaSandboxBlocked(t *testing.T) {
	// os / io / debug / package / require 均不可用
	for _, code := range []string{
		`os.exit(1)`,
		`io.open("x")`,
		`debug.sethook()`,
		`require("os")`,
		`package.path = "x"`,
		`dofile("x")`,
		`loadfile("x")`,
	} {
		if _, err := Run("lua", code, nil, 0); err == nil {
			t.Fatalf("危险调用未拦截: %s", code)
		}
	}
}

func TestLuaTimeout(t *testing.T) {
	_, err := Run("lua", `while true do end`, nil, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("应超时中断: %v", err)
	}
}

func TestJSPBasic(t *testing.T) {
	env := map[string]any{"name": "LQBOT", "n": 2}
	out, err := Run("js", `
		env.greeting = "hello " + env.name
		env.doubled = env.n * 2
		env.list = ["a", "b"]
		env.nested = { price: 12 }
	`, env, 0)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out["greeting"] != "hello LQBOT" {
		t.Fatalf("greeting = %#v", out["greeting"])
	}
	if out["doubled"] != int64(4) {
		t.Fatalf("doubled = %#v", out["doubled"])
	}
	if arr, ok := out["list"].([]any); !ok || len(arr) != 2 || arr[0] != "a" {
		t.Fatalf("list = %#v", out["list"])
	}
	if m, ok := out["nested"].(map[string]any); !ok || m["price"] != int64(12) {
		t.Fatalf("nested = %#v", out["nested"])
	}
}

func TestJSSandboxBlocked(t *testing.T) {
	// 无标准库注入：fs/net/process 均不存在
	for _, code := range []string{
		`require("fs")`,
		`process.exit(1)`,
		`fetch("http://x")`,
		`Java`,
	} {
		if _, err := Run("js", code, nil, 0); err == nil {
			t.Fatalf("危险调用未拦截: %s", code)
		}
	}
}

func TestJSTimeout(t *testing.T) {
	_, err := Run("js", `for(;;){}`, nil, 50*time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("应超时中断: %v", err)
	}
}

func TestRunErrors(t *testing.T) {
	if _, err := Run("lua", "", nil, 0); err == nil {
		t.Fatal("空脚本应报错")
	}
	if _, err := Run("python", "print(1)", nil, 0); err == nil {
		t.Fatal("未知语言应报错")
	}
	if _, err := Run("lua", `syntax error ???`, nil, 0); err == nil {
		t.Fatal("语法错误应报错")
	}
	if _, err := Run("js", `syntax ???`, nil, 0); err == nil {
		t.Fatal("语法错误应报错")
	}
	// 超时上限
	if _, err := Run("lua", `while true do end`, nil, time.Hour); err == nil {
		t.Fatal("超上限应被钳制为 MaxTimeout 并超时")
	}
}
