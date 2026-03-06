package web

import "embed"

// DistFS 嵌入前端构建产物目录。
// Docker 构建流程会先生成 `frontend/dist`，再把结果复制到 `backend/web/dist`，
// 最终由 Go 二进制把整套前端静态资源一起打包进镜像。
//
//go:embed all:dist
var DistFS embed.FS
