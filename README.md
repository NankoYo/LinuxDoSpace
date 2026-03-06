# LinuxDoSpace（佬友空间）

LinuxDoSpace 是一个基于 Cloudflare DNS 的二级域名分发站，主要面向 Linux Do 社区用户。

当前仓库包含：

- `frontend/`：基于 Vite 的前端演示界面。
- `backend/`：基于 Go 的后端服务。
- `docs/development/`：项目开发文档、API 文档、已知问题与变更记录。

当前开发重点是后端基础能力：

- Linux Do OAuth 登录
- Cloudflare DNS 记录安全管理
- 多根域名分发
- 配额与子域名归属控制
- 审计日志与安全校验

详细开发文档见 [docs/development/README.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/README.md)。
