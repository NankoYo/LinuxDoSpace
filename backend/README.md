# LinuxDoSpace Backend

This directory contains the Go backend for LinuxDoSpace.

## Responsibilities

- Linux Do OAuth login and session management
- Cloudflare DNS management for managed root domains such as `linuxdo.space`
- SQLite persistence for users, sessions, quotas, allocations, admin data, and audit logs
- Static asset hosting for the main frontend build bundled into the backend image

## Key endpoints

Public and user-facing endpoints:

- `GET /healthz`
- `GET /v1/public/domains`
- `GET /v1/public/supervision`
- `GET /v1/public/allocations/check`
- `GET /v1/auth/login`
- `GET /v1/auth/callback`
- `POST /v1/auth/logout`
- `GET /v1/me`
- `GET/POST /v1/my/allocations`
- `GET/POST/PATCH/DELETE /v1/my/allocations/{allocationID}/records`

Administrator endpoints:

- `GET /v1/admin/auth/login`
- `GET /v1/admin/me`
- `GET/POST /v1/admin/domains`
- `POST /v1/admin/quotas`
- `GET /v1/admin/users`
- `GET/PATCH /v1/admin/users/{userID}`
- `GET /v1/admin/allocations`
- `GET /v1/admin/records`
- `POST/PATCH/DELETE /v1/admin/allocations/{allocationID}/records/{recordID}`
- `GET/POST/PATCH/DELETE /v1/admin/email-routes`
- `GET/PATCH /v1/admin/applications/{applicationID}`
- `GET/POST/DELETE /v1/admin/redeem-codes`

## Local run

```powershell
cd backend
go run ./cmd/linuxdospace
```

## Next reading

- [docs/development/README.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/README.md)
- [docs/development/architecture.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/architecture.md)
- [docs/development/api.md](/G:/ClaudeProjects/LinuxDoSpace/docs/development/api.md)
