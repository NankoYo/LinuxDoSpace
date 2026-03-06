# LinuxDoSpace 开发运行手册

## 本地启动后端

1. 进入 `backend/` 目录。
2. 参考 `.env.example` 设置环境变量。
3. 执行 `go run ./cmd/linuxdospace`。
4. 访问 `http://localhost:8080/healthz` 进行健康检查。

## 当前关键依赖

- Go 1.25.x
- SQLite
- Cloudflare API Token
- Linux Do OAuth Client

## 说明

- 当前代码允许在开发环境未配置 OAuth 的情况下先启动基础骨架。
- 需要真实 DNS 联调时，再注入 `CLOUDFLARE_API_TOKEN`。
