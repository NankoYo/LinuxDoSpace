# LinuxDoSpace 开发运行手册

## 本地启动后端

1. 进入 `backend/` 目录。
2. 参考 `.env.example` 设置环境变量。
3. 执行 `go run ./cmd/linuxdospace`。
4. 访问 `http://localhost:8080/healthz` 进行健康检查。
5. 访问 `GET /v1/public/domains` 检查默认根域名是否已经自动引导。

## 本地启动前端

1. 进入 `frontend/` 目录。
2. 参考 `.env.example` 设置 `VITE_API_BASE_URL`，默认值为 `http://localhost:8080`。
3. 执行 `npm install`。
4. 执行 `npm run dev`。
5. 浏览器访问 `http://localhost:3000`。
6. 点击登录按钮时，前端会跳转到 `${VITE_API_BASE_URL}/v1/auth/login`。

## 当前关键依赖

- Go 1.25.x
- SQLite
- Cloudflare API Token
- Linux Do OAuth Client

## 本地开发建议环境变量

- `APP_SESSION_SECRET`
- `CLOUDFLARE_API_TOKEN`
- `LINUXDO_OAUTH_CLIENT_ID`
- `LINUXDO_OAUTH_CLIENT_SECRET`
- `LINUXDO_OAUTH_REDIRECT_URL`

## Cloudflare 集成测试

执行真实集成测试前，设置：

- `LINUXDOSPACE_CF_API_TOKEN`
- `LINUXDOSPACE_CF_ZONE_ID`
- 可选：`LINUXDOSPACE_CF_ROOT_DOMAIN`

然后执行：

```powershell
go test ./internal/cloudflare -run TestClientIntegrationCreateGetDelete -v
```

## 说明

- 当前代码允许在开发环境未配置 OAuth 的情况下先启动；这时认证接口会返回 `503`。
- 默认根域名支持自动引导，如果未显式配置 `CLOUDFLARE_DEFAULT_ZONE_ID`，服务会尝试通过 Cloudflare API 查询。
- 前端已接入登录态、域名查询、命名空间申请和 DNS 记录管理接口。
- 为了支持 OAuth 回跳，前端当前通过浏览器 URL 和内部 tab 状态双向同步。
