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
