# LinuxDoSpace 部署说明

## 部署形态

当前仓库采用单镜像部署方案：

- GitHub Actions 构建前端静态资源
- Go 后端把前端构建产物嵌入二进制
- Debian 服务器只需要运行一个容器

这样可以避免前后端拆分部署带来的跨域、回调地址和静态资源同步问题。

## Docker 镜像

- Dockerfile：仓库根目录 [Dockerfile](/G:/ClaudeProjects/LinuxDoSpace/Dockerfile)
- 运行时镜像默认监听容器内 `8080`
- SQLite 数据库默认挂载到 `/app/data/linuxdospace.sqlite`

## Debian 服务器准备

需要安装：

- Docker Engine
- Docker Compose Plugin

推荐部署目录：

- `/opt/linuxdospace`

## 服务器文件

仓库提供：

- Compose 文件：[deploy/docker-compose.yml](/G:/ClaudeProjects/LinuxDoSpace/deploy/docker-compose.yml)
- 环境变量模板：[deploy/linuxdospace.env.example](/G:/ClaudeProjects/LinuxDoSpace/deploy/linuxdospace.env.example)

在 Debian 服务器上，通常需要：

1. 创建 `/opt/linuxdospace`
2. 放入 `docker-compose.yml`
3. 放入 `.env`
4. 执行 `docker compose pull`
5. 执行 `docker compose up -d`

## GitHub Actions 工作流

工作流文件：

- [container-release.yml](/G:/ClaudeProjects/LinuxDoSpace/.github/workflows/container-release.yml)

功能：

- push 到 `main` 时自动构建并推送镜像到 GHCR
- push 版本 tag 时自动构建并推送对应 tag 镜像
- `workflow_dispatch` 手动触发时可选直接部署到 Debian 服务器

## 需要配置的 GitHub Secrets

构建推送到 GHCR：

- 默认使用 `GITHUB_TOKEN`，无需额外 Secrets

手动部署到 Debian 服务器时需要：

- `DEPLOY_HOST`
- `DEPLOY_PORT`（可选，默认 `22`）
- `DEPLOY_USER`
- `DEPLOY_PATH`（可选，默认 `/opt/linuxdospace`）
- `DEPLOY_SSH_PRIVATE_KEY`
- `DEPLOY_ENV_FILE`
- `DEPLOY_GHCR_USERNAME`
- `DEPLOY_GHCR_TOKEN`

其中：

- `DEPLOY_ENV_FILE` 应是完整的多行 `.env` 文件内容
- `DEPLOY_GHCR_TOKEN` 需要具备读取 GHCR 镜像的权限

## 部署后验证

可以在服务器上执行：

```bash
docker compose ps
docker compose logs -f
curl http://127.0.0.1:8080/healthz
```

如果服务对外经过反代，还应验证：

- 前端首页是否可访问
- `/v1/me` 是否可返回未登录状态
- Linux Do OAuth 回调地址是否与生产域名一致
