// Package bot implements the core Bot that ties all framework components together.
package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Luoyangan/LQBOT/internal/adapter"
	"github.com/Luoyangan/LQBOT/internal/bus"
	"github.com/Luoyangan/LQBOT/internal/config"
	"github.com/Luoyangan/LQBOT/internal/contract"
	"github.com/Luoyangan/LQBOT/internal/handler"
	framelog "github.com/Luoyangan/LQBOT/internal/log"
	"github.com/Luoyangan/LQBOT/internal/middleware"
	"github.com/Luoyangan/LQBOT/internal/storage"
	"github.com/Luoyangan/LQBOT/internal/types"
	"github.com/Luoyangan/LQBOT/internal/utils"
	"github.com/Luoyangan/LQBOT/internal/version"
	"github.com/Luoyangan/LQBOT/plugins/ark"
	"github.com/Luoyangan/LQBOT/plugins/echo"
	"github.com/Luoyangan/LQBOT/plugins/embed"
	"github.com/Luoyangan/LQBOT/plugins/hello"
	"github.com/Luoyangan/LQBOT/plugins/info"
	"github.com/Luoyangan/LQBOT/plugins/manage"
	"github.com/Luoyangan/LQBOT/plugins/markdown"
	"github.com/Luoyangan/LQBOT/plugins/media"
	"github.com/Luoyangan/LQBOT/plugins/ping"
	// <--new-plugin-import-here
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/constant"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/keyboard"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
)

// Bot is the core framework instance.
type Bot struct {
	cfg      *types.Config
	logger   *framelog.Logger
	storage  *storage.Storage
	eventBus *bus.EventBus
	mwChain  *middleware.Chain
	adapter  contract.Adapter
	router   *handler.CommandRouter
	api      *qqAPIImpl

	// Rate limiter (for lifecycle management)
	rateLimiter *middleware.RateLimitMiddleware

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewFromConfig creates a Bot instance from a configuration file path.
func NewFromConfig(configPath string) (*Bot, error) {
	configPath = config.Init(configPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	return New(cfg)
}

// New creates a Bot instance from a parsed Config.
func New(cfg *types.Config) (*Bot, error) {
	ensureDirs("data")

	// Initialize logger
	logger := framelog.NewWithConfig(cfg.LogLevel, cfg.LogNoColor)

	// Bridge botgo logger to our zerolog logger
	botgo.SetLogger(&botgoLogger{logger})

	// Initialize storage
	store, err := storage.New(cfg.Storage)
	if err != nil {
		return nil, fmt.Errorf("init storage: %w", err)
	}

	// Initialize API client
	api := &qqAPIImpl{
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
		sandbox:   cfg.Sandbox,
		logger:    logger,
	}

	// Initialize event bus
	eb := bus.New()

	// Initialize middleware chain
	mwChain := middleware.New()
	mwChain.Add(middleware.NewLoggingMiddleware(logger))
	rl := middleware.NewRateLimitMiddleware(logger)
	mwChain.Add(rl)

	// Initialize command router
	router := handler.NewCommandRouter()

	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{
		cfg:      cfg,
		logger:   logger,
		storage:  store,
		eventBus: eb,
		mwChain:  mwChain,
		router:   router,
		api:      api,
		rateLimiter: rl,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Register plugins (commands + event listeners)
	bot.registerPlugins()

	return bot, nil
}

// registerPlugins imports and registers all plugin packages.
// Each plugin's Register() is called with the narrow interfaces it needs.
func (b *Bot) registerPlugins() {
	ping.Register(b.router)
	echo.Register(b.router)
	hello.Register(b.router, b.eventBus, b.api)
	embed.Register(b.router, b.api)
	ark.Register(b.router, b.api)
	info.Register(b.router, b.api)
	media.Register(b.router, b.api)
	manage.Register(b.router, b.api)
	markdown.Register(b.router, b.api)
}

// Run starts the bot and blocks until shutdown.
// Returns nil on normal shutdown (SIGINT/SIGTERM), error on startup failure.
func (b *Bot) Run() error {
	b.logger.Info(version.String()+" starting",
		"access_type", b.cfg.AccessType,
	)

	// 1. Initialize adapter
	if err := b.initAdapter(); err != nil {
		return fmt.Errorf("init adapter: %w", err)
	}

	// 2. Start adapter (connect to QQ)
	if err := b.adapter.Start(b.ctx); err != nil {
		return fmt.Errorf("start adapter: %w", err)
	}

	// 3. Initialize API client with OpenAPI
	b.api.initOpenAPI()

	// 4. Start event processing loop
	b.wg.Add(1)
	go b.eventLoop()

	b.logger.Info("LQBOT is running. Press Ctrl+C to stop.")

	// 5. Wait for shutdown signal
	sig := utils.WaitForSignal()
	b.logger.Info("received signal, shutting down", "signal", sig)

	// 6. Graceful shutdown
	b.shutdown()
	return nil
}

// initAdapter creates the appropriate adapter based on config.
func (b *Bot) initAdapter() error {
	switch b.cfg.AccessType {
	case types.AccessWebSocket:
		ada, err := adapter.NewWebSocketAdapter(b.cfg.AppID, b.cfg.AppSecret, b.cfg.Sandbox, b.cfg.Intents, b.logger)
		if err != nil {
			return err
		}
		b.adapter = ada

	case types.AccessWebhook:
		b.adapter = adapter.NewWebhookAdapter(b.cfg.Webhook.Port, b.cfg.Webhook.Path, b.logger)

	default:
		return fmt.Errorf("unsupported access type: %s", b.cfg.AccessType)
	}
	return nil
}

// eventLoop processes events from the adapter.
func (b *Bot) eventLoop() {
	defer b.wg.Done()

	events := b.adapter.Events()
	for {
		select {
		case <-b.ctx.Done():
			return
		case raw, ok := <-events:
			if !ok {
				return
			}
			b.processRawEvent(raw)
		}
	}
}

// rawEvent uses the same format as QQ WSPayload: {"t":"EVENT_TYPE","d":<data>}
type rawEvent struct {
	T string          `json:"t"` // Native QQ API event type, e.g. "MESSAGE_CREATE"
	D json.RawMessage `json:"d"` // Event payload (dto.Message, dto.WSInteractionData, etc.)
}

// processRawEvent handles a raw event from the adapter using native QQ event types.
func (b *Bot) processRawEvent(raw []byte) {
	var evt rawEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		b.logger.Error("failed to parse event", "error", err)
		return
	}

	// Dispatch by native QQ event type
	switch evt.T {
	case types.EventInteractionCreate:
		b.processInteractionEvent(evt.D)

	case types.EventMessageCreate,
		types.EventAtMessageCreate,
		types.EventGroupAtMessageCreate,
		types.EventGroupMessageCreate,
		types.EventC2CMessageCreate:
		b.processMessageEvent(evt.T, evt.D)

	case types.EventGuildCreate,
		types.EventGuildDelete,
		types.EventMemberJoin,
		types.EventMemberLeave,
		types.EventMessageDelete:
		// Forward to event bus
		b.processGuildEvent(evt.T, evt.D)

	default:
		b.logger.Debug("unhandled event type", "type", evt.T)
	}
}

// processMessageEvent handles message-type events using native QQ event types.
func (b *Bot) processMessageEvent(eventType string, rawData json.RawMessage) {
	var msg dto.Message
	if err := json.Unmarshal(rawData, &msg); err != nil {
		b.logger.Error("failed to parse message data", "error", err)
		return
	}

	eventCtx := newEventContext(eventType, &msg, b.api, b.adapter.BotUserID())

	// Run through middleware chain, then dispatch
	_ = b.mwChain.Execute(eventCtx, func() error {
		// Try command routing first
		cmd, args := b.router.Resolve(eventCtx.Content())
		if cmd != nil {
			return b.executeCommand(cmd, args, eventCtx)
		}

		// Otherwise dispatch to event listeners
		b.eventBus.Publish(b.ctx, eventType, eventCtx)
		return nil
	})
}

// processGuildEvent handles guild/member events by forwarding to event bus.
func (b *Bot) processGuildEvent(eventType string, rawData json.RawMessage) {
	// Create a minimal event context for guild events
	ctx := &guildEventContext{eventType: eventType, api: b.api}
	b.eventBus.Publish(b.ctx, eventType, ctx)
}

// processInteractionEvent handles interaction.create events (button clicks, etc.).
func (b *Bot) processInteractionEvent(rawData json.RawMessage) {
	var data dto.WSInteractionData
	if err := json.Unmarshal(rawData, &data); err != nil {
		b.logger.Error("failed to parse interaction data", "error", err)
		return
	}

	ictx := newInteractionContext(&data, b.api)
	b.eventBus.Publish(b.ctx, types.EventInteractionCreate, ictx)
}

// executeCommand runs a matched command.
func (b *Bot) executeCommand(cmd *contract.Command, args []string, ctx contract.EventContext) error {
	cmdCtx := &commandContextImpl{
		args:         args,
		EventContext: ctx,
	}
	return cmd.Handler(cmdCtx)
}

// shutdown performs graceful shutdown of all components.
func (b *Bot) shutdown() {
	b.logger.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Stop adapter
	if b.adapter != nil {
		if err := b.adapter.Stop(shutdownCtx); err != nil {
			b.logger.Error("adapter stop error", "error", err)
		}
	}

	// 2. Cancel main context (stops event loop)
	b.cancel()

	// 3. Wait for goroutines to finish
	b.wg.Wait()

	// 4. Close event bus
	b.eventBus.Close()

	// 5. Close storage
	if b.storage != nil {
		if err := b.storage.Close(); err != nil {
			b.logger.Error("storage close error", "error", err)
		}
	}

	// 6. Close rate limiter (stops cleanup goroutine)
	if b.rateLimiter != nil {
		b.rateLimiter.Close()
	}

	b.logger.Info("shutdown complete")
}

// ---
// EventContext implementation — aligned with QQ Official API v2 dto.Message fields
// ---

type eventContextImpl struct {
	msg         *dto.Message
	api         contract.QQAPI
	content     string
	rawContent  string
	isMentioned bool
	scene       contract.MessageScene
}

func newEventContext(eventType string, msg *dto.Message, api contract.QQAPI, botID string) *eventContextImpl {
	// Extract mentioned user IDs
	var mentionIDs []string
	for _, m := range msg.Mentions {
		mentionIDs = append(mentionIDs, m.ID)
	}

	// Determine scene from native event type
	scene := sceneFromEventType(eventType)

	content, isMentioned := handler.IsMentioned(msg.Content, botID, mentionIDs)

	return &eventContextImpl{
		msg:         msg,
		api:         api,
		content:     content,
		rawContent:  msg.Content,
		isMentioned: isMentioned || scene == contract.SceneC2C, // C2C messages are always directed at bot
		scene:       scene,
	}
}

// sceneFromEventType maps a native QQ event type to a MessageScene.
func sceneFromEventType(eventType string) contract.MessageScene {
	switch eventType {
	case types.EventMessageCreate, types.EventAtMessageCreate:
		return contract.SceneGuild
	case types.EventGroupAtMessageCreate, types.EventGroupMessageCreate:
		return contract.SceneGroup
	case types.EventC2CMessageCreate:
		return contract.SceneC2C
	default:
		if eventType == types.EventMessageCreate {
			return contract.SceneGuild
		}
		return contract.SceneGuild
	}
}

func (c *eventContextImpl) Content() string      { return c.content }
func (c *eventContextImpl) RawContent() string   { return c.rawContent }
func (c *eventContextImpl) ChannelID() string    { return c.msg.ChannelID }
func (c *eventContextImpl) AuthorID() string     { return c.msg.Author.ID }
func (c *eventContextImpl) MessageID() string    { return c.msg.ID }
func (c *eventContextImpl) IsMentioned() bool    { return c.isMentioned }
func (c *eventContextImpl) Scene() contract.MessageScene { return c.scene }

func (c *eventContextImpl) GuildID() string { return c.msg.GuildID }
func (c *eventContextImpl) GroupID() string { return c.msg.GroupID }

func (c *eventContextImpl) Mentions() []string {
	var ids []string
	for _, m := range c.msg.Mentions {
		ids = append(ids, m.ID)
	}
	return ids
}

func (c *eventContextImpl) Attachments() []contract.Attachment {
	var atts []contract.Attachment
	for _, a := range c.msg.Attachments {
		att := contract.Attachment{
			URL:      a.URL,
			FileName: a.FileName,
			MimeType: a.ContentType,
			Width:    a.Width,
			Height:   a.Height,
		}
		atts = append(atts, att)
	}
	return atts
}

func (c *eventContextImpl) Reply(msg string) error {
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		msgToCreate := &dto.MessageToCreate{Content: msg, MsgType: dto.TextMsg, MsgID: c.msg.ID}
		ctx := context.TODO()

		if c.msg.GroupID != "" {
			_, err := impl.api.PostGroupMessage(ctx, c.msg.GroupID, msgToCreate)
			return err
		}
		if c.msg.DirectMessage || c.scene == contract.SceneC2C {
			_, err := impl.api.PostC2CMessage(ctx, c.msg.Author.ID, msgToCreate)
			return err
		}
		// Channel message: use passive reply with msg_id to avoid active push restrictions
		_, err := impl.api.PostMessage(ctx, c.msg.ChannelID, msgToCreate)
		return err
	}

	return c.api.SendMessage(c.msg.ChannelID, msg)
}

// ReplyMarkdown sends a markdown message as a passive reply with msg_id.
func (c *eventContextImpl) ReplyMarkdown(content string) error {
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		msgToCreate := &dto.MessageToCreate{
			Markdown: &dto.Markdown{Content: content},
			MsgType:  dto.MarkdownMsg,
			MsgID:    c.msg.ID,
		}
		ctx := context.TODO()

		if c.msg.GroupID != "" {
			_, err := impl.api.PostGroupMessage(ctx, c.msg.GroupID, msgToCreate)
			return err
		}
		if c.msg.DirectMessage || c.scene == contract.SceneC2C {
			_, err := impl.api.PostC2CMessage(ctx, c.msg.Author.ID, msgToCreate)
			return err
		}
		_, err := impl.api.PostMessage(ctx, c.msg.ChannelID, msgToCreate)
		return err
	}
	return c.api.SendMarkdown(c.msg.ChannelID, content)
}

// ReplyImage sends an image as a passive reply with msg_id.
func (c *eventContextImpl) ReplyImage(url string) error {
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		msgToCreate := &dto.MessageToCreate{
			Image: url,
			MsgID: c.msg.ID,
		}
		ctx := context.TODO()

		if c.msg.GroupID != "" {
			_, err := impl.api.PostGroupMessage(ctx, c.msg.GroupID, msgToCreate)
			return err
		}
		if c.msg.DirectMessage || c.scene == contract.SceneC2C {
			_, err := impl.api.PostC2CMessage(ctx, c.msg.Author.ID, msgToCreate)
			return err
		}
		_, err := impl.api.PostMessage(ctx, c.msg.ChannelID, msgToCreate)
		return err
	}
	return c.api.SendImage(c.msg.ChannelID, url)
}

// ReplyWithButtons sends markdown + buttons as a passive reply with msg_id.
// Uses raw keyboard JSON for full action field support (reply, anchor, unsupport_tips).
func (c *eventContextImpl) ReplyWithButtons(content string, buttons []contract.MessageButton) error {
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		rows := [][]contract.MessageButton{buttons}
		ctx := context.TODO()

		if c.msg.GroupID != "" {
			msg := newButtonAPIMessage(content, rows)
			msg.MsgID = c.msg.ID
			_, err := impl.api.PostGroupMessage(ctx, c.msg.GroupID, msg)
			return err
		}
		if c.msg.DirectMessage || c.scene == contract.SceneC2C {
			msg := newButtonAPIMessage(content, rows)
			msg.MsgID = c.msg.ID
			_, err := impl.api.PostC2CMessage(ctx, c.msg.Author.ID, msg)
			return err
		}
		// Channel: fallback to botgo types (PostMessage requires *dto.MessageToCreate)
		msg := buildButtonMessage(content, buttons)
		msg.MsgID = c.msg.ID
		_, err := impl.api.PostMessage(ctx, c.msg.ChannelID, msg)
		return err
	}
	return c.api.SendMessageWithButtons(c.msg.ChannelID, content, buttons)
}

// ReplyWithButtonRows sends markdown + multi-row buttons as a passive reply.
// Uses raw keyboard JSON for full action field support (reply, anchor, unsupport_tips).
func (c *eventContextImpl) ReplyWithButtonRows(content string, rows [][]contract.MessageButton) error {
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		ctx := context.TODO()

		if c.msg.GroupID != "" {
			msg := newButtonAPIMessage(content, rows)
			msg.MsgID = c.msg.ID
			_, err := impl.api.PostGroupMessage(ctx, c.msg.GroupID, msg)
			return err
		}
		if c.msg.DirectMessage || c.scene == contract.SceneC2C {
			msg := newButtonAPIMessage(content, rows)
			msg.MsgID = c.msg.ID
			_, err := impl.api.PostC2CMessage(ctx, c.msg.Author.ID, msg)
			return err
		}
		// Channel: fallback to botgo types
		msg := buildButtonRowsMessage(content, rows)
		msg.MsgID = c.msg.ID
		_, err := impl.api.PostMessage(ctx, c.msg.ChannelID, msg)
		return err
	}
	return c.api.SendMessageWithButtons(c.msg.ChannelID, content, rows[0])
}

// ReplyArk sends an Ark template message as a passive reply with msg_id.
func (c *eventContextImpl) ReplyArk(ark *contract.MessageArk) error {
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		msg := buildArkMessage(ark)
		msg.MsgID = c.msg.ID
		ctx := context.TODO()

		if c.msg.GroupID != "" {
			_, err := impl.api.PostGroupMessage(ctx, c.msg.GroupID, msg)
			return err
		}
		if c.msg.DirectMessage || c.scene == contract.SceneC2C {
			_, err := impl.api.PostC2CMessage(ctx, c.msg.Author.ID, msg)
			return err
		}
		_, err := impl.api.PostMessage(ctx, c.msg.ChannelID, msg)
		return err
	}
	return c.api.SendArkMessage(c.msg.ChannelID, ark)
}

// ReplyMarkdownTemplate sends a templated markdown message as a passive reply with msg_id.
func (c *eventContextImpl) ReplyMarkdownTemplate(templateID string, params []contract.MarkdownParam) error {
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		msg := &dto.MessageToCreate{
			Markdown: buildMarkdownTemplate(templateID, params),
			MsgType:  dto.MarkdownMsg,
			MsgID:    c.msg.ID,
		}
		ctx := context.TODO()

		if c.msg.GroupID != "" {
			_, err := impl.api.PostGroupMessage(ctx, c.msg.GroupID, msg)
			return err
		}
		if c.msg.DirectMessage || c.scene == contract.SceneC2C {
			_, err := impl.api.PostC2CMessage(ctx, c.msg.Author.ID, msg)
			return err
		}
		_, err := impl.api.PostMessage(ctx, c.msg.ChannelID, msg)
		return err
	}
	return c.api.SendMarkdownTemplate(c.msg.ChannelID, templateID, params)
}

// guildEventContext implements contract.EventContext for guild/member events.
type guildEventContext struct {
	eventType string
	api       contract.QQAPI
}

func (g *guildEventContext) Content() string    { return "" }
func (g *guildEventContext) RawContent() string { return "" }
func (g *guildEventContext) ChannelID() string  { return "" }
func (g *guildEventContext) AuthorID() string   { return "" }
func (g *guildEventContext) MessageID() string  { return "" }
func (g *guildEventContext) IsMentioned() bool  { return false }
func (g *guildEventContext) GuildID() string    { return "" }
func (g *guildEventContext) GroupID() string    { return "" }
func (g *guildEventContext) Mentions() []string { return nil }
func (g *guildEventContext) Attachments() []contract.Attachment { return nil }
func (g *guildEventContext) Scene() contract.MessageScene       { return contract.SceneGuild }
func (g *guildEventContext) Reply(msg string) error             { return nil }
func (g *guildEventContext) ReplyMarkdown(content string) error { return nil }
func (g *guildEventContext) ReplyImage(url string) error        { return nil }
func (g *guildEventContext) ReplyWithButtons(content string, buttons []contract.MessageButton) error { return nil }
func (g *guildEventContext) ReplyWithButtonRows(content string, rows [][]contract.MessageButton) error { return nil }
func (g *guildEventContext) ReplyArk(ark *contract.MessageArk) error { return nil }
func (g *guildEventContext) ReplyMarkdownTemplate(templateID string, params []contract.MarkdownParam) error { return nil }

// interactionContextImpl implements contract.InteractionContext.
type interactionContextImpl struct {
	data *contract.InteractionData
	api  contract.QQAPI
}

func newInteractionContext(d *dto.WSInteractionData, api contract.QQAPI) *interactionContextImpl {
	if d == nil {
		return &interactionContextImpl{api: api}
	}

	idata := &contract.InteractionData{
		ID:          d.ID,
		ChannelID:   d.ChannelID,
		GuildID:     d.GuildID,
		GroupOpenID: d.GroupOpenID,
		UserOpenID:  d.UserOpenID,
		Scene:       d.Scene,
	}
	idata.Type = int(d.Type)

	if d.GroupMemberOpenID != "" {
		idata.UserID = d.GroupMemberOpenID
	} else {
		idata.UserID = d.UserOpenID
	}

	if d.Data != nil && len(d.Data.Resolved) > 0 {
		var resolved dto.Resolved
		if err := json.Unmarshal(d.Data.Resolved, &resolved); err == nil {
			idata.ButtonID = resolved.ButtonID
			idata.ButtonData = resolved.ButtonData
			if resolved.MessageID != "" {
				idata.MessageID = resolved.MessageID
			}
		}
	}

	return &interactionContextImpl{data: idata, api: api}
}

func (c *interactionContextImpl) InteractionData() *contract.InteractionData { return c.data }
func (c *interactionContextImpl) ButtonID() string                          { return c.data.ButtonID }
func (c *interactionContextImpl) ButtonData() string                        { return c.data.ButtonData }
func (c *interactionContextImpl) GroupOpenID() string { return c.data.GroupOpenID }
func (c *interactionContextImpl) UserOpenID() string  { return c.data.UserOpenID }
func (c *interactionContextImpl) UserID() string      { return c.data.UserID }
func (c *interactionContextImpl) ChannelID() string                         { return c.data.ChannelID }
func (c *interactionContextImpl) MessageID() string                         { return c.data.MessageID }
func (c *interactionContextImpl) Reply(msg string) error {
	// 使用 botgo API 直接发送被动回复，优先使用存储的 msg_id
	if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
		msgToCreate := &dto.MessageToCreate{
			Content: msg,
			MsgType: dto.TextMsg,
		}
		// 尝试查找存储的按钮消息 msg_id 用于被动回复
		hasMsgID := false
		if storedMsgID := contract.GetButtonMsgID(c.data.GroupOpenID); storedMsgID != "" {
			msgToCreate.MsgID = storedMsgID
			// 设置消息序号防去重：同一 msg_id 的每次回复使用递增序号
			msgToCreate.MsgSeq = contract.NextMsgSeq(c.data.GroupOpenID)
			hasMsgID = true
		}
		ctx := context.TODO()
		if c.data.GroupOpenID != "" {
			// 有 msg_id 用被动回复，没有则用交互回调（type 2 markdown）
			if hasMsgID {
				_, err := impl.api.PostGroupMessage(ctx, c.data.GroupOpenID, msgToCreate)
				return err
			}
			// Fallback: 使用交互回调 type 2（至少用户能看到反馈）
			return c.api.ReplyInteraction(c.data.ID, msg)
		}
		if c.data.ChannelID != "" {
			_, err := impl.api.PostMessage(ctx, c.data.ChannelID, msgToCreate)
			return err
		}
		// C2C interaction: use PostC2CMessage
		if c.data.UserOpenID != "" {
			_, err := impl.api.PostC2CMessage(ctx, c.data.UserOpenID, msgToCreate)
			return err
		}
	}
	// Fallback
	if c.data.GroupOpenID != "" {
		return c.api.SendGroupMessage(c.data.GroupOpenID, msg)
	}
	return c.api.SendMessage(c.data.ChannelID, msg)
}

func (c *interactionContextImpl) ReplyMarkdown(content string) error {
	if c.data.GroupOpenID != "" {
		return c.api.SendGroupMarkdown(c.data.GroupOpenID, content)
	}
	return c.api.SendMarkdown(c.data.ChannelID, content)
}

func (c *interactionContextImpl) ReplyImage(url string) error {
	if c.data.GroupOpenID != "" {
		return c.api.SendGroupMessage(c.data.GroupOpenID, "[图片] "+url)
	}
	return c.api.SendImage(c.data.ChannelID, url)
}

func (c *interactionContextImpl) ReplyWithButtons(content string, buttons []contract.MessageButton) error {
	if c.data.GroupOpenID != "" {
		return c.api.SendGroupMessageWithButtons(c.data.GroupOpenID, content, buttons)
	}
	return c.api.SendMessageWithButtons(c.data.ChannelID, content, buttons)
}

func (c *interactionContextImpl) ReplyWithButtonRows(content string, rows [][]contract.MessageButton) error {
	ctx := context.TODO()
	if c.data.GroupOpenID != "" {
		if impl, ok := c.api.(*qqAPIImpl); ok && impl.api != nil {
			msg := newButtonAPIMessage(content, rows)
			if storedMsgID := contract.GetButtonMsgID(c.data.GroupOpenID); storedMsgID != "" {
				msg.MsgID = storedMsgID
				msg.MsgSeq = contract.NextMsgSeq(c.data.GroupOpenID)
			}
			_, err := impl.api.PostGroupMessage(ctx, c.data.GroupOpenID, msg)
			return err
		}
		return c.api.SendGroupMessageWithButtons(c.data.GroupOpenID, content, rows[0])
	}
	return c.api.SendMessageWithButtons(c.data.ChannelID, content, rows[0])
}

func (c *interactionContextImpl) ReplyArk(ark *contract.MessageArk) error {
	if c.data.GroupOpenID != "" {
		return c.api.SendGroupArkMessage(c.data.GroupOpenID, ark)
	}
	return c.api.SendArkMessage(c.data.ChannelID, ark)
}

func (c *interactionContextImpl) ReplyMarkdownTemplate(templateID string, params []contract.MarkdownParam) error {
	if c.data.GroupOpenID != "" {
		return c.api.SendGroupMarkdownTemplate(c.data.GroupOpenID, templateID, params)
	}
	return c.api.SendMarkdownTemplate(c.data.ChannelID, templateID, params)
}
func (c *interactionContextImpl) Callback(content string) error {
	// ReplyInteraction acknowledges the interaction AND sends a reply to the user.
	// This is the QQ API's built-in interaction callback mechanism and is NOT
	// subject to active push limits (the reply is part of the interaction protocol).
	if c.data.ID != "" {
		return c.api.ReplyInteraction(c.data.ID, content)
	}
	return nil
}
func (c *interactionContextImpl) DeferReply() error {
	return c.api.PutInteraction(c.data.ID, `{"type":0}`)
}

// EventContext interface implementation (for EventBus dispatch compatibility).
func (c *interactionContextImpl) Content() string    { return "" }
func (c *interactionContextImpl) RawContent() string { return "" }
func (c *interactionContextImpl) AuthorID() string   { return c.data.UserID }
func (c *interactionContextImpl) IsMentioned() bool  { return false }
func (c *interactionContextImpl) GuildID() string    { return c.data.GuildID }
func (c *interactionContextImpl) GroupID() string    { return "" }
func (c *interactionContextImpl) Mentions() []string { return nil }
func (c *interactionContextImpl) Attachments() []contract.Attachment { return nil }
func (c *interactionContextImpl) Scene() contract.MessageScene       { return contract.SceneC2C }

// commandContextImpl implements contract.CommandContext.
type commandContextImpl struct {
	args []string
	contract.EventContext
}

func (c *commandContextImpl) Args() []string   { return c.args }
func (c *commandContextImpl) Arg(i int) string {
	if i >= 0 && i < len(c.args) {
		return c.args[i]
	}
	return ""
}
func (c *commandContextImpl) ArgCount() int { return len(c.args) }

// ---
// QQAPI implementation using botgo OpenAPI
// ---

type qqAPIImpl struct {
	appID     string
	appSecret string
	sandbox   bool
	logger    *framelog.Logger
	api       openapi.OpenAPI
	mu        sync.Mutex
}

func (a *qqAPIImpl) initOpenAPI() {
	if a.appID == "" || a.appID == "your_app_id_here" {
		a.logger.Warn("QQAPI: app_id not configured, API calls will be no-ops")
		return
	}

	// Note: Token endpoint (https://bots.qq.com/app/getAppAccessToken) is the SAME
	// for both production and sandbox environments. Do NOT change TokenDomain for sandbox.

	credentials := &token.QQBotCredentials{
		AppID:     a.appID,
		AppSecret: a.appSecret,
	}
	tokenSource := token.NewQQBotTokenSource(credentials)
	a.mu.Lock()
	if a.sandbox {
		a.api = botgo.NewSandboxOpenAPI(a.appID, tokenSource)
	} else {
		a.api = botgo.NewOpenAPI(a.appID, tokenSource)
	}
	a.mu.Unlock()

	if a.sandbox {
		a.logger.Info("QQAPI initialized (sandbox mode)")
	} else {
		a.logger.Info("QQAPI initialized")
	}
}

func (a *qqAPIImpl) sendToChannel(id string, msg *dto.MessageToCreate) error {
	a.mu.Lock()
	api := a.api
	a.mu.Unlock()
	if api == nil {
		return nil
	}
	_, err := api.PostMessage(context.TODO(), id, msg)
	if err != nil {
		a.logger.Error("send message failed", "error", err, "target_id", id)
	}
	return err
}

func (a *qqAPIImpl) sendToGroup(id string, msg *dto.MessageToCreate) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	_, err := api.PostGroupMessage(context.TODO(), id, msg)
	if err != nil {
		a.logger.Error("send group message failed", "error", err, "target_id", id)
	}
	return err
}

func (a *qqAPIImpl) SendMessage(channelID, content string) error {
	msg := &dto.MessageToCreate{Content: content, MsgType: dto.TextMsg}
	return a.sendToChannel(channelID, msg)
}

func (a *qqAPIImpl) SendMarkdown(channelID, content string) error {
	msg := &dto.MessageToCreate{
		Markdown: &dto.Markdown{Content: content},
		MsgType:  dto.MarkdownMsg,
	}
	return a.sendToChannel(channelID, msg)
}

func (a *qqAPIImpl) ReplyMarkdown(channelID, msgID, content string) error {
	msg := &dto.MessageToCreate{
		Markdown: &dto.Markdown{Content: content},
		MsgType:  dto.MarkdownMsg,
		MsgID:    msgID,
	}
	return a.sendToChannel(channelID, msg)
}

func (a *qqAPIImpl) SendMarkdownTemplate(channelID, templateID string, params []contract.MarkdownParam) error {
	msg := &dto.MessageToCreate{
		Markdown: buildMarkdownTemplate(templateID, params),
		MsgType:  dto.MarkdownMsg,
	}
	return a.sendToChannel(channelID, msg)
}

func (a *qqAPIImpl) SendImage(channelID, imageURL string) error {
	msg := &dto.MessageToCreate{Image: imageURL}
	return a.sendToChannel(channelID, msg)
}

func (a *qqAPIImpl) ReplyImage(channelID, msgID, imageURL string) error {
	msg := &dto.MessageToCreate{Image: imageURL, MsgID: msgID}
	return a.sendToChannel(channelID, msg)
}

func (a *qqAPIImpl) SendMessageWithButtons(channelID string, content string, buttons []contract.MessageButton) error {
	msg := buildButtonMessage(content, buttons)
	return a.sendToChannel(channelID, msg)
}

// buildButtonMessage creates a MessageToCreate with keyboard buttons.
// Per QQ API docs, keyboards are only supported on markdown messages.
func buildButtonMessage(content string, buttons []contract.MessageButton) *dto.MessageToCreate {
	var kbButtons []*keyboard.Button
	for _, btn := range buttons {
		kbButtons = append(kbButtons, buildBotgoButton(&btn))
	}

	return &dto.MessageToCreate{
		MsgType: dto.MarkdownMsg,
		Markdown: &dto.Markdown{
			Content: content,
		},
		Keyboard: &keyboard.MessageKeyboard{
			Content: &keyboard.CustomKeyboard{
				Rows: []*keyboard.Row{{Buttons: kbButtons}},
			},
		},
	}
}

// buildBotgoButton converts a contract.MessageButton to a botgo keyboard.Button,
// mapping the supported action type and permission fields.
func buildBotgoButton(btn *contract.MessageButton) *keyboard.Button {
	data := btn.Data
	if data == "" {
		data = btn.ID
	}

	// Map action type.
	// Priority: URL field > ActionType field > default (command)
	var actionType keyboard.ActionType
	if btn.URL != "" {
		// URL field takes priority → jump button (ActionType=0)
		actionType = keyboard.ActionTypeURL
		data = btn.URL
	} else {
		// Go int zero value 0 is ambiguous (unset vs QQ API URL=0).
		// Use URL field for jump buttons. Treat 0/2 as command, 1 as callback.
		switch btn.ActionType {
		case 0:
			actionType = keyboard.ActionTypeAtBot // default: command
		case 1:
			actionType = keyboard.ActionTypeCallback
		case 2:
			actionType = keyboard.ActionTypeAtBot
		default:
			actionType = keyboard.ActionTypeAtBot
		}
	}

	// Map permission type
	// 注意: Permission 的 Go 零值为 0 (=specific users)，但业务上未设置时应默认 everyone
	permType := keyboard.PermissionTypAll
	switch btn.Permission {
	case 0:
		if len(btn.SpecifyUserIDs) > 0 {
			permType = keyboard.PermissionTypeSpecifyUserIDs
		}
		// 无 SpecifyUserIDs → 保持默认 everyone
	case 1:
		permType = keyboard.PermissionTypManager
	case 2:
		permType = keyboard.PermissionTypAll
	case 3:
		permType = keyboard.PermissionTypSpecifyRoleIDs
	}

	perm := &keyboard.Permission{Type: permType}
	if len(btn.SpecifyUserIDs) > 0 {
		perm.SpecifyUserIDs = btn.SpecifyUserIDs
	}
	if len(btn.SpecifyRoleIDs) > 0 {
		perm.SpecifyRoleIDs = btn.SpecifyRoleIDs
	}

	action := &keyboard.Action{
		Type:       actionType,
		Permission: perm,
		Data:       data,
		Enter:      btn.Enter,
	}

	return &keyboard.Button{
		ID: btn.ID,
		RenderData: &keyboard.RenderData{
			Label:        btn.Label,
			VisitedLabel: btn.Label,
			Style:        btn.Style,
		},
		Action: action,
	}
}

// ── Full action support via raw JSON (includes fields botgo doesn't have) ──

// fullAction mirrors QQ API button action with ALL supported fields.
type fullAction struct {
	Type                 int             `json:"type"`
	Permission           *fullPermission `json:"permission,omitempty"`
	Data                 string          `json:"data,omitempty"`
	Enter                bool            `json:"enter,omitempty"`
	Reply                bool            `json:"reply,omitempty"`
	Anchor               int             `json:"anchor,omitempty"`
	UnsupportTips        string          `json:"unsupport_tips,omitempty"`
	AtBotShowChannelList bool            `json:"at_bot_show_channel_list,omitempty"`
}

type fullPermission struct {
	Type            int      `json:"type"`
	SpecifyUserIDs  []string `json:"specify_user_ids,omitempty"`
	SpecifyRoleIDs  []string `json:"specify_role_ids,omitempty"`
}

// fullButton mirrors QQ API button with full action support.
type fullButton struct {
	ID         string      `json:"id"`
	RenderData *renderData `json:"render_data"`
	Action     *fullAction `json:"action"`
}

type renderData struct {
	Label        string `json:"label"`
	VisitedLabel string `json:"visited_label"`
	Style        int    `json:"style"`
}

// buildKeyboardJSON builds a complete keyboard JSON with full action field support.
func buildKeyboardJSON(content string, rows [][]contract.MessageButton) []byte {
	var kbRows []map[string]interface{}
	for _, buttons := range rows {
		var btns []*fullButton
		for _, btn := range buttons {
			btns = append(btns, buildFullButton(&btn))
		}
		kbRows = append(kbRows, map[string]interface{}{"buttons": btns})
	}

	keyboard := map[string]interface{}{
		"content": map[string]interface{}{
			"rows": kbRows,
		},
	}
	jsonBytes, _ := json.Marshal(keyboard)
	return jsonBytes
}

// buildFullButton converts a contract.MessageButton to a fullButton with all fields.
func buildFullButton(btn *contract.MessageButton) *fullButton {
	data := btn.Data
	if data == "" {
		data = btn.ID
	}

	// ActionType: URL field > ActionType field > default (command).
	actionType := btn.ActionType
	if btn.URL != "" {
		actionType = 0 // jump
		data = btn.URL
	} else if actionType == 0 {
		actionType = 2 // default: command (@bot)
	}

	perm := &fullPermission{Type: 2} // default: everyone
	switch btn.Permission {
	case 0:
		if len(btn.SpecifyUserIDs) > 0 {
			perm.Type = 0
			perm.SpecifyUserIDs = btn.SpecifyUserIDs
		}
		// 无 SpecifyUserIDs → 保持默认 everyone
	case 1:
		perm.Type = 1
	case 2:
		perm.Type = 2
	case 3:
		perm.Type = 3
		perm.SpecifyRoleIDs = btn.SpecifyRoleIDs
	}

	return &fullButton{
		ID: btn.ID,
		RenderData: &renderData{
			Label:        btn.Label,
			VisitedLabel: btn.Label,
			Style:        btn.Style,
		},
		Action: &fullAction{
			Type:         actionType,
			Permission:   perm,
			Data:         data,
			Enter:        btn.Enter,
			Reply:        btn.Reply,
			Anchor:       btn.Anchor,
			UnsupportTips: btn.UnsupportTips,
		},
	}
}

// buttonAPIMessage implements dto.APIMessage with raw keyboard JSON for full action support.
type buttonAPIMessage struct {
	Content  string          `json:"content,omitempty"`
	MsgType  dto.MessageType `json:"msg_type,omitempty"`
	MsgID    string          `json:"msg_id,omitempty"`
	MsgSeq   uint32          `json:"msg_seq,omitempty"`
	Markdown *dto.Markdown   `json:"markdown,omitempty"`
	Keyboard json.RawMessage `json:"keyboard,omitempty"`
}

func (m *buttonAPIMessage) GetEventID() string  { return "" }
func (m *buttonAPIMessage) GetSendType() dto.SendType { return dto.Text }

// newButtonAPIMessage creates a buttonAPIMessage with full action support.
func newButtonAPIMessage(content string, rows [][]contract.MessageButton) *buttonAPIMessage {
	keyboardJSON := buildKeyboardJSON(content, rows)
	return &buttonAPIMessage{
		MsgType: dto.MarkdownMsg,
		Markdown: &dto.Markdown{
			Content: content,
		},
		Keyboard: keyboardJSON,
	}
}

// buildButtonRowsMessage creates a MessageToCreate with multi-row keyboard buttons.
// Each inner slice represents one row of buttons in the keyboard.
func buildButtonRowsMessage(content string, rows [][]contract.MessageButton) *dto.MessageToCreate {
	var kbRows []*keyboard.Row
	for _, buttons := range rows {
		var kbButtons []*keyboard.Button
		for _, btn := range buttons {
			kbButtons = append(kbButtons, buildBotgoButton(&btn))
		}
		kbRows = append(kbRows, &keyboard.Row{Buttons: kbButtons})
	}

	return &dto.MessageToCreate{
		MsgType: dto.MarkdownMsg,
		Markdown: &dto.Markdown{
			Content: content,
		},
		Keyboard: &keyboard.MessageKeyboard{
			Content: &keyboard.CustomKeyboard{
				Rows: kbRows,
			},
		},
	}
}

func (a *qqAPIImpl) SendGroupMessage(groupID string, content string) error {
	msg := &dto.MessageToCreate{Content: content}
	a.mu.Lock()
	api := a.api
	a.mu.Unlock()
	if api == nil {
		return nil
	}
	_, err := api.PostGroupMessage(context.TODO(), groupID, msg)
	if err != nil {
		a.logger.Error("send group message failed", "error", err, "target_id", groupID)
	}
	return err
}

func (a *qqAPIImpl) SendGroupMessageWithButtons(groupID string, content string, buttons []contract.MessageButton) error {
	msg := buildButtonMessage(content, buttons)
	return a.sendToGroup(groupID, msg)
}

// ── New message type implementations ──

func (a *qqAPIImpl) SendEmbedMessage(channelID string, embed *contract.MessageEmbed) error {
	// Convert contract embed to botgo embed
	bgEmbed := &dto.Embed{
		Title:  embed.Title,
		Prompt: embed.Prompt,
	}
	if embed.Thumbnail != "" {
		bgEmbed.Thumbnail = dto.MessageEmbedThumbnail{URL: embed.Thumbnail}
	}
	for _, f := range embed.Fields {
		bgEmbed.Fields = append(bgEmbed.Fields, &dto.EmbedField{Name: f.Name})
	}
	msg := &dto.MessageToCreate{
		Embed:   bgEmbed,
		MsgType: dto.EmbedMsg,
	}
	return a.sendToChannel(channelID, msg)
}

// buildArkMessage converts a contract.MessageArk to a dto.MessageToCreate.
func buildArkMessage(ark *contract.MessageArk) *dto.MessageToCreate {
	bgArk := &dto.Ark{
		TemplateID: ark.TemplateID,
	}
	for _, kv := range ark.KV {
		bgKV := &dto.ArkKV{Key: kv.Key, Value: kv.Value}
		for _, obj := range kv.Obj {
			bgObj := &dto.ArkObj{}
			for _, okv := range obj.ObjKV {
				bgObj.ObjKV = append(bgObj.ObjKV, &dto.ArkObjKV{Key: okv.Key, Value: okv.Value})
			}
			bgKV.Obj = append(bgKV.Obj, bgObj)
		}
		bgArk.KV = append(bgArk.KV, bgKV)
	}
	return &dto.MessageToCreate{
		Ark:     bgArk,
		MsgType: dto.ArkMsg,
	}
}

// buildMarkdownTemplate converts a templateID + params to a dto.Markdown.
func buildMarkdownTemplate(templateID string, params []contract.MarkdownParam) *dto.Markdown {
	bgParams := make([]*dto.MarkdownParams, len(params))
	for i, p := range params {
		bgParams[i] = &dto.MarkdownParams{Key: p.Key, Values: p.Values}
	}
	return &dto.Markdown{
		CustomTemplateID: templateID,
		Params:           bgParams,
	}
}

func (a *qqAPIImpl) SendArkMessage(channelID string, ark *contract.MessageArk) error {
	return a.sendToChannel(channelID, buildArkMessage(ark))
}

func (a *qqAPIImpl) SendGroupArkMessage(groupID string, ark *contract.MessageArk) error {
	msg := buildArkMessage(ark)
	a.mu.Lock()
	api := a.api
	a.mu.Unlock()
	if api == nil {
		return nil
	}
	_, err := api.PostGroupMessage(context.TODO(), groupID, msg)
	return err
}

func (a *qqAPIImpl) SendC2CArkMessage(userID string, ark *contract.MessageArk) error {
	msg := buildArkMessage(ark)
	a.mu.Lock()
	api := a.api
	a.mu.Unlock()
	if api == nil {
		return nil
	}
	_, err := api.PostC2CMessage(context.TODO(), userID, msg)
	return err
}

func (a *qqAPIImpl) SendRichMedia(channelID string, media *contract.RichMedia) error {
	// If URL is provided but FileInfo is empty, auto-upload first
	if media.FileInfo == "" && media.URL != "" {
		fileInfo, err := a.UploadChannelMedia(channelID, media.FileType, media.URL)
		if err != nil {
			return fmt.Errorf("upload channel media: %w", err)
		}
		media.FileInfo = fileInfo
	}

	msg := &dto.MessageToCreate{
		MsgType: dto.RichMediaMsg,
	}
	if media.FileInfo != "" {
		msg.Media = &dto.MediaInfo{FileInfo: []byte(media.FileInfo)}
	}
	if media.Content != "" {
		msg.Content = media.Content
	}
	return a.sendToChannel(channelID, msg)
}

func (a *qqAPIImpl) SendGroupMarkdown(groupID string, content string) error {
	msg := &dto.MessageToCreate{
		Markdown: &dto.Markdown{Content: content},
		MsgType:  dto.MarkdownMsg,
	}
	return a.sendToGroup(groupID, msg)
}

func (a *qqAPIImpl) SendGroupMarkdownTemplate(groupID, templateID string, params []contract.MarkdownParam) error {
	msg := &dto.MessageToCreate{
		Markdown: buildMarkdownTemplate(templateID, params),
		MsgType:  dto.MarkdownMsg,
	}
	return a.sendToGroup(groupID, msg)
}

func (a *qqAPIImpl) SendGroupRichMedia(groupID string, media *contract.RichMedia) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}

	// Step 1: Upload the file to get file_info
	fileInfo, err := a.UploadGroupMedia(groupID, media.FileType, media.URL)
	if err != nil {
		return fmt.Errorf("upload group media: %w", err)
	}

	// Step 2: Send as a passive reply with msg_id
	// Build the request body manually to avoid botgo's base64 encoding of []byte.
	baseURL := constant.APIDomain
	if a.sandbox {
		baseURL = constant.SandBoxAPIDomain
	}

	body := map[string]interface{}{
		"msg_type": 7,
		"media": map[string]string{
			"file_info": fileInfo,
		},
	}
	if media.MsgID != "" {
		body["msg_id"] = media.MsgID
	}
	if media.Content != "" {
		body["content"] = media.Content
	}

	uploadURL := fmt.Sprintf("%s/v2/groups/%s/messages", baseURL, groupID)
	_, err = api.Transport(context.TODO(), "POST", uploadURL, body)
	if err != nil {
		a.logger.Error("send group rich media failed", "error", err, "target_id", groupID)
	}
	return err
}

// ── C2C message sending ──

func (a *qqAPIImpl) sendToC2C(userID string, msg *dto.MessageToCreate) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	_, err := api.PostC2CMessage(context.TODO(), userID, msg)
	if err != nil {
		a.logger.Error("send c2c message failed", "error", err, "target_id", userID)
	}
	return err
}

func (a *qqAPIImpl) SendC2CMessage(userID, content string) error {
	msg := &dto.MessageToCreate{Content: content, MsgType: dto.TextMsg}
	return a.sendToC2C(userID, msg)
}

func (a *qqAPIImpl) SendC2CMarkdown(userID, content string) error {
	msg := &dto.MessageToCreate{
		Markdown: &dto.Markdown{Content: content},
		MsgType:  dto.MarkdownMsg,
	}
	return a.sendToC2C(userID, msg)
}

func (a *qqAPIImpl) SendC2CMarkdownTemplate(userID, templateID string, params []contract.MarkdownParam) error {
	msg := &dto.MessageToCreate{
		Markdown: buildMarkdownTemplate(templateID, params),
		MsgType:  dto.MarkdownMsg,
	}
	return a.sendToC2C(userID, msg)
}

func (a *qqAPIImpl) SendC2CRichMedia(userID string, media *contract.RichMedia) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}

	// Step 1: Upload the file to get file_info
	fileInfo, err := a.UploadC2CMedia(userID, media.FileType, media.URL)
	if err != nil {
		return fmt.Errorf("upload c2c media: %w", err)
	}

	// Step 2: Send as a passive reply with msg_id
	// Build the request body manually to avoid botgo's base64 encoding of []byte.
	baseURL := constant.APIDomain
	if a.sandbox {
		baseURL = constant.SandBoxAPIDomain
	}

	body := map[string]interface{}{
		"msg_type": 7,
		"media": map[string]string{
			"file_info": fileInfo,
		},
	}
	if media.MsgID != "" {
		body["msg_id"] = media.MsgID
	}
	if media.Content != "" {
		body["content"] = media.Content
	}

	uploadURL := fmt.Sprintf("%s/v2/users/%s/messages", baseURL, userID)
	_, err = api.Transport(context.TODO(), "POST", uploadURL, body)
	if err != nil {
		a.logger.Error("send c2c rich media failed", "error", err, "target_id", userID)
	}
	return err
}

func (a *qqAPIImpl) SendC2CMessageWithButtons(userID string, content string, buttons []contract.MessageButton) error {
	msg := buildButtonMessage(content, buttons)
	return a.sendToC2C(userID, msg)
}

// ── Message management ──

func (a *qqAPIImpl) DeleteMessage(channelID, messageID string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	return api.RetractMessage(context.TODO(), channelID, messageID)
}

func (a *qqAPIImpl) DeleteGroupMessage(groupID, messageID string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	return api.RetractGroupMessage(context.TODO(), groupID, messageID)
}

func (a *qqAPIImpl) DeleteC2CMessage(userID, messageID string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	return api.RetractC2CMessage(context.TODO(), userID, messageID)
}

func (a *qqAPIImpl) PinMessage(channelID, messageID string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	_, err := api.AddPins(context.TODO(), channelID, messageID)
	return err
}

func (a *qqAPIImpl) UnpinMessage(channelID, messageID string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	return api.DeletePins(context.TODO(), channelID, messageID)
}

// ── Reactions ──

func (a *qqAPIImpl) CreateReaction(channelID, messageID, emoji string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	e := parseEmoji(emoji)
	return api.CreateMessageReaction(context.TODO(), channelID, messageID, e)
}

func (a *qqAPIImpl) DeleteReaction(channelID, messageID, emoji string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	e := parseEmoji(emoji)
	return api.DeleteOwnMessageReaction(context.TODO(), channelID, messageID, e)
}

// parseEmoji parses a "type:id" string into a dto.Emoji struct.
// Format: "1:4" for system emoji (type=1, id=4), or "2:❤️" for unicode emoji.
func parseEmoji(s string) dto.Emoji {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		typeVal, err := strconv.Atoi(parts[0])
		if err == nil && typeVal > 0 {
			return dto.Emoji{Type: typeVal, ID: parts[1]}
		}
	}
	// Default: treat as unicode emoji
	return dto.Emoji{Type: 2, ID: s}
}

// ── Active push ──

// activeC2CBody implements dto.APIMessage with is_wakeup support for active push.
type activeC2CBody struct {
	Content  string `json:"content"`
	MsgType  int    `json:"msg_type"`
	IsWakeup bool   `json:"is_wakeup"`
}

func (b *activeC2CBody) GetEventID() string       { return "" }
func (b *activeC2CBody) GetSendType() dto.SendType { return 0 }

func (a *qqAPIImpl) SendActiveC2CMessage(userID, content string, isWakeup bool) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	body := &activeC2CBody{
		Content:  content,
		MsgType:  int(dto.TextMsg),
		IsWakeup: isWakeup,
	}
	_, err := api.PostC2CMessage(context.TODO(), userID, body)
	if err != nil {
		a.logger.Error("send active c2c message failed", "error", err, "target_id", userID)
	}
	return err
}

// ── Guild / Channel info ──

func (a *qqAPIImpl) GetGuild(guildID string) (*contract.Guild, error) {
	api := a.getAPI()
	if api == nil {
		return nil, nil
	}
	g, err := api.Guild(context.TODO(), guildID)
	if err != nil {
		a.logger.Error("get guild failed", "error", err, "guild_id", guildID)
		return nil, err
	}
	return &contract.Guild{
		ID:          g.ID,
		Name:        g.Name,
		Icon:        g.Icon,
		OwnerID:     g.OwnerID,
		MemberCount: g.MemberCount,
		MaxMembers:  g.MaxMembers,
		Desc:        g.Desc,
	}, nil
}

func (a *qqAPIImpl) GetChannel(channelID string) (*contract.Channel, error) {
	api := a.getAPI()
	if api == nil {
		return nil, nil
	}
	c, err := api.Channel(context.TODO(), channelID)
	if err != nil {
		a.logger.Error("get channel failed", "error", err, "channel_id", channelID)
		return nil, err
	}
	return &contract.Channel{
		ID:       c.ID,
		GuildID:  c.GuildID,
		Name:     c.Name,
		Type:     int(c.Type),
		ParentID: c.ParentID,
	}, nil
}

func (a *qqAPIImpl) GetGuildMember(guildID, userID string) (*contract.Member, error) {
	api := a.getAPI()
	if api == nil {
		return nil, nil
	}
	m, err := api.GuildMember(context.TODO(), guildID, userID)
	if err != nil {
		a.logger.Error("get guild member failed", "error", err, "guild_id", guildID, "user_id", userID)
		return nil, err
	}
	member := &contract.Member{
		Nick:     m.Nick,
		Roles:    m.Roles,
		JoinedAt: string(m.JoinedAt),
	}
	if m.User != nil {
		member.User = &contract.User{
			ID:       m.User.ID,
			Username: m.User.Username,
			Avatar:   m.User.Avatar,
			Bot:      m.User.Bot,
		}
	}
	return member, nil
}

// uploadChannelMediaUploadRequest is the request body for channel file upload.
type uploadChannelMediaUploadRequest struct {
	FileType    int    `json:"file_type"`
	URL         string `json:"url"`
	SrvSendMsg  bool   `json:"srv_send_msg"`
}

// uploadChannelMediaUploadResponse is the response body for channel file upload.
type uploadChannelMediaUploadResponse struct {
	FileInfo string `json:"file_info"`
}

func (a *qqAPIImpl) UploadChannelMedia(channelID string, fileType int, url string) (string, error) {
	api := a.getAPI()
	if api == nil {
		return "", nil
	}

	// Build the upload URL (base URL depends on sandbox mode)
	baseURL := constant.APIDomain
	if a.sandbox {
		baseURL = constant.SandBoxAPIDomain
	}
	uploadURL := fmt.Sprintf("%s/channels/%s/files", baseURL, channelID)

	body := &uploadChannelMediaUploadRequest{
		FileType:   fileType,
		URL:        url,
		SrvSendMsg: false,
	}

	resp, err := api.Transport(context.TODO(), "POST", uploadURL, body)
	if err != nil {
		a.logger.Error("upload channel media failed", "error", err, "channel_id", channelID)
		return "", err
	}

	var result uploadChannelMediaUploadResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		a.logger.Error("upload channel media: parse response failed", "error", err)
		return "", err
	}

	if result.FileInfo == "" {
		return "", fmt.Errorf("upload channel media: empty file_info in response")
	}

	a.logger.Info("channel media uploaded",
		"channel_id", channelID,
		"file_type", fileType,
	)

	return result.FileInfo, nil
}

// uploadGroupMediaUploadRequest is the request body for group file upload.
type uploadGroupMediaUploadRequest struct {
	FileType   int    `json:"file_type"`
	URL        string `json:"url"`
	SrvSendMsg bool   `json:"srv_send_msg"`
}

// uploadGroupMediaUploadResponse is the response body for group file upload.
type uploadGroupMediaUploadResponse struct {
	FileInfo string `json:"file_info"`
}

// UploadGroupMedia uploads a media file to a group and returns the file_info string.
func (a *qqAPIImpl) UploadGroupMedia(groupID string, fileType int, url string) (string, error) {
	api := a.getAPI()
	if api == nil {
		return "", nil
	}

	baseURL := constant.APIDomain
	if a.sandbox {
		baseURL = constant.SandBoxAPIDomain
	}
	uploadURL := fmt.Sprintf("%s/v2/groups/%s/files", baseURL, groupID)

	body := &uploadGroupMediaUploadRequest{
		FileType:   fileType,
		URL:        url,
		SrvSendMsg: false,
	}

	resp, err := api.Transport(context.TODO(), "POST", uploadURL, body)
	if err != nil {
		a.logger.Error("upload group media failed", "error", err, "group_id", groupID)
		return "", err
	}

	var result uploadGroupMediaUploadResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		a.logger.Error("upload group media: parse response failed", "error", err)
		return "", err
	}

	if result.FileInfo == "" {
		return "", fmt.Errorf("upload group media: empty file_info in response")
	}

	a.logger.Info("group media uploaded",
		"group_id", groupID,
		"file_type", fileType,
	)

	return result.FileInfo, nil
}

// UploadC2CMedia uploads a media file for C2C and returns the file_info string.
func (a *qqAPIImpl) UploadC2CMedia(userID string, fileType int, url string) (string, error) {
	api := a.getAPI()
	if api == nil {
		return "", nil
	}

	baseURL := constant.APIDomain
	if a.sandbox {
		baseURL = constant.SandBoxAPIDomain
	}
	uploadURL := fmt.Sprintf("%s/v2/users/%s/files", baseURL, userID)

	body := &uploadGroupMediaUploadRequest{
		FileType:   fileType,
		URL:        url,
		SrvSendMsg: false,
	}

	resp, err := api.Transport(context.TODO(), "POST", uploadURL, body)
	if err != nil {
		a.logger.Error("upload c2c media failed", "error", err, "user_id", userID)
		return "", err
	}

	var result uploadGroupMediaUploadResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		a.logger.Error("upload c2c media: parse response failed", "error", err)
		return "", err
	}

	if result.FileInfo == "" {
		return "", fmt.Errorf("upload c2c media: empty file_info in response")
	}

	a.logger.Info("c2c media uploaded",
		"user_id", userID,
		"file_type", fileType,
	)

	return result.FileInfo, nil
}

func (a *qqAPIImpl) getAPI() openapi.OpenAPI {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.api
}

func (a *qqAPIImpl) PutInteraction(interactionID string, body string) error {
	api := a.getAPI()
	if api == nil {
		return nil
	}
	return api.PutInteraction(context.TODO(), interactionID, body)
}

func (a *qqAPIImpl) ReplyInteraction(interactionID string, content string) error {
	encoded, _ := json.Marshal(content)
	// type:2 = reply with markdown (visible in group/channel, bypasses active push limits)
	body := fmt.Sprintf(`{"type":2,"data":{"markdown":{"content":%s}}}`, string(encoded))
	return a.PutInteraction(interactionID, body)
}

// ---
// botgoLogger bridges botgo's log.Logger to our zerolog.Logger.
// ---

type botgoLogger struct {
	*framelog.Logger
}

func (l *botgoLogger) Debug(v ...interface{})                 { l.Logger.Debug(fmt.Sprint(v...)) }
func (l *botgoLogger) Info(v ...interface{})                  { l.Logger.Info(fmt.Sprint(v...)) }
func (l *botgoLogger) Warn(v ...interface{})                  { l.Logger.Warn(fmt.Sprint(v...)) }
func (l *botgoLogger) Error(v ...interface{})                 { l.Logger.Error(fmt.Sprint(v...)) }
func (l *botgoLogger) Debugf(format string, v ...interface{}) { l.Logger.Debug(fmt.Sprintf(format, v...)) }
func (l *botgoLogger) Infof(format string, v ...interface{})  { l.Logger.Info(fmt.Sprintf(format, v...)) }
func (l *botgoLogger) Warnf(format string, v ...interface{})  { l.Logger.Warn(fmt.Sprintf(format, v...)) }
func (l *botgoLogger) Errorf(format string, v ...interface{}) { l.Logger.Error(fmt.Sprintf(format, v...)) }
func (l *botgoLogger) Sync() error                            { return nil }

// ---
// Helpers
// ---

func ensureDirs(dirs ...string) {
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[bot] failed to create directory %s: %v\n", d, err)
		}
	}
}
