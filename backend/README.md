# LinuxDoSpace Backend

本目录包含 LinuxDoSpace 的 Go 后端实现。

当前后端设计目标：

- 只接受 Linux Do OAuth 登录
- 使用 Cloudflare API 管理 `linuxdo.space` 等根域名下的 DNS 记录
- 通过本地 SQLite 保存用户、会话、配额、分配记录和审计日志
- 保持“从不信任，始终验证”的零信任风格

建议阅读顺序：

1. [docs/development/README.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/README.md)
2. [docs/development/architecture.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/architecture.md)
3. [docs/development/api.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/api.md)
