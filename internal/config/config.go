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
app_id: "your_app_id_here"
app_secret: "your_app_secret_here"
sandbox: false
intents:
  - GUILD_MESSAGES
  - GROUP_AND_C2C_EVENT
access_type: websocket
log_level: info

webhook:
  port: 9000
  path: /webhook

storage:
  driver: sqlite
  dsn: "data/lqbot.db"
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
