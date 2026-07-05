# LQBOT 框架

## 项目概述
LQBOT 是一个基于 Go 语言开发的 QQ 机器人框架。
- 使用方法请参考[文档示例](https://github.com/Luoyangan/LQBOT/tree/main/plugins)。

## 技术栈
- **语言**: Go 1.22+
- **API**: [QQ 官方 API v2](https://bot.q.qq.com/wiki/develop/api-v2/)（WebSocket/Webhook 接入）
- **官方 SDK**: [tencent-connect/botgo](https://github.com/tencent-connect/botgo)
- **认证方式**: OAuth2 Client Credentials（AppID + AppSecret → Access Token）
- **数据库**: SQLite（通过 GORM 操作）

## 构建
```powershell
# Windows
.\scripts\build.ps1 windows

# Linux
.\scripts\build.ps1 linux

# 双平台
.\scripts\build.ps1 all
```

## 配置
- [配置文件模板](https://github.com/Luoyangan/LQBOT/blob/main/configs/config.example.yaml)
```yaml
# LQBOT 配置文件示例
# 复制此文件为 config.yaml 并填入实际值

app_id: "your_app_id_here"       # QQ 开放平台申请的 AppID
app_secret: "your_app_secret_here" # QQ 开放平台申请的 AppSecret
sandbox: false                    # true = 沙箱模式, false = 生产模式
intents:
  # 公域机器人请使用 AT_MESSAGES 代替 GUILD_MESSAGES
  #- AT_MESSAGES                  # @机器人消息（公域机器人用这个）
  - GUILD_MESSAGES               # 频道全部消息（仅私域机器人可用）
  - GROUP_AND_C2C_EVENT          # 群聊和私聊（需申请权限）
  - INTERACTION                  # 按钮/选择框交互事件
access_type: websocket           # websocket | webhook
log_level: info

webhook:                         # webhook 模式配置（access_type: webhook 时生效）
  port: 8080                     # HTTP 监听端口
  path: /webhook                 # 回调路径

storage:                         # 数据库配置
  driver: sqlite                 # 数据库驱动，支持 sqlite
  dsn: "data/lqbot.db"           # 数据库文件路径

```

## 作者
- [Luoyangan](https://github.com/Luoyangan)
- [个人博客](https://mcyszl.top)
- [QQ群：812500721](https://qm.qq.com/q/9Rvq6VylQA)
- [MIT License](LICENSE)
