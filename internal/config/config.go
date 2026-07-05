// Package config handles configuration loading and management.
package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Luoyangan/LQBOT/internal/types"
	"gopkg.in/yaml.v3"
)

// Init ensures the config file exists, creating it from the example template if missing.
// Returns the path to use for loading.
func Init(path string) string {
	if fileExists(path) {
		return path
	}

	// Try the example template
	examplePath := examplePathFrom(path)
	if examplePath != "" && fileExists(examplePath) {
		log.Printf("[config] %s not found, copying from %s", path, examplePath)
		if err := copyFile(examplePath, path); err != nil {
			log.Printf("[config] failed to copy config: %v", err)
		} else {
			return path
		}
	}

	// No config at all — create a minimal one
	log.Printf("[config] no config files found, creating minimal %s", path)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err == nil {
		minimal := `# LQBOT 配置文件
app_id: "your_app_id_here"       # QQ 开放平台申请的 AppID
app_secret: "your_app_secret_here" # QQ 开放平台申请的 AppSecret
sandbox: false                    # true = 沙箱模式, false = 生产模式
intents:
  - AT_MESSAGES                  # @机器人消息
  #- GUILD_MESSAGES               # 频道全部消息
  - GROUP_AND_C2C_EVENT          # 群聊和私聊
  - INTERACTION                  # 按钮/选择框交互事件
access_type: websocket           # websocket | webhook
log_level: info                  # 控制台日志级别: trace/debug/info/warn/error
log_level_db: info               # 数据库日志级别: trace/debug/info/warn/error（空=不写入DB）
log_db_exclude:                  # 数据库日志排除关键词（消息包含任一关键词则不写入DB）
  - "Heartbeat"                  # 过滤心跳日志，保留事件消息

webhook:                         # webhook 模式配置（access_type: webhook 时生效）
  port: 8080                     # HTTP 监听端口
  path: /webhook                 # 回调路径

storage:                         # 数据库配置
  driver: sqlite                 # 数据库驱动，支持 sqlite
  dsn: "data/lqbot.db"           # 数据库文件路径
  log_cleanup:                   # 日志清理配置
    enabled: true                # 是否启用日志清理
    interval: "24h"              # 清理周期（time.Duration 格式）
    retain_days: 7              # 保留多少天的日志
`
		_ = os.WriteFile(path, []byte(minimal), 0644)
	}

	return path
}

// Load reads and parses a YAML configuration file.
// Returns a validated Config struct. Missing optional fields get defaults.
func Load(path string) (*types.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &types.Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	applyDefaults(cfg)
	warnPlaceholders(cfg)

	return cfg, nil
}

// applyDefaults fills in default values for optional fields.
func applyDefaults(cfg *types.Config) {
	if cfg.AccessType == "" {
		cfg.AccessType = types.AccessWebSocket
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = types.LogLevelInfo
	}
	if cfg.Storage.Driver == "" {
		cfg.Storage.Driver = types.StorageSQLite
	}
	if cfg.Storage.DSN == "" {
		cfg.Storage.DSN = "data/lqbot.db"
	}
	if len(cfg.Intents) == 0 {
		cfg.Intents = []string{"GUILD_MESSAGES"}
	}
	// Log cleanup defaults
	if cfg.Storage.LogCleanup.Interval == "" {
		cfg.Storage.LogCleanup.Interval = "24h"
	}
	if cfg.Storage.LogCleanup.RetainDays == 0 {
		cfg.Storage.LogCleanup.RetainDays = 7
	}
}

// warnPlaceholders logs warnings for placeholder/empty values but doesn't block startup.
func warnPlaceholders(cfg *types.Config) {
	if cfg.AppID == "" || cfg.AppID == "your_app_id_here" {
		log.Println("[config] WARNING: app_id is not set. Edit configs/config.yaml to connect to QQ.")
	}
	if cfg.AppSecret == "" || cfg.AppSecret == "your_app_secret_here" {
		log.Println("[config] WARNING: app_secret is not set. Edit configs/config.yaml to connect to QQ.")
	}
}

// --- helpers ---

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func examplePathFrom(configPath string) string {
	dir := filepath.Dir(configPath)
	base := filepath.Base(configPath)

	candidates := []string{
		filepath.Join(dir, "config.example.yaml"),
		filepath.Join(dir, "config.sample.yaml"),
		filepath.Join(dir, "example.yaml"),
	}

	// Try inserting .example before extension
	ext := filepath.Ext(base)
	if ext != "" {
		name := base[:len(base)-len(ext)]
		candidates = append([]string{filepath.Join(dir, name+".example"+ext)}, candidates...)
	}

	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
