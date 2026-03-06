# LinuxDoSpace 开发文档

本目录用于保存 LinuxDoSpace 的开发期持久化文档，确保后续维护、功能迭代、Bug 修复和审计都具备可追溯性。

建议阅读顺序：

1. [architecture.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/architecture.md)
2. [api.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/api.md)
3. [runbook.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/runbook.md)
4. [references.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/references.md)
5. [known-issues.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/known-issues.md)
6. [changelog.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/changelog.md)

当前阶段说明：

- 已完成 Go 后端可运行版本，包含 Linux Do OAuth、服务端会话、CSRF、防越权 DNS 命名空间管理和管理员接口。
- 已完成 SQLite 持久化、Cloudflare 真实集成测试和开发期文档沉淀。
- 已完成前端核心页面对接，包含登录态同步、域名查询、命名空间申请和 DNS 记录管理。
- 管理员相关后端接口尚未配套后台管理页面。
