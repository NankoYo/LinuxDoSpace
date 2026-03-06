package httpapi

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	webassets "linuxdospace/backend/web"
)

// newSPAHandler 创建一个前端静态资源处理器。
// 该处理器承担两个职责：
// 1. 直接返回实际存在的静态文件，例如 `/assets/*.js`。
// 2. 对于前端单页应用路由，例如 `/settings`，统一回退到 `index.html`。
func newSPAHandler() (http.Handler, error) {
	distFS, err := fs.Sub(webassets.DistFS, "dist")
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		requestedPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if requestedPath == "." || requestedPath == "" {
			requestedPath = "index.html"
		}

		if requestedPath != "index.html" {
			if info, err := fs.Stat(distFS, requestedPath); err == nil && !info.IsDir() {
				http.ServeFileFS(w, r, distFS, requestedPath)
				return
			}
		}

		http.ServeFileFS(w, r, distFS, "index.html")
	}), nil
}
