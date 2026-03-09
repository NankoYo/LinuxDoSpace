package httpapi

import (
	"log"
	"net/http"
	"net/url"

	"linuxdospace/backend/internal/security"
	"linuxdospace/backend/internal/service"
)

// handleAuthLogin starts the standard user-facing Linux Do OAuth login flow.
func (a *API) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthLogin(w, r, oauthTargetApp)
}

// handleAdminAuthLogin starts the administrator console login flow and remembers the admin redirect target.
func (a *API) handleAdminAuthLogin(w http.ResponseWriter, r *http.Request) {
	a.handleOAuthLogin(w, r, oauthTargetAdmin)
}

// handleOAuthLogin starts one Linux Do OAuth login flow for the selected frontend target.
func (a *API) handleOAuthLogin(w http.ResponseWriter, r *http.Request, target string) {
	result, err := a.authService.BeginLogin(r.Context(), r.URL.Query().Get("next"))
	if err != nil {
		writeError(w, err)
		return
	}

	a.setOAuthStateCookie(w, result.StateID)
	a.setOAuthTargetCookie(w, target)
	http.Redirect(w, r, result.RedirectURL, http.StatusFound)
}

// handleAuthCallback completes the shared Linux Do OAuth callback for both the app frontend and the admin frontend.
func (a *API) handleAuthCallback(w http.ResponseWriter, r *http.Request) {
	target := a.currentOAuthTargetCookie(r)
	result, err := a.authService.CompleteLogin(
		r.Context(),
		r.URL.Query().Get("state"),
		a.currentOAuthStateCookie(r),
		r.URL.Query().Get("code"),
		security.FingerprintUserAgent(r),
	)
	if err != nil {
		log.Printf("linuxdo oauth callback failed: %v", err)
		a.clearOAuthStateCookie(w)
		a.clearOAuthTargetCookie(w)
		normalized := service.NormalizeError(err)
		redirectTarget := oauthTargetApp
		if normalizeOAuthTarget(target) == oauthTargetAdmin {
			redirectTarget = oauthTargetAdmin
		}
		http.Redirect(w, r, a.frontendRedirectURL(redirectTarget, "/?auth_error="+url.QueryEscape(normalized.Code)), http.StatusFound)
		return
	}

	if normalizeOAuthTarget(target) == oauthTargetAdmin && !result.User.IsAppAdmin {
		if logoutErr := a.authService.Logout(r.Context(), result.Session.ID, result.User.ID); logoutErr != nil {
			log.Printf("cleanup unauthorized admin session failed: %v", logoutErr)
		}
		a.clearOAuthStateCookie(w)
		a.clearOAuthTargetCookie(w)
		http.Redirect(w, r, a.frontendRedirectURL(oauthTargetAdmin, "/?auth_error=admin_required"), http.StatusFound)
		return
	}

	// Auto-provisioning is not required for successful login, so failures are logged without aborting the session.
	if err := a.domainService.AutoProvisionForUser(r.Context(), result.User); err != nil {
		log.Printf("auto provision after login failed for user %s: %v", result.User.Username, err)
	}

	a.setSessionCookie(w, result.Session.ID)
	a.clearOAuthStateCookie(w)
	a.clearOAuthTargetCookie(w)
	http.Redirect(w, r, a.frontendRedirectURL(target, result.NextPath), http.StatusFound)
}

// handleAuthLogout destroys the current authenticated session.
func (a *API) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return
	}
	if !a.enforceCSRF(w, r, session) {
		return
	}

	if err := a.authService.Logout(r.Context(), session.ID, user.ID); err != nil {
		writeError(w, err)
		return
	}

	a.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"logged_out": true})
}

// handleMe returns the current public-site session together with the visible allocations.
func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	session, user, err := a.optionalActor(w, r)
	if err != nil {
		writeError(w, err)
		return
	}

	if session == nil || user == nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"authenticated":    false,
			"oauth_configured": a.config.OAuthConfigured(),
		})
		return
	}

	allocations, err := a.domainService.ListVisibleAllocationsForUser(r.Context(), *user)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated":      true,
		"user":               user,
		"csrf_token":         session.CSRFToken,
		"session_expires_at": session.ExpiresAt,
		"allocations":        allocations,
	})
}
