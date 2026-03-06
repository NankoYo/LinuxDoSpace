# LinuxDoSpace Frontend

本目录包含 LinuxDoSpace 的 Vite 前端实现。

当前前端已完成：

- 与后端 `/v1/me` 的登录态同步
- Linux Do OAuth 登录跳转接入
- 公开根域名列表加载
- 前缀可用性查询
- 命名空间申请
- allocation 选择与真实 DNS 记录 CRUD

本地开发：

1. 复制或参考 `.env.example`，设置 `VITE_API_BASE_URL`。
2. 安装依赖：`npm install`
3. 启动开发服务器：`npm run dev`
4. 运行类型检查：`npm run lint`
5. 构建生产包：`npm run build`

默认前端会连接 `http://localhost:8080`。
