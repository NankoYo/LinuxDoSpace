# LinuxDoSpace API 文档

## 当前已实现接口

### `GET /healthz`

用途：
返回服务存活状态、版本、环境与基础依赖配置状态。

响应示例：

```json
{
  "status": "ok",
  "app": "LinuxDoSpace",
  "version": "dev",
  "env": "development",
  "oauth_ready": false,
  "cf_ready": false,
  "time": "2026-03-06T00:00:00Z"
}
```

## 规划中的接口

- `GET /v1/public/domains`
- `GET /v1/public/allocations/check`
- `GET /v1/auth/login`
- `GET /v1/auth/callback`
- `POST /v1/auth/logout`
- `GET /v1/me`
- `GET /v1/my/allocations`
- `POST /v1/my/allocations`
- `GET /v1/my/allocations/{id}/records`
- `POST /v1/my/allocations/{id}/records`
- `PATCH /v1/my/records/{id}`
- `DELETE /v1/my/records/{id}`
- `GET /v1/admin/domains`
- `POST /v1/admin/domains`
