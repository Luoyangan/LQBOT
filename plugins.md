# plugins/ — 业务指令模块

`plugins/` 下的每个子目录是一个独立的 Go 包，编译时静态链接进主程序。

## 快速创建一个新插件

```go
// plugins/mycmd/mycmd.go
package mycmd

import "github.com/Luoyangan/LQBOT/internal/contract"

func Register(r contract.CommandRegister) {
    r.Register(contract.Command{
        Name:        "hello",
        Aliases:     []string{"hi"},
        Description: "打个招呼",
        Handler: func(ctx contract.CommandContext) error {
            return ctx.Reply("Hello!")
        },
    })
}
```

然后在 `internal/bot/bot.go` 的 `registerPlugins()` 中添加一行：

```go
func (b *Bot) registerPlugins() {
    // ... 已有插件的 Register 调用
    mycmd.Register(b.router)          // <-- 新增
}
```

## 注册模式

每个插件包导出一个 `Register()` 函数，只接收自身需要的窄接口：

```go
// 只注册指令
func Register(r contract.CommandRegister)

// 指令 + 事件监听
func Register(r contract.CommandRegister, l contract.ListenerRegister)

// 指令 + 事件 + QQ API
func Register(r contract.CommandRegister, l contract.ListenerRegister, api contract.QQAPI)
```

### 可用窄接口

| 接口 | 方法 | 用途 |
|------|------|------|
| `contract.CommandRegister` | `Register(cmd Command)` | 注册一个指令 |
| `contract.ListenerRegister` | `Subscribe(listener Listener)` | 注册事件监听器 |
| `contract.QQAPI` | SendMessage / SendMarkdown / ReplyMarkdown / ReplyImage / SendImage / SendEmbedMessage / SendArkMessage / SendRichMedia / SendMessageWithButtons / SendGroupMessage / SendGroupMarkdown / SendGroupRichMedia / SendGroupMessageWithButtons / SendC2CMessage / SendC2CMarkdown / SendC2CRichMedia / SendC2CMessageWithButtons / DeleteMessage / DeleteGroupMessage / DeleteC2CMessage / PinMessage / UnpinMessage / CreateReaction / DeleteReaction / PutInteraction / ReplyInteraction / SendActiveC2CMessage / GetGuild / GetChannel / GetGuildMember / UploadChannelMedia | 调用 QQ API |

> 不需要的参数不要在 `Register()` 中声明，框架不会传入 nil。

## 指令定义

```go
contract.Command{
    Name:        "echo",            // 指令名，用户通过 /echo 触发
    Aliases:     []string{"e"},     // 别名，/e 也可触发
    Description: "重复输入",         // 帮助说明
    Usage:       "echo <消息>",     // 用法示例（可选）
    Permission:  "admin",           // 权限节点（可选，由中间件检查）
    Handler: func(ctx contract.CommandContext) error {
        // ctx.Args()  → []string{"消息内容"}
        // ctx.Reply() → 快速回复
        return nil
    },
}
```

### 指令触发方式

| 场景 | 触发 |
|------|------|
| `/command` | 标准指令前缀 |
| `command` | 不带 / 前缀（仅非混淆时） |
| 群聊 | 发送消息即触发（无需 @，需订阅 `types.EventGroupMessageCreate`） |

### CommandContext 接口

```go
type CommandContext interface {
    Args() []string         // 按空格分割的参数列表
    Arg(i int) string       // 获取第 i 个参数
    ArgCount() int          // 参数个数
    Reply(msg string) error // 快速回复文本消息
    EventContext            // 嵌入消息上下文
}
```

## 事件监听

```go
contract.Listener{
    Event:   types.EventMessageCreate, // 事件类型（QQ API 原生格式）
    Handler: func(ctx contract.EventContext) error {
        // 处理事件
        return nil
    },
}
```

### 常用事件类型

| 常量 | QQ 原生 | 触发时机 |
|------|---------|---------|
| `types.EventMessageCreate` | `MESSAGE_CREATE` | 频道普通消息 |
| `types.EventAtMessageCreate` | `AT_MESSAGE_CREATE` | 频道 @机器人 消息 |
| `types.EventGroupAtMessageCreate` | `GROUP_AT_MESSAGE_CREATE` | 群聊 @机器人 消息 |
| `types.EventGroupMessageCreate` | `GROUP_MESSAGE_CREATE` | 群聊消息（含 @ 和不含 @） |
| `types.EventC2CMessageCreate` | `C2C_MESSAGE_CREATE` | 私聊消息 |
| `types.EventInteractionCreate` | `INTERACTION_CREATE` | 按钮/选择框交互 |

### EventContext 接口

```go
type EventContext interface {
    Content() string           // 消息文本（已去除 @机器人 前缀）
    RawContent() string        // 原始消息内容
    ChannelID() string         // 频道/子频道 ID（群聊场景为空）
    AuthorID() string          // 发送者 ID
    MessageID() string         // 消息 ID
    IsMentioned() bool         // 是否被 @
    GuildID() string           // 频道 ID（群聊/私聊为空）
    GroupID() string           // 群聊 ID（频道/私聊为空）
    Mentions() []string        // 被 @ 的用户 ID 列表
    Attachments() []Attachment // 附件信息
    Scene() MessageScene       // 来源场景：Guild / Group / C2C
    Reply(msg string) error    // 快速回复（自动选择频道/群聊/私聊 API）
    ReplyMarkdown(content string) error  // Markdown 被动回复（带 msg_id）
    ReplyImage(url string) error         // 图片被动回复（带 msg_id）
    ReplyWithButtons(content string, buttons []MessageButton) error
    ReplyWithButtonRows(content string, rows [][]MessageButton) error
    ReplyArk(ark *MessageArk) error               // Ark 被动回复
    ReplyMarkdownTemplate(templateID string, params []MarkdownParam) error // 模板 Markdown 被动回复
}
```

## 使用 QQ API

需要调用 SendMessage、SendMarkdown 等 API 时，在 `Register()` 中接收 `contract.QQAPI` 参数：

```go
func Register(r contract.CommandRegister, api contract.QQAPI) {
    r.Register(contract.Command{
        Name: "info",
        Handler: func(ctx contract.CommandContext) error {
            // 根据场景选择正确的发送 API
            if ctx.Scene() == contract.SceneGroup {
                return api.SendGroupMessage(ctx.GroupID(), "Hello Group!")
            }
            return api.SendMessage(ctx.ChannelID(), "Hello Channel!")
        },
    })
}
```

### 文本消息

```go
// 频道文本消息
api.SendMessage(channelID, "这是一条文本消息")

// 群聊文本消息
api.SendGroupMessage(groupOpenID, "群聊文本消息")

// C2C 私聊文本消息
api.SendC2CMessage(userOpenID, "私聊文本消息")
```

### Markdown 消息

两种模式：自定义 Markdown 和模板 Markdown。

> **公域机器人注意**：频道 Markdown 需内邀开通（不支持原生 markdown 时建议用 Embed 或 Ark 替代）。
> 频道场景使用 `ReplyMarkdown` 替代 `SendMarkdown` 可绕过主动推送限制。

```go
// ── 自定义 Markdown（直接写 markdown 内容）──
// 频道（推荐用 ReplyMarkdown 带 msg_id 被动回复）
api.ReplyMarkdown(channelID, msgID, "# 标题\n**加粗**\n1. 列表1\n2. 列表2")

// 群聊
api.SendGroupMarkdown(groupID, "# 通知\n大家好，这里是 **LQBOT**")

// C2C
api.SendC2CMarkdown(userID, "你好 **用户**")


// ── 模板 Markdown（使用管理端注册的模板）──
// 需先在管理端注册 custom_template_id
params := []contract.MarkdownParam{
    {Key: "title", Values: []string{"标题"}},
    {Key: "desc",  Values: []string{"简介"}},
}

// 频道
api.SendMarkdownTemplate(channelID, "YOUR_TEMPLATE_ID", params)

// 群聊
api.SendGroupMarkdownTemplate(groupID, "YOUR_TEMPLATE_ID", params)

// C2C
api.SendC2CMarkdownTemplate(userID, "YOUR_TEMPLATE_ID", params)
```

### 图片消息

> **公域机器人注意**：频道图片用 `ReplyImage` 替代 `SendImage` 可绕过主动推送时间限制（00:00-06:00 禁止主动推送）。

```go
// 频道图片（URL 需在管理端报备，推荐用 ReplyImage 带 msg_id）
api.ReplyImage(channelID, msgID, "https://example.com/image.png")

// 群聊图片（自动两步流程：上传 → 带 msg_id 被动回复发送）
api.SendGroupRichMedia(groupID, &contract.RichMedia{
    FileType: 1,   // 1=image
    URL:      "https://example.com/photo.jpg",
    MsgID:    msgID,
})

// C2C 图片（同上两步流程）
api.SendC2CRichMedia(userID, &contract.RichMedia{
    FileType: 1,
    URL:      "https://example.com/logo.png",
    MsgID:    msgID,
})
```

### Embed 消息（富卡片，仅频道）

```go
api.SendEmbedMessage(channelID, &contract.MessageEmbed{
    Title:   "等级提升",
    Prompt:  "你升级了！",      // 通知栏提示
    Description: "恭喜你达到黄金段位",
    Thumbnail: "https://example.com/badge.png",
    Fields: []contract.EmbedField{
        {Name: "当前等级：黄金"},
        {Name: "之前等级：白银"},
    },
})
```

### Ark 模板消息

Ark 消息使用预注册模板，通过 KV 填充变量。**群聊/C2C 场景可能需要先在管理端注册模板 ID**（部分内置模板如 23 未注册时可能返回 304004）。

```go
// 频道 Ark（template_id 23: 链接+文本列表模板）
api.SendArkMessage(channelID, &contract.MessageArk{
    TemplateID: 23,
    KV: []contract.ArkKV{
    {Key: "#DESC#", Value: "机器人订阅消息"},
    {Key: "#PROMPT#", Value: "LQBOT 机器人"},
    {
        Key: "#LIST#",
        Obj: []contract.ArkObj{
            {ObjKV: []contract.ArkObjKV{
                {Key: "desc", Value: "需求标题：UI问题解决"},
            }},
            {ObjKV: []contract.ArkObjKV{
                {Key: "desc", Value: "当前状态：体验中"},
            }},
            {ObjKV: []contract.ArkObjKV{
                {Key: "desc", Value: "已评审"},
                {Key: "link", Value: "https://qun.qq.com"},
            }},
            {ObjKV: []contract.ArkObjKV{
                {Key: "desc", Value: "已排期"},
                {Key: "link", Value: "https://qun.qq.com"},
            }},
            {ObjKV: []contract.ArkObjKV{
                {Key: "desc", Value: "开发中"},
                {Key: "link", Value: "https://qun.qq.com"},
            }},
        },
    },
},
})

// 群聊 Ark（同样支持 template_id 23/24/37，但群聊/C2C 可能需要注册模板）
api.SendGroupArkMessage(groupID, &contract.MessageArk{TemplateID: 23, KV: [...]})

// C2C Ark
api.SendC2CArkMessage(userID, &contract.MessageArk{TemplateID: 23, KV: [...]})
```

> 模板 23/24/37 为 QQ 内置的默认模板，但群聊/C2C 场景如果返回 `304004 (no permission to use this ark template)`，需在管理端注册模板后使用。频道场景通常无需注册。

### 富媒体消息（视频/语音/文件）

> **公域机器人注意**：所有群聊/C2C 富媒体发送均务必设置 `MsgID`，框架内部自动执行两步流程（上传 → 带 `msg_id` 被动回复），绕过主动推送限制。

```go
// 群聊发送视频
api.SendGroupRichMedia(groupID, &contract.RichMedia{
    FileType: 2,          // 2=video
    URL: "https://example.com/video.mp4",
    Content: "看这个视频",
    MsgID:   ctx.MessageID(),
})

// C2C 发送语音
api.SendC2CRichMedia(userID, &contract.RichMedia{
    FileType: 3,          // 3=voice
    URL: "https://example.com/audio.mp3",
    MsgID:   ctx.MessageID(),
})

// C2C 发送文件
api.SendC2CRichMedia(userID, &contract.RichMedia{
    FileType: 4,          // 4=file
    URL: "https://example.com/doc.pdf",
    Content: "请看这份文档",
    MsgID:   ctx.MessageID(),
})
```

### 按钮消息

```go
buttons := []contract.MessageButton{
    {ID: "btn_yes", Label: "同意", Style: 1, Data: "yes", ActionType: 1},
    {ID: "btn_no",  Label: "拒绝", Style: 0, Data: "no", ActionType: 1},
}

// 频道按钮
api.SendMessageWithButtons(channelID, "请选择：", buttons)

// 群聊按钮
api.SendGroupMessageWithButtons(groupID, "请确认：", buttons)

// C2C 按钮
api.SendC2CMessageWithButtons(userID, "是否继续？", buttons)
```

### 多行按钮消息

使用 `ReplyWithButtonRows` 可按行组织按钮布局：

```go
rows := [][]contract.MessageButton{
    // 第 1 行：1 个主按钮
    {{ID: "btn_hello", Label: "你好", Style: 1, ActionType: 1}},
    // 第 2 行：2 个次要按钮
    {
        {ID: "btn_ping", Label: "Ping", Style: 0, ActionType: 1},
        {ID: "btn_info", Label: "信息", Style: 0, ActionType: 1},
    },
    // 第 3 行：3 个混合样式按钮
    {
        {ID: "btn_a", Label: "按钮A", Style: 1, ActionType: 1},
        {ID: "btn_b", Label: "按钮B", Style: 2, ActionType: 1},
        {ID: "btn_c", Label: "按钮C", Style: 3, ActionType: 1},
    },
}
contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
return ctx.ReplyWithButtonRows("选择操作：", rows)
```

### 消息管理

```go
// 撤回消息（有权限要求）
api.DeleteMessage(channelID, messageID)        // 频道
api.DeleteGroupMessage(groupID, messageID)     // 群聊
api.DeleteC2CMessage(userID, messageID)        // C2C

// 精华消息（仅频道）
api.PinMessage(channelID, messageID)           // 置顶
api.UnpinMessage(channelID, messageID)         // 取消置顶
```

### 表情表态（仅频道）

```go
// 系统表情（QQ 内置）：type=1, id=表情编号
api.CreateReaction(channelID, messageID, "1:4")     // 添加
api.DeleteReaction(channelID, messageID, "1:4")     // 删除

// Unicode 表情：type=2, id=表情文本
api.CreateReaction(channelID, messageID, "2:❤️")   // 添加❤️
api.CreateReaction(channelID, messageID, "2:😄")   // 添加😄
```

如需测试，可在频道中使用 `/react <表情ID>` 指令：
```
/react 2:❤️     → 给当前消息添加❤️
/react 1:4      → 给当前消息添加第4号系统表情
```

### 被动回复（推荐）

公域机器人应尽量使用被动回复代替主动推送，以绕过主动消息的各种限制。

**被动回复 = 在 API 请求中携带 `msg_id` 参数**，表示这条消息是对用户某条消息的回复。

框架自动处理被动回复的场景：

| 方法 | 说明 |
|------|------|
| `ctx.Reply(msg)` | EventContext/CommandContext 的快速回复，内部自动带 `msg_id` |
| `api.ReplyMarkdown(channelID, msgID, content)` | 频道 Markdown 被动回复 |
| `api.ReplyImage(channelID, msgID, imageURL)` | 频道图片被动回复 |
| `api.SendGroupRichMedia(groupID, media)` | 设置 `media.MsgID` 后走被动回复通道 |
| `api.SendC2CRichMedia(userID, media)` | 同上 |

```go
// 所有 Reply* 方法自动携带 msg_id（框架内部处理）
ctx.Reply("这是一条被动回复")                      // 自动带 msg_id
api.ReplyMarkdown(channelID, msgID, "**Markdown**") // 频道 Markdown 被动回复
api.ReplyImage(channelID, msgID, imageURL)           // 频道图片被动回复

// 富媒体消息需手动设置 MsgID
api.SendGroupRichMedia(groupID, &contract.RichMedia{
    FileType: 1,
    URL:      imageURL,
    MsgID:    msgID,  // 显式设置
})
```

> **优势**：被动回复不受主动消息频率限制（频道 00:00-06:00 禁止、群聊每月 20 条、C2C 每月 4 条）。

### 主动消息推送

主动推送是指机器人在用户没有发送任何消息的情况下，主动向用户发送消息。

代码中调用方式：

```go
// C2C 互动召回（30天内最多4次，用户需曾在30天内互动过）
api.SendActiveC2CMessage(userID, "好久不见！", true)

// 普通 C2C 主动消息（每月最多4条/用户）
api.SendActiveC2CMessage(userID, "这是一条主动消息", false)
```

> **频控说明**：C2C 每月 4 条/用户，群聊每月 4 条/群，频道每天 20 条/子频道。
> **建议**：尽量使用被动回复（带 `msg_id`）替代主动推送。

### @ 提及用户

在 Markdown 消息中嵌入 `<qqbot-at-user>` 标签即可 @ 指定用户：

```go
// @ 特定用户（Markdown 消息）
msg := "# 通知\n" + contract.MentionUser(userID) + " 你好！"
api.SendMarkdown(channelID, msg)        // 频道
api.SendGroupMarkdown(groupID, msg)     // 群聊
api.SendC2CMarkdown(userID, msg)        // C2C

// @ 消息发送者自己（在命令中）
msg := "# @你\n" + contract.MentionUser(ctx.AuthorID()) + " 你被提到了！"
api.SendGroupMarkdown(ctx.GroupID(), msg)
```

> **注意**：`<qqbot-at-user>` 仅支持 Markdown 消息（msg_type=2）。旧格式 `<@!userID>` 已弃用，后续会被移除。

### 文本链交互元素

文本链交互元素允许在消息中嵌入可点击的操作标签，需在 Markdown 消息中使用。

| 函数 | 生成格式 | 场景支持 | 说明 |
|------|---------|----------|------|
| `MentionUser(id)` | `<qqbot-at-user id="..." />` | 频道/群聊/C2C | @ 指定用户 |
| `MentionEveryone()` | `<qqbot-at-everyone />` | 频道/群聊/C2C | @ 全体成员 |
| `CmdEnter(text)` | `<qqbot-cmd-enter text="..." />` | 仅 C2C Markdown | 点击直接发送文本 |
| `CmdInput(text, show, reference)` | `<qqbot-cmd-input ... />` | 三场景 Markdown | 点击插入输入框 |
| `ChannelLink(id)` | `<#channelID>` | 仅频道 | 跳转子频道链接 |
| `Emoji(id)` | `<emoji:id>` | 频道/群聊/C2C | 系统表情 |

```go
// 回车指令（点击直接发送文本）— 仅 C2C Markdown
contract.CmdEnter("/ping")

// 参数指令（点击插入输入框）— 所有场景 Markdown
contract.CmdInput("/echo 你好", "点我输入", false)      // 无引用
contract.CmdInput("/echo 收到", "带引用回复", true)     // 带引用

// 跳转子频道链接（仅频道）
contract.ChannelLink("123456")

// 系统表情
contract.Emoji("4")  // 显示 😊
```

> **CmdEnter**：文档明确"群聊和文字子频道不支持该能力"，仅 C2C 场景。
> **CmdInput**：无此限制，所有场景均支持。

### 获取信息

```go
// 获取频道（Guild）信息
guild, err := api.GetGuild(guildID)
// -> &Guild{Name: "游戏交流群", MemberCount: 1000}

// 获取子频道信息
ch, err := api.GetChannel(channelID)
// -> &Channel{Name: "闲聊区", Type: 0}  // 0=文字频道

// 获取成员信息
member, err := api.GetGuildMember(guildID, userID)
// -> &Member{Nick: "张三", Roles: ["admin"]}
```

### QQAPI 方法一览

```go
type QQAPI interface {
    // === Channel message sending (文字子频道) ===
    SendMessage(channelID, content string) error
    SendMarkdown(channelID, content string) error
    ReplyMarkdown(channelID, msgID, content string) error   // 被动回复 Markdown（带 msg_id）
    ReplyImage(channelID, msgID, imageURL string) error     // 被动回复图片（带 msg_id）
    SendMarkdownTemplate(channelID, templateID string, params []MarkdownParam) error
    SendImage(channelID, imageURL string) error
    SendEmbedMessage(channelID string, embed *MessageEmbed) error
    SendArkMessage(channelID string, ark *MessageArk) error
    SendRichMedia(channelID string, media *RichMedia) error
    SendMessageWithButtons(channelID string, content string, buttons []MessageButton) error

    // === Group message sending (群聊) ===
    SendGroupMessage(groupID, content string) error
    SendGroupMarkdown(groupID string, content string) error
    SendGroupMarkdownTemplate(groupID, templateID string, params []MarkdownParam) error
    SendGroupRichMedia(groupID string, media *RichMedia) error
    SendGroupMessageWithButtons(groupID string, content string, buttons []MessageButton) error
    SendGroupArkMessage(groupID string, ark *MessageArk) error

    // === C2C / DM message sending (私聊) ===
    SendC2CMessage(userID, content string) error
    SendC2CMarkdown(userID, content string) error
    SendC2CMarkdownTemplate(userID, templateID string, params []MarkdownParam) error
    SendC2CRichMedia(userID string, media *RichMedia) error
    SendC2CMessageWithButtons(userID string, content string, buttons []MessageButton) error
    SendC2CArkMessage(userID string, ark *MessageArk) error

    // === Message management ===
    DeleteMessage(channelID, messageID string) error
    DeleteGroupMessage(groupID, messageID string) error
    DeleteC2CMessage(userID, messageID string) error
    PinMessage(channelID, messageID string) error
    UnpinMessage(channelID, messageID string) error

    // === Reactions ===
    CreateReaction(channelID, messageID, emoji string) error
    DeleteReaction(channelID, messageID, emoji string) error

    // === Interaction callback ===
    PutInteraction(interactionID string, body string) error
    ReplyInteraction(interactionID string, content string) error

    // === Active push (主动推送) ===
    SendActiveC2CMessage(userID, content string, isWakeup bool) error

    // === Guild / Channel info ===
    GetGuild(guildID string) (*Guild, error)
    GetChannel(channelID string) (*Channel, error)
    GetGuildMember(guildID, userID string) (*Member, error)

    // === Channel media upload ===
    UploadChannelMedia(channelID string, fileType int, url string) (string, error)
}
```

### 支持的消息类型

| 类型 | 常量 | msg_type | 场景支持 | 说明 |
|------|------|----------|----------|------|
| 文本 | `TextMsg` | 0 | 频道/群/C2C | 普通文本消息 |
| Markdown | `MarkdownMsg` | 2 | 频道/群/C2C | 支持 markdown 格式 |
| Ark | `ArkMsg` | 3 | 频道/群/C2C | 模板消息，需先在管理端注册模板 |
| Embed | `EmbedMsg` | 4 | 频道/频道私信 | 富卡片消息（不支持群聊/C2C） |
| 富媒体 | `RichMediaMsg` | 7 | 频道/群/C2C | 图片/视频/语音/文件 |

### MessageEmbed 结构体 (msg_type=4)

仅支持文字子频道和频道私信场景。

```go
type MessageEmbed struct {
    Title       string       // 卡片标题
    Prompt      string       // 通知栏提示文本
    Description string       // 卡片描述
    Thumbnail   string       // 缩略图 URL
    Fields      []EmbedField // 文本字段（最多4个）
}

type EmbedField struct {
    Name string // 字段文本内容
}
```

示例：
```go
api.SendEmbedMessage(channelID, &contract.MessageEmbed{
    Title:   "等级提升",
    Prompt:  "消息通知",
    Fields:  []contract.EmbedField{
        {Name: "当前等级：黄金"},
        {Name: "之前等级：白银"},
    },
})
```

### MessageArk 结构体 (msg_type=3)

Ark 消息使用预注册的模板 ID，通过 KV 填充模板变量。

| 模板 ID | 名称 | 说明 | 可用键 |
|---------|------|------|--------|
| 23 | 链接+文本列表 | 描述 + 提示 + 列表项（内置默认） | `#DESC#`(文本) `#PROMPT#`(提示) `#LIST#`(数组: desc + link) |
| 24 | 文本+缩略图 | 标题 + 描述 + 图片（内置默认） | `#TITLE#`(标题) `#DESC#`(描述) `#META_URL#`(图片) `#PROMPT#`(提示) |
| 37 | 大图模板 | 大图展示（内置默认） | `#TITLE#`(标题) `#DESC#`(描述) `#META_URL#`(图片) `#PROMPT#`(提示) |

发送方法支持全场景：

| 场景 | API |
|------|-----|
| 文字子频道 | `SendArkMessage(channelID, ark)` |
| 群聊 | `SendGroupArkMessage(groupID, ark)` |
| C2C 私聊 | `SendC2CArkMessage(userID, ark)` |

```go
type MessageArk struct {
    TemplateID int      // Ark 模板 ID（需在管理端注册）
    KV         []ArkKV  // 模板参数
}

type ArkKV struct {
    Key   string   // 参数 Key
    Value string   // 参数 Value（与 Obj 二选一）
    Obj   []ArkObj // 嵌套对象（与 Value 二选一）
}

type ArkObj struct {
    ObjKV []ArkObjKV
}

type ArkObjKV struct {
    Key   string
    Value string
}
```

### RichMedia 结构体 (msg_type=7)

统一发送图片、视频、语音、文件。URL 需先在管理端报备。

```go
type RichMedia struct {
    FileType   int    // 1=image, 2=video, 3=voice, 4=file
    URL        string // 文件 URL（群聊/C2C 必填）
    Content    string // 可选文本描述
    SrvSendMsg bool   // 控制是否直接发送（框架内部使用，插件无需关心）
    FileInfo   string // 频道场景：从 UploadChannelMedia 获取的 file_info（字符串，非 []byte）
    MsgID      string // 原消息 ID，用于被动回复（公域机器人必须设置以绕过主动推送限制）
}
```

> **频道场景**：先调用 `UploadChannelMedia` 上传获取 `file_info`，再传入 `FileInfo` 字段。
> **群聊/C2C 场景**：直接提供 URL 和 FileType，`SendGroupRichMedia`/`SendC2CRichMedia` 内部自动执行上传 → 发送两步流程。
> **公域机器人注意**：务必设置 `MsgID` 字段为原始消息 ID，使发送走被动回复通道，绕过主动推送的频率/时间限制。
> **FileInfo 类型**：为 `string` 而非 `[]byte`，避免 botgo SDK 的 JSON 序列化自动进行 base64 编码导致 `invalid file_info`。

```go
// 群聊发送图片（自动两步流程：上传 → 带 msg_id 被动回复）
api.SendGroupRichMedia(groupID, &contract.RichMedia{
    FileType: 1,
    URL:      "https://example.com/image.png",
    Content:  "看这张图片",
    MsgID:    ctx.MessageID(),  // 公域机器人必填
})

// C2C 发送视频（同上）
api.SendC2CRichMedia(userID, &contract.RichMedia{
    FileType: 2,
    URL:      "https://example.com/video.mp4",
    MsgID:    ctx.MessageID(),  // 公域机器人必填
})
```

## 按钮交互

按钮消息支持三种 ActionType：

| ActionType | 值 | 行为 | Data/URL 用法 |
|-----------|----|------|-------------|
| 跳转 | `0` | 点击打开 URL | **使用 `URL` 字段**（因 Go int 零值 0 与跳转冲突） |
| 回调 | `1` | 点击触发 `INTERACTION_CREATE` 事件 | Data = 回调数据 |
| 指令 | `2` | 点击在输入框插入 `@bot Data` | Data = 指令文本（不含 @bot） |

> **ActionType 默认值**：`2`（指令）。不设 `ActionType` 的按钮默认为指令按钮。
> **跳转按钮务必使用 `URL` 字段**，而非 `ActionType: 0 + Data`。

### 示例：多行按钮（推荐使用 `ReplyWithButtonRows`）

```go
r.Register(contract.Command{
    Name: "buttons",
    Handler: func(ctx contract.CommandContext) error {
        isC2C := ctx.Scene() == contract.SceneC2C
        rows := [][]contract.MessageButton{
            // 第 1 行：1 个跳转按钮
            {
                {ID: "btn_github", Label: "GitHub", Style: 4,
                    URL: "https://github.com",
                    UnsupportTips: "请使用最新版手机QQ"},
            },
            // 第 2 行：2 个指令按钮（Enter 仅 C2C）
            {
                {ID: "btn_ping", Label: "Ping", Style: 0,
                    Data: "/ping", ActionType: 2,
                    Enter: isC2C, UnsupportTips: "请升级客户端"},
                {ID: "btn_info", Label: "Info", Style: 0,
                    Data: "/info", ActionType: 2,
                    UnsupportTips: "请升级客户端"},
            },
            // 第 3 行：3 个回调按钮
            {
                {ID: "btn_hello", Label: "你好", Style: 1,
                    Data: "btn_hello", ActionType: 1},
                {ID: "btn_ping", Label: "Ping", Style: 2,
                    Data: "btn_ping", ActionType: 1},
                {ID: "btn_info", Label: "信息", Style: 3,
                    Data: "btn_info", ActionType: 1},
            },
        }
        // 存储原始消息 msg_id 供按钮回调使用
        contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
        return ctx.ReplyWithButtonRows("请选择操作：", rows)
    },
})
```

### MessageButton 字段

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| ID | string | — | 按钮唯一标识 |
| Label | string | — | 按钮文字 |
| Style | int | `0` | 0=灰底, 1=蓝底, 2=灰白, 3=红字白底, 4=蓝底白字 |
| Data | string | `ID` | Action 数据（回调 data/指令文本） |
| URL | string | — | **跳转 URL。** 设置后按钮变为跳转类型（ActionType=0），推荐替代 `ActionType: 0` |
| ActionType | int | `2`(指令) | 0=跳转(用URL字段), 1=回调, 2=指令 |
| Permission | int | `2` | 0=指定用户, 1=管理员, 2=所有人, 3=指定身份组(仅频道) |
| SpecifyUserIDs | []string | — | Permission=0 时指定可操作用户 |
| SpecifyRoleIDs | []string | — | Permission=3 时指定身份组 ID |
| Enter | bool | `false` | 指令按钮自动发送（**仅 C2C 有效**） |
| Reply | bool | `false` | 指令按钮带引用回复本消息 |
| Anchor | int | `0` | 1=唤起选图器（仅手机端 8983+ C2C） |
| UnsupportTips | string | — | 客户端不支持此 Action 时的 toast 文案 |

> **场景注意事项**：
> - Go `int` 零值 `0` 与 QQ API `ActionType=0`（跳转）冲突。**跳转按钮务必使用 `URL` 字段**
> - `Permission` 的 Go 零值为 `0`（指定用户），框架自动转为 `2`（所有人）
> - `Enter=true` 仅在 **C2C** 有效，群聊/频道中使用会导致按钮显示"无权限操作"
> - `Anchor` 仅手机端 QQ 8983+ 的 C2C 场景有效，桌面端不支持
> - 指令按钮在群聊中点击后**在输入框插入 `@bot /ping`**，用户手动发送
> - 指令按钮在 C2C 中可设置 `Enter=true` 点击后自动发送
> - 按钮消息内部使用 Markdown 消息类型发送（`MsgType: 2`）

### 按钮回调处理

```go
l.Subscribe(contract.Listener{
    Event: types.EventInteractionCreate,
    Handler: func(ctx contract.EventContext) error {
        ic := ctx.(contract.InteractionContext)
        _ = ic.DeferReply()  // 先 ACK 交互，防止超时

        switch ic.ButtonID() {
        case "btn_hello":
            return ic.Reply("你好！我是 LQBOT，很高兴认识你")
        case "btn_ping":
            return ic.Reply("Pong! 机器人运行正常 ✓")
        case "btn_info":
            return ic.Reply("LQBOT v1.0 | Go 语言 | QQ Bot API v2")
        default:
            return ic.Reply(fmt.Sprintf("按钮已点击（ID: %s）", ic.ButtonID()))
        }
    },
})
```

**被动回复关键**：回调中使用 `Reply()` 需携带原始消息的 `msg_id` 才能发送全群可见消息。发送按钮消息时调用 `contract.StoreButtonMsgID(groupID, msgID)` 存储，框架自动从 `ButtonReplyStore` 获取。

**MsgSeq 防去重**：框架使用 `contract.NextMsgSeq(groupID)` 生成递增序号，避免同一 `msg_id` 被多次回复时被 QQ 服务端判定为重复消息（`40054005`）。

```go
// 示例：手动设置 msg_seq 防去重（框架在 ReplyWithButtons/ReplyWithButtonRows 中自动处理）
msgSeq := contract.NextMsgSeq(ctx.GroupID())
api.SendGroupMessageWithButtons(ctx.GroupID(), "请确认：", buttons, msgSeq)
```

### InteractionData 结构体

```go
type InteractionData struct {
    ID          string `json:"id"`
    Type        int    `json:"type"`
    ButtonID    string `json:"button_id"`
    ButtonData  string `json:"button_data"`
    ChannelID   string `json:"channel_id"`
    GuildID     string `json:"guild_id"`
    GroupOpenID string `json:"group_openid"`
    UserID      string `json:"user_id"`
    UserOpenID  string `json:"user_openid"`  // C2C 用户 OpenID
    MessageID   string `json:"message_id"`
    Scene       string `json:"scene"`         // "guild" / "group" / "c2c"
}
```

### InteractionContext 接口

```go
type InteractionContext interface {
    InteractionData() *InteractionData
    ButtonID() string
    ButtonData() string
    UserID() string
    ChannelID() string
    GroupOpenID() string
    UserOpenID() string           // C2C 用户 OpenID
    MessageID() string
    Reply(msg string) error       // 发送文本消息（自动路由到频道/群聊/C2C）
    Callback(content string) error
    DeferReply() error            // 仅确认交互不回复（用于耗时处理）
}
```

### Callback 行为

| 场景 | 行为 |
|------|------|
| **群聊** | 先调用 `PutInteraction({"code":0})` 消除 loading，再发送消息到群 |
| **频道** | 直接调用 `ReplyInteraction` 通过交互回调回复 |
| **私聊** | 同上，通过交互回调直接回复 |

## 现有插件参考

### ping — 在线测试

`/ping` 或 `/p` → 回复 `Pong! (场景: xx, 发送者: xxx)`

完整源码：

```go
package ping

import (
	"fmt"
	"github.com/Luoyangan/LQBOT/internal/contract"
)

func Register(r contract.CommandRegister) {
	r.Register(contract.Command{
		Name:        "ping",
		Aliases:     []string{"p"},
		Description: "测试机器人是否在线",
		Usage:       "ping",
		Handler: func(ctx contract.CommandContext) error {
			reply := fmt.Sprintf("Pong! (场景: %s, 发送者: %s)",
				sceneLabel(ctx.Scene()), ctx.AuthorID())
			return ctx.Reply(reply)
		},
	})
}

func sceneLabel(s contract.MessageScene) string {
	switch s {
	case contract.SceneGuild: return "频道"
	case contract.SceneGroup:  return "群聊"
	case contract.SceneC2C:    return "私聊"
	default: return "未知"
	}
}
```

**演示特性**：
- 仅依赖 `contract.CommandRegister`（最简依赖）
- `Scene()` 场景识别：区分频道/群聊/私聊
- `AuthorID()` 获取发送者
- 命令别名（`/p` 同效）
- `Usage` 帮助说明

### echo — 回声

`/echo <消息>` → 回复相同的消息，超过 200 字提示长度限制。

完整源码：

```go
package echo

import (
	"fmt"
	"strings"
	"github.com/Luoyangan/LQBOT/internal/contract"
)

const maxEchoLength = 200

func Register(r contract.CommandRegister) {
	r.Register(contract.Command{
		Name:        "echo",
		Description: "重复你输入的消息",
		Usage:       "echo <消息内容>",
		Handler: func(ctx contract.CommandContext) error {
			if ctx.ArgCount() == 0 {
				return ctx.Reply("请提供要重复的消息。用法: /echo <消息内容>")
			}
			msg := strings.Join(ctx.Args(), " ")
			if len(msg) > maxEchoLength {
				return ctx.Reply(fmt.Sprintf("消息过长，请控制在 %d 字以内（当前 %d 字）",
					maxEchoLength, len(msg)))
			}
			return ctx.Reply(msg)
		},
	})
}
```

**演示特性**：
- 仅依赖 `contract.CommandRegister`（最简依赖）
- `Args()` / `ArgCount()` 参数解析
- 输入校验 + 错误提示
- 业务逻辑：长度限制

### hello — 问候 + 按钮演示

- `你好` 消息触发 → 回复 `你好！我是 LQBOT`（三场景：频道/群聊/C2C，三个独立 Subscribe）
- `/button` → 1 个回调按钮（ActionType=1）
- `/button2` → 3 个回调按钮（你好/Ping/信息）
- `/buttons` → 多行按钮布局（1行1个、2行2个、3行3个、跳转按钮行）
- `/buttonaction` → 演示跳转 URL / 指令 Enter / 回调 / 权限控制四种按钮
- `/at <消息>` → 在 Markdown 中 @ 提及发送者（使用 `ctx.ReplyMarkdown` 被动回复）
- `/textchain` → 按场景演示文本链元素

完整源码：

```go
package hello

import (
	"strings"

	"github.com/Luoyangan/LQBOT/internal/contract"
	"github.com/Luoyangan/LQBOT/internal/types"
)

func Register(r contract.CommandRegister, l contract.ListenerRegister, api contract.QQAPI) {
	// ── Event listener: responds to "你好" in all scenes ──
	l.Subscribe(contract.Listener{
		Event: types.EventMessageCreate,
		Handler: func(ctx contract.EventContext) error {
			if ctx.Content() == "你好" {
				return ctx.Reply("你好！我是 LQBOT")
			}
			return nil
		},
	})
	l.Subscribe(contract.Listener{
		Event: types.EventGroupMessageCreate,
		Handler: func(ctx contract.EventContext) error {
			if ctx.Content() == "你好" {
				return ctx.Reply("你好！我是 LQBOT")
			}
			return nil
		},
	})
	l.Subscribe(contract.Listener{
		Event: types.EventC2CMessageCreate,
		Handler: func(ctx contract.EventContext) error {
			if ctx.Content() == "你好" {
				return ctx.Reply("你好！我是 LQBOT")
			}
			return nil
		},
	})

	// ── /button: 1个回调按钮 ──
	r.Register(contract.Command{
		Name:        "button",
		Description: "发送一个按钮交互消息",
		Handler: func(ctx contract.CommandContext) error {
			buttons := []contract.MessageButton{
				{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},
			}
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtons("请点击按钮：", buttons)
		},
	})

	// ── /button2: 3个回调按钮 ──
	r.Register(contract.Command{
		Name:        "button2",
		Description: "演示不同样式和行为的多个按钮",
		Handler: func(ctx contract.CommandContext) error {
			buttons := []contract.MessageButton{
				{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},
				{ID: "btn_ping", Label: "Ping", Data: "btn_ping", Style: 0, ActionType: 1},
				{ID: "btn_info", Label: "机器人信息", Data: "btn_info", Style: 0, ActionType: 1},
			}
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtons("请点击按钮：", buttons)
		},
	})

	// ── /buttons: 多行按钮布局 ──
	r.Register(contract.Command{
		Name:        "buttons",
		Description: "演示多行按钮布局（1个/2个/3个按钮每行）",
		Handler: func(ctx contract.CommandContext) error {
			rows := [][]contract.MessageButton{
				// 第 1 行：1 个按钮（蓝色主按钮）
				{
					{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},
				},
				// 第 2 行：2 个按钮
				{
					{ID: "btn_ping", Label: "Ping", Data: "btn_ping", Style: 0, ActionType: 1},
					{ID: "btn_info", Label: "机器人信息", Data: "btn_info", Style: 4, ActionType: 1},
				},
				// 第 3 行：3 个按钮（混合样式）
				{
					{ID: "btn_hello", Label: "你好", Data: "btn_hello", Style: 1, ActionType: 1},
					{ID: "btn_ping", Label: "Ping", Data: "btn_ping", Style: 2, ActionType: 1},
					{ID: "btn_info", Label: "信息", Data: "btn_info", Style: 3, ActionType: 1},
				},
				// 第 4 行：跳转按钮（使用 URL 字段）
				{
					{ID: "btn_qun", Label: "加群", Style: 1,
						URL: "https://qm.qq.com/q/9Rvq6VylQA"},
					{ID: "btn_aaaaa", Label: "加入频道", Style: 0,
						URL: "https://pd.qq.com/s/7zeumh7of?b=9"},
				},
			}
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtonRows("多行按钮布局演示：", rows)
		},
	})

	// ── /buttonaction: 演示 ActionType 和权限 ──
	r.Register(contract.Command{
		Name:        "buttonaction",
		Description: "演示按钮的 action type（跳转/回调/指令）和权限控制",
		Handler: func(ctx contract.CommandContext) error {
			isC2C := ctx.Scene() == contract.SceneC2C
			rows := [][]contract.MessageButton{
				// 第 1 行：跳转按钮（使用 URL 字段）
				{
					{
						ID: "btn_github", Label: "GitHub", Style: 1,
						URL:           "https://github.com/Luoyangan/LQBOT",
						UnsupportTips: "请使用最新版手机QQ",
					},
				},
				// 第 2 行：指令按钮（ActionType=2, Enter 仅 C2C）
				{
					{
						ID: "btn_cmd_ping", Label: "发 Ping", Style: 3,
						Data: "/ping", ActionType: 2,
						Enter: isC2C, UnsupportTips: "请升级客户端",
					},
					{
						ID: "btn_cmd_info", Label: "发 info", Style: 4,
						Data: "/info", ActionType: 2,
						UnsupportTips: "请升级客户端",
					},
				},
				// 第 3 行：回调按钮 + 指令按钮(带引用回复)
				{
					{
						ID: "btn_hello3", Label: "你好(回调)", Style: 1,
						Data: "btn_hello", ActionType: 1,
					},
					{
						ID: "btn_cmd_reply", Label: "发 Ping(引用)", Style: 0,
						Data: "/ping", ActionType: 2, Reply: true,
						UnsupportTips: "请升级客户端",
					},
				},
				// 第 4 行：管理员权限按钮
				{
					{
						ID: "btn_hello3a", Label: "管理权限", Style: 0,
						Data: "你好", ActionType: 2,
						Enter: isC2C, Permission: 1,
					},
				},
			}
			contract.StoreButtonMsgID(ctx.GroupID(), ctx.MessageID())
			return ctx.ReplyWithButtonRows(
				"按钮 Action 类型演示：[跳转] [指令] [回调]",
				rows,
			)
		},
	})

	// ── 按钮交互回调 ──
	l.Subscribe(contract.Listener{
		Event: types.EventInteractionCreate,
		Handler: func(ctx contract.EventContext) error {
			ic := ctx.(contract.InteractionContext)
			_ = ic.DeferReply()
			switch ic.ButtonID() {
			case "btn_hello":
				return ic.Reply("你好！我是 LQBOT，很高兴认识你")
			case "btn_ping":
				return ic.Reply("Pong! 机器人运行正常 ✓")
			case "btn_info":
				return ic.Reply("LQBOT - 基于 Go 的 QQ 机器人\n技术栈: Go + botgo SDK + SQLite\n支持: 文本 / Markdown / Ark / 按钮交互")
			default:
				return ic.Reply("按钮已点击 (ID: " + ic.ButtonID() + ")")
			}
		},
	})

	// ── /at: 演示 @ 提及用户 ──
	r.Register(contract.Command{
		Name:        "at",
		Description: "艾特(提及)消息发送者演示",
		Usage:       "at <消息>",
		Handler: func(ctx contract.CommandContext) error {
			content := "你被提到了！"
			if ctx.ArgCount() > 0 {
				content = strings.Join(ctx.Args(), " ")
			}
			msg := contract.MentionUser(ctx.AuthorID()) + " " + content
			return ctx.ReplyMarkdown(msg)
		},
	})

	// ── /textchain: 演示文本链交互元素 ──
	r.Register(contract.Command{
		Name:        "textchain",
		Description: "演示文本链交互元素（Markdown）",
		Usage:       "textchain",
		Handler: func(ctx contract.CommandContext) error {
			switch ctx.Scene() {
			case contract.SceneGuild:
				msg := "# 文本链演示（频道）\n" +
					"1. @某人: " + contract.MentionUser(ctx.AuthorID()) + "\n" +
					"2. 参数指令: " + contract.CmdInput("/ping", "点我发 ping", false) + "\n" +
					"3. 跳转子频道: " + contract.ChannelLink(ctx.ChannelID()) + "\n" +
					"4. 表情: " + contract.Emoji("4") + " " + contract.Emoji("14") + "\n\n" +
					"回车指令仅 C2C，请用 /md interactive c2c"
				return ctx.ReplyMarkdown(msg)

			case contract.SceneGroup:
				msg := "# 文本链演示（群聊）\n" +
					"1. @某人: " + contract.MentionUser(ctx.AuthorID()) + "\n" +
					"2. 参数指令: " + contract.CmdInput("/ping", "点我发 ping", false)
				return ctx.ReplyMarkdown(msg)

			default:
				return ctx.Reply("C2C 场景请用: /md interactive c2c <用户ID>")
			}
		},
	})
}
```

**演示特性**：
- 依赖 `CommandRegister` + `ListenerRegister` + `QQAPI`（三接口）
- 事件监听器：三场景分别订阅（`MESSAGE_CREATE` / `GROUP_MESSAGE_CREATE` / `C2C_MESSAGE_CREATE`）
- 多行按钮（`ReplyWithButtonRows`）支持不同行数布局
- 三种 ActionType（跳转 URL / 回调 / 指令 Enter）完整演示
- Permission 权限控制（管理员权限按钮）
- 按钮交互处理：`DeferReply()` + `Reply()` 发送全群可见消息
- `StoreButtonMsgID` 存储 msg_id 供被动回复
- `/at` 和 `/textchain` 使用 `ctx.ReplyMarkdown()` 统一走被动回复通道

### embed — 富卡片消息

`/embed` → 发送一条富卡片消息（仅文字子频道场景）。

- 依赖 `CommandRegister` + `QQAPI`
- 演示：`SendEmbedMessage`、场景守卫（`SceneGuild` 检查）
- Embed 仅支持频道场景，群聊/C2C 会提示不可用

### ark — Ark 模板消息

`/ark channel` → 在当前频道发 Ark（template_id 23 模板，全字段填充示例）
`/ark group` → 在当前群聊发 Ark
`/ark c2c <用户ID>` → 给指定用户发 C2C Ark

- 依赖 `CommandRegister` + `QQAPI`
- 演示：`SendArkMessage`、`SendGroupArkMessage`、`SendC2CArkMessage` 三个场景
- 嵌套 ObjKV（`#LIST#` 数组变量）和简单 KV 混合使用
- **注意**：群聊/C2C 场景 Ark 模板可能需管理端注册，否则返回 `304004`

### info — 信息查询

`/server` → 显示当前频道/群聊信息
`/whoami` → 显示你的频道成员信息（仅频道）

- 依赖 `CommandRegister` + `QQAPI`
- 演示：`GetGuild`、`GetChannel`、`GetGuildMember`
- 场景处理：群聊场景下 `/server` 只返回群 ID

### media — 富媒体消息

`/image` → 发送示例图片
`/video` → 发送示例视频

- 依赖 `CommandRegister` + `QQAPI`
- 演示：`ReplyImage`（频道，被动回复带 msg_id）、`SendGroupRichMedia`（群聊）、`SendC2CRichMedia`（C2C）
- 群聊/C2C 场景使用两步流程：上传 → 携带 msg_id 被动回复发送
- 三场景自动路由（`switch ctx.Scene()`），频道用 `ReplyImage` 绕过主动推送时间限制
- 视频仅在群聊/C2C 支持，频道会提示不可用
- **注意**：URL 需先在管理端报备

### manage — 消息管理

`/delete <消息ID>` → 撤回消息（三场景均支持）
`/react <表情ID>` → 给消息添加表情反应（仅频道）
`/pin <消息ID>` → 设置精华消息（仅频道）
`/unpin <消息ID>` → 取消精华消息（仅频道）

- 依赖 `CommandRegister` + `QQAPI`
- 演示：`DeleteMessage`（三场景路由）、`CreateReaction`、`PinMessage`、`UnpinMessage`
- 表情格式：`1:4`（系统表情）或 `2:❤️`（Unicode 表情）

### markdown — Markdown 消息

`/md custom channel` → 在当前频道发自定义 Markdown
`/md custom group` → 在当前群聊发自定义 Markdown
`/md custom c2c <用户ID>` → 给用户发自定义 Markdown
`/md interactive channel/group/c2c` → 参数指令/回车指令 Markdown
`/md template channel/group/c2c` → 模板 Markdown
`/mda` → 发送 Markdown 代码消息

```go
r.Register(contract.Command{
    Name:        "mda",
    Description: "发送 Markdown 代码消息",
    Usage:       "mda",
    Handler: func(ctx contract.CommandContext) error {
        md := "## 代码演示\n\n" +
            "```markdown\n" +
            "# 标题\n" +
            "**加粗文字**\n" +
            "- 列表项\n" +
            "```"
        return ctx.ReplyMarkdown(md)
    },
})
```

- 依赖 `CommandRegister` + `QQAPI`
- 自定义 Markdown：2026/04 起单聊/群聊直接发送，无需申请
- 频道场景的自定义 Markdown 需内邀开通
- `CmdInput` 参数指令（所有场景）和 `CmdEnter` 回车指令（仅 C2C）
- `/mda` 命令演示 `ctx.ReplyMarkdown()` 被动回复

### menu — 简易菜单

`/menu` → 显示当前场景和发送者信息

- 依赖 `contract.CommandRegister`（最简依赖）
- 演示：`Scene()` 场景识别 + `AuthorID()` 获取发送者

## 数据库访问

框架使用 SQLite（通过 GORM）持久化运行时数据，插件可通过 `contract.Storage` 接口进行简单的 KV 存取：

```go
type Storage interface {
    Get(key string, dest interface{}) error
    Set(key string, value interface{}) error
    Delete(key string) error
    Close() error
}
```

在 `Register()` 中通过 `contract.PluginContext` 接收：

```go
func Register(pc *contract.PluginContext) {
    // KV 读写
    var val string
    if err := pc.Storage.Get("my_key", &val); err == nil {
        // 使用 val
    }
    _ = pc.Storage.Set("my_key", "my_value")
}
```

### 数据库表结构

框架自动维护以下表，无需手动创建，启动时通过 `AutoMigrate` 自动迁移。

#### KVEntry — 通用 KV 存储

| 字段 | 类型 | 说明 |
|------|------|------|
| `key` | VARCHAR(512) PK | 键 |
| `value` | TEXT | 值 |

由 `contract.Storage.Get/Set/Delete` 操作。

#### UserRecord — 用户记录（群聊 + C2C）

群聊 `member_openid` 与 C2C `user_openid` 共享同一 ID 命名空间，统一记录在此表。

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | VARCHAR(128) UNIQUE | QQ 用户 Open ID |
| `username` | VARCHAR(256) | 最近一次用户名 |
| `last_message` | TEXT | 最近一条消息内容（截断 500 字符） |
| `scene` | VARCHAR(32) INDEX | 最近场景：`"group"` / `"c2c"` |
| `first_seen_at` | DATETIME | 首次出现时间 |
| `last_seen_at` | DATETIME INDEX | 最近活跃时间 |
| `message_count` | INTEGER | 累计消息数 |

查询示例（需在 `Register()` 签名中获取 `*storage.Storage`）：

```go
// 获取最近活跃的 10 个用户
users, _ := store.QueryUsers(10, 0)
for _, u := range users {
    fmt.Printf("%s | %s | %d 条消息\n", u.UserID, u.Username, u.MessageCount)
}

// 统计总数
count, _ := store.CountUsers()
```

#### ChannelUserRecord — 频道用户记录（独立命名空间）

频道内用户 ID 与群聊/C2C 用户 ID 属于不同的命名空间，单独记录。

| 字段 | 类型 | 说明 |
|------|------|------|
| `user_id` | VARCHAR(128) | 频道内的用户 ID |
| `guild_id` | VARCHAR(128) | 所属频道组 ID |
| `channel_id` | VARCHAR(128) | 最近出现的子频道 ID |
| `username` | VARCHAR(256) | 最近一次用户名 |
| `last_message` | TEXT | 最近一条消息内容 |
| `first_seen_at` | DATETIME | 首次出现时间 |
| `last_seen_at` | DATETIME INDEX | 最近活跃时间 |
| `message_count` | INTEGER | 累计消息数 |

复合唯一索引 `(user_id, guild_id)`。

```go
users, _ := store.QueryChannelUsers("guild_id_xxx", 10, 0)
```

#### GroupRecord — 群聊记录

| 字段 | 类型 | 说明 |
|------|------|------|
| `group_id` | VARCHAR(128) UNIQUE | QQ 群 Open ID |
| `first_seen_at` | DATETIME | 首次出现时间 |
| `last_seen_at` | DATETIME INDEX | 最近活跃时间 |
| `message_count` | INTEGER | 累计消息数 |

```go
groups, _ := store.QueryGroups(10, 0)
count, _  := store.CountGroups()
```

#### ChannelRecord — 频道记录

| 字段 | 类型 | 说明 |
|------|------|------|
| `channel_id` | VARCHAR(128) UNIQUE | QQ 频道 ID |
| `guild_id` | VARCHAR(128) INDEX | 所属频道组 ID |
| `first_seen_at` | DATETIME | 首次出现时间 |
| `last_seen_at` | DATETIME INDEX | 最近活跃时间 |
| `message_count` | INTEGER | 累计消息数 |

```go
channels, _ := store.QueryChannels(10, 0)
count, _    := store.CountChannels()
```

#### LogEntry — 日志记录

所有符合条件的日志自动写入此表（受 `log_level_db` 和 `log_db_exclude` 配置过滤）。

| 字段 | 类型 | 说明 |
|------|------|------|
| `level` | VARCHAR(16) INDEX | 日志级别 |
| `message` | TEXT | 日志内容 |
| `fields` | TEXT | JSON 格式的附加 KV 上下文 |
| `source` | VARCHAR(128) INDEX | 来源模块 |
| `event_type` | VARCHAR(64) INDEX | QQ 事件类型 |
| `channel_id` | VARCHAR(128) INDEX | 来源频道 ID |
| `guild_id` | VARCHAR(128) INDEX | 频道服务器 ID |
| `group_id` | VARCHAR(128) INDEX | 群聊 ID |
| `author_id` | VARCHAR(128) INDEX | 消息发送者 ID |
| `created_at` | DATETIME INDEX | 日志时间 |

```go
// 查询最近 50 条 info 级别日志
logs, _ := store.QueryLogs("info", "", "", "", "", "", "", 50, 0)

// 按事件类型筛选
logs, _ = store.QueryLogs("", "", "GROUP_MESSAGE_CREATE", "", "", "", "", 20, 0)

// 按用户筛选
logs, _ = store.QueryLogs("", "", "", "", "", "", "user_openid", 10, 0)

// 统计日志总数
total, _ := store.CountLogs()
```

#### DailyRecord — 每日统计

每个自然日对应一行，框架自动累计。

| 分类 | 字段 | 说明 |
|------|------|------|
| C2C | `c2c_active_users` | 使用用户数（去重） |
| | `c2c_new_users` | 新添加用户数 |
| | `c2c_removed_users` | 新移除用户数 |
| | `c2c_incoming_msg` | 上行消息量 |
| | `c2c_incoming_users` | 上行消息人数（去重） |
| | `c2c_outgoing_msg` | 下行消息量 |
| 群聊 | `group_active_count` | 使用群数（去重） |
| | `group_new_count` | 新添加群数 |
| | `group_removed_count` | 新移除群数 |
| | `group_incoming_msg` | 群上行消息量 |
| | `group_outgoing_msg` | 群下行消息量 |
| 频道 | `channel_active_count` | 使用频道数（去重） |
| | `channel_new_count` | 新添加频道数 |
| | `channel_removed_count` | 新移除频道数 |
| | `channel_incoming_msg` | 频道上行消息量 |
| | `channel_outgoing_msg` | 频道下行消息量 |
| 通用 | `total_commands` | 指令执行次数 |
| | `total_interactions` | 按钮/交互点击次数 |

```go
// 查询最近一周的统计
records, _ := store.QueryDailyRecords("2026-06-30", "2026-07-06", 10, 0)
for _, r := range records {
    fmt.Printf("%s → 指令:%d C2C上行:%d 群上行:%d 频道上行:%d\n",
        r.Date, r.TotalCommands,
        r.C2CIncomingMsg, r.GroupIncomingMsg, r.ChannelIncomingMsg)
}

// 统计天数
total, _ := store.CountDailyRecords()
```

### 自动更新机制

这些表由框架自动维护，插件无需手动调用更新方法：

| 触发事件 | 更新的表和字段 |
|---------|-------------|
| 收到频道消息 | `ChannelRecord` upsert、`ChannelUserRecord` upsert、`DailyRecord.ChannelIncomingMsg` +1 |
| 收到群聊消息 | `GroupRecord` upsert、`UserRecord` upsert、`DailyRecord.GroupIncomingMsg` +1 |
| 收到 C2C 消息 | `UserRecord` upsert、`DailyRecord.C2CIncomingMsg` +1 |
| 指令被执行 | `DailyRecord.TotalCommands` +1 |
| 按钮被点击 | `DailyRecord.TotalInteractions` +1 |
| 回复/发送消息 | 对应场景的 `DailyRecord.*OutgoingMsg` +1 |
| 机器人加入群/频道 | `DailyRecord.GroupNewCount` / `ChannelNewCount` +1 |
| 机器人被移出 | `DailyRecord.GroupRemovedCount` / `ChannelRemovedCount` +1 |

### 日志清理

通过 `config.yaml` 配置自动清理：

```yaml
storage:
  log_cleanup:
    enabled: true       # 启用定时清理
    interval: "24h"     # 每 24 小时执行一次
    retain_days: 30     # 保留 30 天内的日志
```

启动时立即执行一次清理，之后按 `interval` 周期执行。

## 开发规范

- 只依赖 `internal/contract` 和 `internal/types`，不依赖框架内部包
- `Register()` 签名根据实际需要选择参数，不需要的不传
- 禁止使用 `init()` 注册，所有注册通过显式 `Register()` 调用
- 新增插件三步：① 创建包 ② 实现 Register() ③ 在 bot.go 的 registerPlugins() 中添加

更新时间：2026-07-06
