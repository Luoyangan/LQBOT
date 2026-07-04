// Package adapter implements protocol adapters for QQ API connections.
// It provides WebSocket and Webhook adapters that implement contract.Adapter.
package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Luoyangan/LQBOT/internal/log"
	"github.com/Luoyangan/LQBOT/internal/types"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
)

// WebSocketAdapter implements contract.Adapter using QQ WebSocket (via botgo).
type WebSocketAdapter struct {
	name      string
	appID     string
	appSecret string
	sandbox   bool     // true = use sandbox API endpoints
	intents   []string // only register these intents; empty = all registered
	events    chan []byte
	logger    *log.Logger
	closeOnce sync.Once
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	botUserID string // Bot's own user ID, set from Ready event
	mu        sync.RWMutex
}

// NewWebSocketAdapter creates a new WebSocket adapter.
// If sandbox is true, the adapter connects to QQ's sandbox environment.
// intents is the list of configured intents; if empty, all registered handlers are used.
func NewWebSocketAdapter(appID, appSecret string, sandbox bool, intents []string, logger *log.Logger) (*WebSocketAdapter, error) {
	return &WebSocketAdapter{
		name:      "websocket",
		appID:     appID,
		appSecret: appSecret,
		sandbox:   sandbox,
		intents:   intents,
		events:    make(chan []byte, 512),
		logger:    logger,
	}, nil
}

// Name returns the adapter name.
func (a *WebSocketAdapter) Name() string { return a.name }

// Start establishes the WebSocket connection and begins receiving events.
func (a *WebSocketAdapter) Start(ctx context.Context) error {
	// Skip actual connection if credentials are placeholder values
	if a.appID == "" || a.appID == "your_app_id_here" {
		a.logger.Warn("WebSocket adapter: app_id not configured, skipping connection")
		return nil
	}

	// Redirect botgo's internal logs to our zerolog logger
	botgo.SetLogger(&webhookLogger{inner: a.logger})

	ctx, a.cancel = context.WithCancel(ctx)

	// 1. Create OAuth2 credentials
	credentials := &token.QQBotCredentials{
		AppID:     a.appID,
		AppSecret: a.appSecret,
	}
	tokenSource := token.NewQQBotTokenSource(credentials)

	// 3. Start background token refresh
	if err := token.StartRefreshAccessToken(ctx, tokenSource); err != nil {
		return fmt.Errorf("start token refresh: %w", err)
	}

	// 4. Create OpenAPI client (used to get WebSocket endpoint)
	var api openapi.OpenAPI
	if a.sandbox {
		api = botgo.NewSandboxOpenAPI(a.appID, tokenSource)
	} else {
		api = botgo.NewOpenAPI(a.appID, tokenSource)
	}

	// 5. Get WebSocket access point (gateway)
	wsAP, err := api.WS(ctx, nil, "")
	if err != nil {
		return fmt.Errorf("get websocket gateway: %w", err)
	}
	a.logger.Info("WebSocket gateway obtained", "url", wsAP.URL, "shards", wsAP.Shards)

	// 6. Register event handlers 鈥?emit events with native QQ API event type strings
	intent := event.RegisterHandlers(
		// Ready 鈥?connection established
		event.ReadyHandler(func(event *dto.WSPayload, data *dto.WSReadyData) {
			a.mu.Lock()
			a.botUserID = data.User.ID
			a.mu.Unlock()
			a.logger.Info("WebSocket ready",
				"version", data.Version,
				"session_id", data.SessionID,
				"user", data.User.ID,
				"shard", fmt.Sprintf("%d/%d", data.Shard[0], data.Shard[1]),
			)
		}),
		// Error notify
		event.ErrorNotifyHandler(func(err error) {
			a.logger.Error("WebSocket error", "error", err)
		}),
		// Guild channel messages (MESSAGE_CREATE)
		event.MessageEventHandler(func(event *dto.WSPayload, data *dto.WSMessageData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// @bot messages in guild channels (AT_MESSAGE_CREATE)
		event.ATMessageEventHandler(func(event *dto.WSPayload, data *dto.WSATMessageData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// @bot messages in group chats (GROUP_AT_MESSAGE_CREATE)
		event.GroupATMessageEventHandler(func(event *dto.WSPayload, data *dto.WSGroupATMessageData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// C2C direct messages (C2C_MESSAGE_CREATE)
		event.C2CMessageEventHandler(func(event *dto.WSPayload, data *dto.WSC2CMessageData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// Guild events
		event.GuildEventHandler(func(event *dto.WSPayload, data *dto.WSGuildData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// Member events
		event.GuildMemberEventHandler(func(event *dto.WSPayload, data *dto.WSGuildMemberData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// Interaction events (button/select menu callbacks)
		event.InteractionEventHandler(func(event *dto.WSPayload, data *dto.WSInteractionData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// Message delete events
		event.MessageDeleteEventHandler(func(event *dto.WSPayload, data *dto.WSMessageDeleteData) error {
			return a.emitEvent(string(event.Type), data)
		}),
		// Catch-all for QQ API events not yet in botgo (e.g. GROUP_MESSAGE_CREATE)
		event.PlainEventHandler(func(wsEvent *dto.WSPayload, message []byte) error {
			if wsEvent.Type == dto.EventType("GROUP_MESSAGE_CREATE") {
				var raw dto.WSGroupATMessageData
				if err := event.ParseData(message, &raw); err != nil {
					return nil
				}
				return a.emitEvent(string(wsEvent.Type), &raw)
			}
			return nil
		}),
	)

	a.logger.Info("Connecting to QQ WebSocket...")

	// Determine which intents to subscribe to:
	//   - If configured intents are provided, use those (respecting public bot limits)
	//   - Otherwise fall back to the union of all registered handlers (original behaviour)
	sessionIntent := intent
	if len(a.intents) > 0 {
		sessionIntent = types.IntentsToBitmask(a.intents)
		a.logger.Info("using configured intents", "intents", a.intents, "bitmask", sessionIntent)
	}

	// 6. Start session manager (blocking — run in goroutine)
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		sessionManager := botgo.NewSessionManager()
		if err := sessionManager.Start(wsAP, tokenSource, &sessionIntent); err != nil {
			a.logger.Error("WebSocket session ended with error", "error", err)
		}
	}()

	return nil
}

// emitEvent serializes a data object with the native QQ API event type
// in the format matching WSPayload: {"t": "EVENT_TYPE", "d": <data>}
func (a *WebSocketAdapter) emitEvent(eventType string, data interface{}) error {
	raw, err := json.Marshal(map[string]interface{}{
		"t": eventType,
		"d": data,
	})
	if err != nil {
		a.logger.Error("failed to marshal event", "error", err)
		return nil
	}

	select {
	case a.events <- raw:
	default:
		a.logger.Warn("event channel full, dropping event", "type", eventType)
	}
	return nil
}

// Stop gracefully closes the WebSocket connection.
func (a *WebSocketAdapter) Stop(ctx context.Context) error {
	a.closeOnce.Do(func() {
		if a.cancel != nil {
			a.cancel()
		}
	})
	// Wait for the session goroutine to finish
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
	close(a.events)
	a.logger.Info("WebSocket adapter stopped")
	return nil
}

// Events returns the channel for consuming raw events.
func (a *WebSocketAdapter) Events() <-chan []byte {
	return a.events
}

// BotUserID returns the bot's own user ID from the QQ session.
func (a *WebSocketAdapter) BotUserID() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.botUserID
}

// ---
// WebhookAdapter
// ---

// WebhookAdapter implements contract.Adapter using HTTP Webhook.
type WebhookAdapter struct {
	name      string
	port      int
	path      string
	events    chan []byte
	logger    *log.Logger
	server    *http.Server
	closeOnce sync.Once
}

// NewWebhookAdapter creates a new Webhook adapter.
func NewWebhookAdapter(port int, path string, logger *log.Logger) *WebhookAdapter {
	if port == 0 {
		port = 9000
	}
	if path == "" {
		path = "/webhook"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return &WebhookAdapter{
		name:   "webhook",
		port:   port,
		path:   path,
		events: make(chan []byte, 256),
		logger: logger,
	}
}

func (a *WebhookAdapter) Name() string { return a.name }

// Start begins the HTTP server for receiving webhook events.
func (a *WebhookAdapter) Start(ctx context.Context) error {
	botgo.SetLogger(&webhookLogger{inner: a.logger})

	mux := http.NewServeMux()
	mux.HandleFunc(a.path, a.handleWebhook)

	a.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", a.port),
		Handler: mux,
	}

	// Start server in background
	go func() {
		a.logger.Info("Webhook adapter listening", "addr", a.server.Addr, "path", a.path)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.logger.Error("webhook server error", "error", err)
		}
	}()

	// Stop server when context is cancelled
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			a.logger.Error("webhook server shutdown error", "error", err)
		}
	}()

	return nil
}

// webhookPayload represents the native QQ Webhook POST body (matches WSPayload format).
type webhookPayload struct {
	Op int             `json:"op"`
	T  string          `json:"t"` // Native event type, e.g. "MESSAGE_CREATE"
	D  json.RawMessage `json:"d"`
	ID string          `json:"id"`
}

// handleWebhook processes incoming webhook POST requests from QQ.
func (a *WebhookAdapter) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse raw payload
	var p webhookPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		a.logger.Error("failed to parse webhook payload", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Pass through native event type; if unknown, skip
	if !isKnownEventType(p.T) {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Re-wrap in the same format as WebSocket adapter: {"t": "TYPE", "d": <data>}
	raw, err := json.Marshal(map[string]interface{}{
		"t": p.T,
		"d": p.D,
	})
	if err != nil {
		a.logger.Error("failed to marshal webhook event", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	select {
	case a.events <- raw:
	default:
		a.logger.Warn("webhook event channel full, dropping event", "type", p.T)
	}

	w.WriteHeader(http.StatusOK)
}

// isKnownEventType returns true if the event type is recognized.
func isKnownEventType(t string) bool {
	switch t {
	case "READY",
		"MESSAGE_CREATE", "AT_MESSAGE_CREATE",
		"GROUP_AT_MESSAGE_CREATE", "GROUP_MESSAGE_CREATE", "C2C_MESSAGE_CREATE",
		"MESSAGE_DELETE",
		"GUILD_CREATE", "GUILD_DELETE",
		"GUILD_MEMBER_ADD", "GUILD_MEMBER_REMOVE",
		"INTERACTION_CREATE",
		"DIRECT_MESSAGE_CREATE", "DIRECT_MESSAGE_DELETE",
		"MESSAGE_AUDIT_PASS", "MESSAGE_AUDIT_REJECT":
		return true
	}
	return false
}

func (a *WebhookAdapter) Stop(ctx context.Context) error {
	a.closeOnce.Do(func() {
		if a.server != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := a.server.Shutdown(shutdownCtx); err != nil {
				a.logger.Error("webhook server shutdown error", "error", err)
			}
		}
	})
	close(a.events)
	a.logger.Info("Webhook adapter stopped")
	return nil
}

func (a *WebhookAdapter) Events() <-chan []byte {
	return a.events
}

// BotUserID returns the bot's own user ID (not available via webhook).
func (a *WebhookAdapter) BotUserID() string { return "" }

// webhookLogger bridges botgo's log.Logger to our log.Logger.
type webhookLogger struct {
	inner *log.Logger
}

func (l *webhookLogger) Debug(v ...interface{})               { l.inner.Debug(fmt.Sprint(v...)) }
func (l *webhookLogger) Info(v ...interface{})                { l.inner.Info(fmt.Sprint(v...)) }
func (l *webhookLogger) Warn(v ...interface{})                { l.inner.Warn(fmt.Sprint(v...)) }
func (l *webhookLogger) Error(v ...interface{})               { l.inner.Error(fmt.Sprint(v...)) }
func (l *webhookLogger) Debugf(format string, v ...interface{}) { l.inner.Debug(fmt.Sprintf(format, v...)) }
func (l *webhookLogger) Infof(format string, v ...interface{})  { l.inner.Info(fmt.Sprintf(format, v...)) }
func (l *webhookLogger) Warnf(format string, v ...interface{})  { l.inner.Warn(fmt.Sprintf(format, v...)) }
func (l *webhookLogger) Errorf(format string, v ...interface{}) { l.inner.Error(fmt.Sprintf(format, v...)) }
func (l *webhookLogger) Sync() error                            { return nil }

// BackoffConfig controls reconnection backoff behavior.
type BackoffConfig struct {
	Initial time.Duration
	Max     time.Duration
	Factor  float64
}

// DefaultBackoff is the default reconnection backoff configuration.
var DefaultBackoff = BackoffConfig{
	Initial: 1 * time.Second,
	Max:     30 * time.Second,
	Factor:  2.0,
}
