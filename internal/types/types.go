// Package types defines public types, constants, and enumerations for LQBOT.
// Event types and intents align with QQ Official API v2.
package types

import "github.com/tencent-connect/botgo/dto"

// Event type constants from QQ Official API v2.
// These map to the "t" field in WebSocket WSPayload and "t" field in webhook payloads.
const (
	// Guild channel messages (need GUILD_MESSAGES intent)
	EventMessageCreate = string(dto.EventMessageCreate) // "MESSAGE_CREATE"

	// @bot messages in public guild channels (need GUILD_AT_MESSAGE intent)
	EventAtMessageCreate = string(dto.EventAtMessageCreate) // "AT_MESSAGE_CREATE"

	// @bot messages in group chats (need GROUP_AND_C2C_EVENT intent)
	EventGroupAtMessageCreate = string(dto.EventGroupAtMessageCreate) // "GROUP_AT_MESSAGE_CREATE"

	// Direct messages / C2C (need GROUP_AND_C2C_EVENT intent)
	EventC2CMessageCreate = string(dto.EventC2CMessageCreate) // "C2C_MESSAGE_CREATE"

	// Message deleted
	EventMessageDelete = string(dto.EventMessageDelete) // "MESSAGE_DELETE"

	// Guild events
	EventGuildCreate = string(dto.EventGuildCreate) // "GUILD_CREATE"
	EventGuildDelete = string(dto.EventGuildDelete) // "GUILD_DELETE"

	// Member events
	EventMemberJoin  = string(dto.EventGuildMemberAdd)    // "GUILD_MEMBER_ADD"
	EventMemberLeave = string(dto.EventGuildMemberRemove) // "GUILD_MEMBER_REMOVE"

	// Interaction events (button clicks, select menus)
	EventInteractionCreate = string(dto.EventInteractionCreate) // "INTERACTION_CREATE"

	// Group messages (new QQ API v2 event, no @ required)
	EventGroupMessageCreate = "GROUP_MESSAGE_CREATE"
)

// IntentList maps human-readable intent names to botgo's dto.Intent bitmask values.
// These correspond to the intents field in config.yaml.
var IntentList = map[string]dto.Intent{
	"GUILDS":              dto.IntentGuilds,
	"GUILD_MEMBERS":       dto.IntentGuildMembers,
	"GUILD_MESSAGES":      dto.IntentGuildMessages,
	"GUILD_AT_MESSAGE":    dto.IntentGuildAtMessage,
	"AT_MESSAGES":         dto.IntentGuildAtMessage, // alias for GUILD_AT_MESSAGE
	"GROUP_AND_C2C_EVENT": dto.IntentGroupMessages,
	"DIRECT_MESSAGE":      dto.IntentDirectMessages,
	"INTERACTION":         dto.IntentInteraction,
	"MESSAGE_AUDIT":       dto.IntentAudit,
	"FORUM":               dto.IntentForum,
	"AUDIO":               dto.IntentAudio,
}

// IntentsToBitmask converts a list of human-readable intent names to a dto.Intent bitmask.
// Unknown intent names are silently ignored (logged at debug level).
func IntentsToBitmask(intents []string) dto.Intent {
	var mask dto.Intent
	for _, name := range intents {
		if bit, ok := IntentList[name]; ok {
			mask |= bit
		}
	}
	return mask
}

// AccessType defines how the bot connects to QQ.
type AccessType string

const (
	AccessWebSocket AccessType = "websocket"
	AccessWebhook   AccessType = "webhook"
)

// LogLevel defines logging verbosity.
type LogLevel string

const (
	LogLevelTrace LogLevel = "trace"
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// StorageDriver defines the backend storage type.
type StorageDriver string

const (
	StorageSQLite StorageDriver = "sqlite"
	StorageMySQL  StorageDriver = "mysql"
	StorageRedis  StorageDriver = "redis"
)

// Config represents the top-level application configuration.
type Config struct {
	AppID      string        `yaml:"app_id"`
	AppSecret  string        `yaml:"app_secret"`
	Sandbox    bool          `yaml:"sandbox"`     // true = sandbox mode, false = production mode
	Intents    []string      `yaml:"intents"`     // Human-readable names, e.g. ["GUILD_MESSAGES", "GROUP_AND_C2C_EVENT"]
	AccessType AccessType    `yaml:"access_type"`
	LogLevel   LogLevel      `yaml:"log_level"`
	LogNoColor bool          `yaml:"log_no_color"` // Force disable ANSI color in log output
	Webhook    WebhookConfig `yaml:"webhook"`
	Storage    StorageConfig `yaml:"storage"`
}

// WebhookConfig represents webhook adapter configuration.
type WebhookConfig struct {
	Port int    `yaml:"port"`
	Path string `yaml:"path"`
}

// StorageConfig represents storage backend configuration.
type StorageConfig struct {
	Driver StorageDriver `yaml:"driver"`
	DSN    string        `yaml:"dsn"`
}
