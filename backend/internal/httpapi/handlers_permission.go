package httpapi

import (
	"html"
	"log"
	"net/http"
	"strings"

	"linuxdospace/backend/internal/service"
)

// handleMyPermissions returns the current authenticated user's permission cards.
func (a *API) handleMyPermissions(w http.ResponseWriter, r *http.Request) {
	_, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}

	items, err := a.permissionService.ListMyPermissions(r.Context(), *user)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleSubmitPermissionApplication stores or refreshes one permission request.
func (a *API) handleSubmitPermissionApplication(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	var request service.SubmitPermissionApplicationRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.permissionService.SubmitPermissionApplication(r.Context(), *user, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// handleMyEmailRoutes returns the current user's user-manageable email routes.
func (a *API) handleMyEmailRoutes(w http.ResponseWriter, r *http.Request) {
	_, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}

	items, err := a.permissionService.ListMyEmailRoutes(r.Context(), *user)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleMyEmailTargets returns the current user's bound forwarding destinations.
func (a *API) handleMyEmailTargets(w http.ResponseWriter, r *http.Request) {
	_, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}

	items, err := a.permissionService.ListMyEmailTargets(r.Context(), *user)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleCreateMyEmailTarget binds one external mailbox to the current user and
// asks LinuxDoSpace itself to send the verification email when needed.
func (a *API) handleCreateMyEmailTarget(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	var request service.CreateMyEmailTargetRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.permissionService.CreateMyEmailTarget(r.Context(), *user, request)
	if err != nil {
		// Operator-facing logs keep recent target-binding failures visible in
		// container logs so production incidents can be debugged without first
		// correlating browser reports back to database state.
		log.Printf("create my email target failed for user %d email %q: %v", user.ID, request.Email, err)
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// handleVerifyEmailTarget consumes the public verification link sent to one
// claimed target inbox and responds with a tiny standalone success/failure page
// so the user does not need an active browser session to complete ownership
// verification.
func (a *API) handleVerifyEmailTarget(w http.ResponseWriter, r *http.Request) {
	result, err := a.permissionService.VerifyEmailTarget(r.Context(), r.URL.Query().Get("token"))
	if err != nil {
		normalized := service.NormalizeError(err)
		statusCode := normalized.StatusCode
		title := "目标邮箱验证失败"
		description := normalized.Message
		switch normalized.Code {
		case "validation_failed":
			statusCode = http.StatusGone
		case "not_found":
			statusCode = http.StatusNotFound
		}
		writeEmailTargetVerificationHTML(w, statusCode, title, description, strings.TrimRight(strings.TrimSpace(a.config.App.FrontendURL), "/")+"/emails")
		return
	}

	writeEmailTargetVerificationHTML(
		w,
		http.StatusOK,
		"目标邮箱验证成功",
		"该目标邮箱已经完成验证，现在可以回到邮箱页面把它用于默认邮箱或邮箱泛解析转发。",
		strings.TrimRight(strings.TrimSpace(a.config.App.FrontendURL), "/")+"/emails",
	)
	_ = result
}

// handleResendMyEmailTargetVerification retriggers the platform verification
// email for one still-pending target mailbox owned by the current user.
func (a *API) handleResendMyEmailTargetVerification(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	targetID, err := pathInt64(r, "targetID")
	if err != nil {
		writeError(w, err)
		return
	}

	item, err := a.permissionService.ResendMyEmailTargetVerification(r.Context(), *user, targetID)
	if err != nil {
		// Resend failures need the same visibility because they are usually
		// caused by outbound SMTP reachability problems that do not otherwise
		// surface in the application's structured audit table.
		log.Printf("resend my email target verification failed for user %d target %d: %v", user.ID, targetID, err)
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// writeEmailTargetVerificationHTML renders the small standalone page shown when
// the user opens the verification link directly from their mailbox.
func writeEmailTargetVerificationHTML(w http.ResponseWriter, statusCode int, title string, description string, backURL string) {
	escapedTitle := html.EscapeString(strings.TrimSpace(title))
	escapedDescription := html.EscapeString(strings.TrimSpace(description))
	escapedBackURL := html.EscapeString(strings.TrimSpace(backURL))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>` +
		escapedTitle +
		`</title><style>body{margin:0;font-family:"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;background:#0f172a;color:#e2e8f0;display:flex;min-height:100vh;align-items:center;justify-content:center;padding:24px}.card{max-width:640px;width:100%;background:rgba(15,23,42,.82);border:1px solid rgba(148,163,184,.22);border-radius:24px;padding:32px;box-shadow:0 24px 80px rgba(0,0,0,.35)}h1{margin:0 0 16px;font-size:30px;line-height:1.2}p{margin:0 0 20px;font-size:16px;line-height:1.85;color:#cbd5e1}a{display:inline-block;border-radius:14px;background:linear-gradient(135deg,#38bdf8,#22d3ee);padding:12px 18px;color:#082f49;text-decoration:none;font-weight:700}</style></head><body><main class="card"><h1>` +
		escapedTitle +
		`</h1><p>` +
		escapedDescription +
		`</p><a href="` +
		escapedBackURL +
		`">返回邮箱页面</a></main></body></html>`))
}

// handleUpsertDefaultEmailRoute writes the authenticated user's forwarding
// target for the always-owned default mailbox.
func (a *API) handleUpsertDefaultEmailRoute(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	var request service.UpsertMyDefaultEmailRouteRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.permissionService.UpsertMyDefaultEmailRoute(r.Context(), *user, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// handleUpsertCatchAllEmailRoute writes the authenticated user's catch-all
// forwarding target after the permission has been approved.
func (a *API) handleUpsertCatchAllEmailRoute(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	var request service.UpsertMyCatchAllEmailRouteRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.permissionService.UpsertMyCatchAllEmailRoute(r.Context(), *user, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// handleAdminPermissionPolicies returns all administrator-configurable policy
// rows that control permission eligibility and auto-approval.
func (a *API) handleAdminPermissionPolicies(w http.ResponseWriter, r *http.Request) {
	_, _, ok := a.requireVerifiedAdmin(w, r)
	if !ok {
		return
	}

	items, err := a.permissionService.ListPermissionPolicies(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleAdminUpdatePermissionPolicy updates one permission-policy row.
func (a *API) handleAdminUpdatePermissionPolicy(w http.ResponseWriter, r *http.Request) {
	session, actor, ok := a.requireVerifiedAdmin(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	policyKey := r.PathValue("policyKey")
	if policyKey == "" {
		writeError(w, service.ValidationError("policyKey is required"))
		return
	}

	var request service.UpdatePermissionPolicyRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.permissionService.UpdatePermissionPolicy(r.Context(), *actor, policyKey, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// handleAdminUserPermissions returns the current permission cards for one target user.
func (a *API) handleAdminUserPermissions(w http.ResponseWriter, r *http.Request) {
	_, _, ok := a.requireVerifiedAdmin(w, r)
	if !ok {
		return
	}

	userID, err := pathInt64(r, "userID")
	if err != nil {
		writeError(w, err)
		return
	}

	items, err := a.permissionService.ListPermissionsForUser(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleAdminSetUserPermission lets an administrator directly override one target user's permission state.
func (a *API) handleAdminSetUserPermission(w http.ResponseWriter, r *http.Request) {
	session, actor, ok := a.requireVerifiedAdmin(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	userID, err := pathInt64(r, "userID")
	if err != nil {
		writeError(w, err)
		return
	}

	permissionKey := r.PathValue("permissionKey")
	if permissionKey == "" {
		writeError(w, service.ValidationError("permissionKey is required"))
		return
	}

	var request service.AdminSetUserPermissionRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	item, err := a.permissionService.SetPermissionForUser(r.Context(), *actor, userID, permissionKey, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// handleAdminUpdateUserPermissionAccess lets an administrator adjust the
// mutable catch-all runtime access state for one target user.
func (a *API) handleAdminUpdateUserPermissionAccess(w http.ResponseWriter, r *http.Request) {
	session, actor, ok := a.requireVerifiedAdmin(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	userID, err := pathInt64(r, "userID")
	if err != nil {
		writeError(w, err)
		return
	}

	permissionKey := r.PathValue("permissionKey")
	if permissionKey == "" {
		writeError(w, service.ValidationError("permissionKey is required"))
		return
	}

	var request service.AdminUpdateEmailCatchAllAccessRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeError(w, err)
		return
	}

	if permissionKey != service.PermissionKeyEmailCatchAll {
		writeError(w, service.ValidationError("unsupported permission key"))
		return
	}

	item, err := a.permissionService.UpdateEmailCatchAllAccessForUser(r.Context(), *actor, userID, request)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
