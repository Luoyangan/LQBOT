# LQBOT 框架

基于 Go 语言开发的 QQ 机器人框架，支持 WebSocket / Webhook 接入。

## 技术栈

- **语言**: Go 1.22+
- **API**: [QQ 官方 API v2](https://bot.q.qq.com/wiki/develop/api-v2/)（WebSocket/Webhook 接入）
- **官方 SDK**: [tencent-connect/botgo](https://github.com/tencent-connect/botgo)
- **认证方式**: OAuth2 Client Credentials（AppID + AppSecret → Access Token）
- **数据库**: SQLite（通过 GORM 操作）
- **配置**: YAML 格式
- **日志**: zerolog

## 快速开始

### 1. 配置

复制 `configs/config.example.yaml` 为 `configs/config.yaml`，填入 AppID 和 AppSecret：

```yaml
app_id: "your_app_id_here"
app_secret: "your_app_secret_here"
```

### 2. 运行

```bash
go run cmd/bot/main.go -c configs/config.yaml
```

### 3. 构建

```powershell
# Windows
.\scripts\build.ps1 windows

# Linux
.\scripts\build.ps1 linux

# 双平台
.\scripts\build.ps1 all
```

## 管理面板

- admin `admin` 配置项设为 `true` 启用网页管理界面，前提条件是 `enabled` 为 `true`
- 管理地址: `/admin` 首次访问需先设置登录密码
- 使用HTTPS需配置 `cert_file` 和 `key_file`，注释掉为不使用HTTPS
```yaml
http:                            # 内嵌 HTTP 服务配置（可选）
  enabled: false                 # true = 启用内嵌 HTTP 服务
  port: 80                       # HTTP 监听端口（默认 80）
  admin: true                    # true = 启用网页管理界面
  #cert_file: "cert.pem"           # SSL 证书文件
  #key_file: "key.pem"             # SSL 私钥文件
```

## 插件开发

插件是 `plugins/` 下的独立 Go 包，编译时静态链接进主程序。

```go
// plugins/mycmd/mycmd.go
package mycmd

import "github.com/Luoyangan/LQBOT/internal/contract"

func Register(r contract.CommandRegister) {
    r.Register(contract.Command{
        Name:        "hello",
        Description: "打个招呼",
        Handler: func(ctx contract.CommandContext) error {
            return ctx.Reply("Hello!")
        },
    })
}
```

然后在 `internal/bot/bot.go` 的 `registerPlugins()` 中添加：

```go
import "github.com/Luoyangan/LQBOT/plugins/mycmd"

func (b *Bot) registerPlugins() {
    // ... 已有插件
    mycmd.Register(b.router)
}
```

详细插件开发文档请参阅 **[plugins.md](plugins.md)**。

## 核心特性

- **多协议接入**: WebSocket（自动重连 + 指数退避）、Webhook
- **指令系统**: `/command` 前缀、参数解析、别名支持
- **事件驱动**: 事件总线，支持 QQ API v2 原生事件格式
- **中间件**: 日志、限流（Token Bucket）
- **权限控制**: 基于群聊 member_role（owner/admin/member/public）
- **定时任务**: Cron 表达式 + 固定间隔（基于 robfig/cron/v3）
- **内嵌 HTTP**: 可选嵌入式 HTTP 服务器，插件可注册路由
- **数据库**: SQLite + GORM，自动迁移，内置日志/实体/统计表
- **日志系统**: 结构化日志（zerolog），控制台着色 + 数据库双写
- **消息类型**: 文本、Markdown、图片、Ark、Embed、富媒体、按钮交互
- **被动回复**: 统一带 msg_id 的被动回复，绕过公域机器人主动消息限制
- **优雅关闭**: SIGINT/SIGTERM 捕获，超时控制

## 示例插件

| 插件 | 命令 | 说明 |
|------|------|------|
| ping | `/ping` | 在线测试（最简示例） |
| echo | `/echo <消息>` | 回声，演示参数解析 |
| hello | `/button` `/buttons` `/at` 等 | 按钮、文本链、@提及 |
| embed | `/embed` | 富卡片消息（仅频道） |
| ark | `/ark channel/group/c2c` | Ark 模板消息 |
| info | `/server` `/whoami` | 信息查询 |
| media | `/image` `/video` | 图片/视频 |
| manage | `/delete` `/pin` `/react` | 消息管理 |
| markdown | `/md` `/mda` | Markdown 消息 |
| menu | `/menu` | 简易菜单 |

## 配置参考

完整配置模板：[configs/config.example.yaml](configs/config.example.yaml)

```yaml
app_id: "xxx"
app_secret: "xxx"
sandbox: false                    # 沙箱模式
intents:
  - AT_MESSAGES                  # @消息触发
  - GROUP_AND_C2C_EVENT          # 群聊和私聊
  - INTERACTION                  # 按钮交互
access_type: websocket           # websocket | webhook
log_level: info                  # 日志级别
```

## 作者

- [Luoyangan](https://github.com/Luoyangan)
- [博客](https://mcyszl.top)
- [QQ群：812500721](https://qm.qq.com/q/9Rvq6VylQA)
- [MIT License](LICENSE)
