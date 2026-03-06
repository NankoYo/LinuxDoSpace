# LinuxDoSpace 架构说明

## 当前架构分层

- `frontend/`：Vite 前端展示层。
- `backend/cmd/linuxdospace`：Go 进程入口。
- `backend/internal/config`：环境变量配置加载。
- `backend/internal/storage/sqlite`：SQLite 持久化层。
- `backend/internal/linuxdo`：Linux Do OAuth / 用户信息客户端。
- `backend/internal/cloudflare`：Cloudflare API 客户端。
- `backend/internal/httpapi`：HTTP 路由与响应层。
- `backend/internal/security`：随机令牌、指纹和安全辅助函数。

## 安全设计原则

- 不信任来自浏览器的任何用户输入。
- 不把敏感令牌写入仓库。
- 默认使用服务端会话而不是把身份信息直接暴露给前端。
- 所有重要操作都应沉淀为审计日志。

## 下一步架构扩展

- 增加 `service` 层承接业务规则。
- 增加 OAuth 会话、管理员校验和 CSRF 防护。
- 增加 Cloudflare 命名空间级记录管理。
