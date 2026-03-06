package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"linuxdospace/backend/internal/config"
)

// RouterDependencies 汇总构造 HTTP 路由所需的基础依赖。
type RouterDependencies struct {
	Config  config.Config
	Version string
}

// NewRouter 创建当前阶段的基础路由。
// 这里先提供健康检查和基础 CORS 支持，后续功能路由会逐步接入。
func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":      "ok",
			"app":         deps.Config.App.Name,
			"version":     deps.Version,
			"env":         deps.Config.App.Env,
			"oauth_ready": deps.Config.OAuthConfigured(),
			"cf_ready":    deps.Config.CloudflareConfigured(),
			"time":        time.Now().UTC(),
		})
	})

	return withCORS(deps.Config.App.AllowedOrigins, mux)
}

// withCORS 为浏览器调用提供最小可用的跨域支持。
func withCORS(allowedOrigins []string, next http.Handler) http.Handler {
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		originSet[trimmed] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if _, allowed := originSet[origin]; allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
		}

		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON 统一输出 JSON 响应。
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}
