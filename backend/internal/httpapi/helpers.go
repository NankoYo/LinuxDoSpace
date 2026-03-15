package httpapi

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"

	"linuxdospace/backend/internal/model"
	"linuxdospace/backend/internal/security"
	"linuxdospace/backend/internal/service"
)

// oauthStateCookiePrefix namespaces one cookie per OAuth state so concurrent
// login attempts in multiple tabs do not overwrite each other.
const oauthStateCookiePrefix = "linuxdospace_oauth_state_"

// oauthTargetCookiePrefix mirrors the per-state cookie layout for the frontend
// target that should receive the post-login redirect.
const oauthTargetCookiePrefix = "linuxdospace_oauth_target_"

// The legacy shared cookie names are still cleared on write/delete so old
// browsers drop them naturally, but the current code no longer trusts them
// while resolving active OAuth callbacks.
const legacyOAuthStateCookieName = "linuxdospace_oauth_state"
const legacyOAuthTargetCookieName = "linuxdospace_oauth_target"

const (
	oauthTargetApp   = "app"
	oauthTargetAdmin = "admin"
)

// writeJSON writes a successful JSON response envelope.
func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{"data": payload})
}

// writeError writes a normalized JSON error response envelope.
func writeError(w http.ResponseWriter, err error) {
	normalized := service.NormalizeError(err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(normalized.StatusCode)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    normalized.Code,
			"message": normalized.Message,
		},
	})
}

// decodeJSONBody strictly parses one JSON request object and rejects unknown fields.
func decodeJSONBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return service.ValidationError("invalid json request body")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return service.ValidationError("request body must contain exactly one json object")
	}
	return nil
}

// pathInt64 parses one positive int64 from a standard library PathValue.
func pathInt64(r *http.Request, key string) (int64, error) {
	value := strings.TrimSpace(r.PathValue(key))
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return 0, service.ValidationError("invalid path parameter: " + key)
	}
	return parsed, nil
}

// currentSessionCookieValue reads the current session cookie value from the request.
func (a *API) currentSessionCookieValue(r *http.Request) string {
	cookie, err := r.Cookie(a.sessionCookieName())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

// setSessionCookie writes the authenticated session cookie.
func (a *API) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.sessionCookieName(),
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(a.config.App.SessionTTL.Seconds()),
	})
}

// clearSessionCookie removes the authenticated session cookie.
func (a *API) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.sessionCookieName(),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	if legacyName := strings.TrimSpace(a.config.App.SessionCookieName); legacyName != "" && legacyName != a.sessionCookieName() {
		http.SetCookie(w, &http.Cookie{
			Name:     legacyName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   a.config.App.SessionSecure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
	}
}

// setOAuthStateCookie writes the short-lived per-state OAuth cookie.
func (a *API) setOAuthStateCookie(w http.ResponseWriter, stateID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.oauthStateCookieName(stateID),
		Value:    stateID,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((10 * time.Minute).Seconds()),
	})
}

// clearOAuthStateCookie removes the short-lived OAuth state cookie for the
// specified state and also clears the legacy shared cookie names.
func (a *API) clearOAuthStateCookie(w http.ResponseWriter, stateID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.oauthStateCookieName(stateID),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     legacyOAuthStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// currentOAuthStateCookie reads the short-lived OAuth state cookie bound to the
// current callback's state identifier.
func (a *API) currentOAuthStateCookie(r *http.Request, stateID string) string {
	if cookie, err := r.Cookie(a.oauthStateCookieName(stateID)); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

// setOAuthTargetCookie writes the short-lived login target cookie for one
// specific OAuth state so separate tabs can complete independently.
func (a *API) setOAuthTargetCookie(w http.ResponseWriter, stateID string, target string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.oauthTargetCookieName(stateID),
		Value:    normalizeOAuthTarget(target),
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((10 * time.Minute).Seconds()),
	})
}

// clearOAuthTargetCookie removes the short-lived login target cookie for the
// specified state and clears the legacy shared cookie name too.
func (a *API) clearOAuthTargetCookie(w http.ResponseWriter, stateID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.oauthTargetCookieName(stateID),
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     legacyOAuthTargetCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.config.App.SessionSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// currentOAuthTargetCookie reads the short-lived login target cookie for the
// callback's state.
func (a *API) currentOAuthTargetCookie(r *http.Request, stateID string) string {
	if cookie, err := r.Cookie(a.oauthTargetCookieName(stateID)); err == nil {
		return normalizeOAuthTarget(cookie.Value)
	}
	return oauthTargetApp
}

// sessionCookieName derives the effective authentication cookie name. In
// secure deployments the backend upgrades to a `__Host-` name so untrusted
// user-controlled subdomains cannot toss a same-name parent-domain cookie.
func (a *API) sessionCookieName() string {
	return secureHostCookieName(a.config.App.SessionCookieName, a.config.App.SessionSecure)
}

// normalizeOAuthTarget restricts login targets to the two supported frontend applications.
func normalizeOAuthTarget(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case oauthTargetAdmin:
		return oauthTargetAdmin
	default:
		return oauthTargetApp
	}
}

// oauthStateCookieName derives the per-state cookie name used for callback
// verification without reintroducing cross-tab overwrites.
func oauthStateCookieName(stateID string) string {
	if strings.TrimSpace(stateID) == "" {
		return legacyOAuthStateCookieName
	}
	return oauthStateCookiePrefix + strings.TrimSpace(stateID)
}

// oauthStateCookieName derives the per-state cookie name and upgrades it to a
// `__Host-` cookie in secure deployments for the same anti-tossing reason as
// the authenticated session cookie.
func (a *API) oauthStateCookieName(stateID string) string {
	return secureHostCookieName(oauthStateCookieName(stateID), a.config.App.SessionSecure)
}

// oauthTargetCookieName derives the matching per-state target cookie name.
func oauthTargetCookieName(stateID string) string {
	if strings.TrimSpace(stateID) == "" {
		return legacyOAuthTargetCookieName
	}
	return oauthTargetCookiePrefix + strings.TrimSpace(stateID)
}

// oauthTargetCookieName derives the matching per-state target cookie name and
// upgrades it to `__Host-` in secure deployments.
func (a *API) oauthTargetCookieName(stateID string) string {
	return secureHostCookieName(oauthTargetCookieName(stateID), a.config.App.SessionSecure)
}

// secureHostCookieName upgrades one host-only cookie name to the `__Host-`
// prefix in secure deployments. Existing explicit `__Host-` names stay intact.
func secureHostCookieName(baseName string, secure bool) string {
	trimmedName := strings.TrimSpace(baseName)
	if trimmedName == "" {
		return ""
	}
	if !secure || strings.HasPrefix(trimmedName, "__Host-") {
		return trimmedName
	}
	return "__Host-" + trimmedName
}

// optionalActor attempts to resolve the current user but gracefully treats invalid sessions as signed out.
func (a *API) optionalActor(w http.ResponseWriter, r *http.Request) (*model.Session, *model.User, error) {
	if a.authService == nil {
		return nil, nil, nil
	}

	sessionID := a.currentSessionCookieValue(r)
	if sessionID == "" {
		return nil, nil, nil
	}

	session, user, err := a.authService.AuthenticateSession(r.Context(), sessionID, security.FingerprintUserAgent(r))
	if err != nil {
		normalized := service.NormalizeError(err)
		if normalized.StatusCode == http.StatusUnauthorized || normalized.StatusCode == http.StatusForbidden {
			a.clearSessionCookie(w)
			return nil, nil, nil
		}
		return nil, nil, err
	}

	return &session, &user, nil
}

// requireActor requires a valid authenticated session.
func (a *API) requireActor(w http.ResponseWriter, r *http.Request) (*model.Session, *model.User, bool) {
	session, user, err := a.optionalActor(w, r)
	if err != nil {
		writeError(w, err)
		return nil, nil, false
	}
	if session == nil || user == nil {
		writeError(w, service.UnauthorizedError("authentication required"))
		return nil, nil, false
	}
	return session, user, true
}

// requireAdmin requires the current actor to hold application administrator permissions.
func (a *API) requireAdmin(w http.ResponseWriter, r *http.Request) (*model.Session, *model.User, bool) {
	session, user, ok := a.requireActor(w, r)
	if !ok {
		return nil, nil, false
	}
	if !user.IsAppAdmin {
		writeError(w, service.ForbiddenError("admin permission required"))
		return nil, nil, false
	}
	return session, user, true
}

// requireVerifiedAdmin requires one administrator session that has already
// completed the extra admin password verification step.
func (a *API) requireVerifiedAdmin(w http.ResponseWriter, r *http.Request) (*model.Session, *model.User, bool) {
	session, user, ok := a.requireAdmin(w, r)
	if !ok {
		return nil, nil, false
	}
	if !service.AdminVerificationIsFresh(session.AdminVerifiedAt, a.config.App.AdminVerificationTTL, time.Now().UTC()) {
		writeError(w, &service.Error{
			StatusCode: http.StatusForbidden,
			Code:       "admin_password_required",
			Message:    "admin password verification required",
		})
		return nil, nil, false
	}
	return session, user, true
}

// enforceCSRF validates the double-submit token on unsafe requests.
func (a *API) enforceCSRF(w http.ResponseWriter, r *http.Request, session *model.Session) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
		return true
	}
	if !a.requestOriginAllowed(r) {
		writeError(w, service.ForbiddenError("request origin is not allowed"))
		return false
	}
	if strings.TrimSpace(r.Header.Get("X-CSRF-Token")) != session.CSRFToken {
		writeError(w, service.UnauthorizedError("invalid csrf token"))
		return false
	}
	return true
}

// requestOriginAllowed validates one unsafe request's browser origin when the
// caller supplied `Origin` or `Referer`. Requests without either header remain
// accepted for compatibility with tests and non-browser debugging tools.
func (a *API) requestOriginAllowed(r *http.Request) bool {
	requestOrigin := strings.TrimSpace(r.Header.Get("Origin"))
	if requestOrigin == "" {
		fetchSite := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")))
		if fetchSite == "cross-site" {
			return false
		}
		refererValue := strings.TrimSpace(r.Header.Get("Referer"))
		if refererValue == "" {
			return true
		}
		refererURL, err := url.Parse(refererValue)
		if err != nil || refererURL.Scheme == "" || refererURL.Host == "" {
			return false
		}
		requestOrigin = refererURL.Scheme + "://" + refererURL.Host
	}

	allowedOrigins := make(map[string]struct{}, len(a.config.App.AllowedOrigins)+2)
	for _, origin := range a.config.App.AllowedOrigins {
		trimmedOrigin := strings.TrimRight(strings.TrimSpace(origin), "/")
		if trimmedOrigin == "" {
			continue
		}
		allowedOrigins[trimmedOrigin] = struct{}{}
	}
	for _, frontendOrigin := range []string{a.config.App.FrontendURL, a.config.App.AdminFrontendURL} {
		trimmedOrigin := strings.TrimRight(strings.TrimSpace(frontendOrigin), "/")
		if trimmedOrigin == "" {
			continue
		}
		allowedOrigins[trimmedOrigin] = struct{}{}
	}

	normalizedOrigin := strings.TrimRight(requestOrigin, "/")
	_, allowed := allowedOrigins[normalizedOrigin]
	if !allowed {
		return false
	}
	if strings.HasPrefix(r.URL.Path, "/v1/admin/") {
		adminOrigin := strings.TrimRight(strings.TrimSpace(a.config.App.AdminFrontendURL), "/")
		return adminOrigin != "" && normalizedOrigin == adminOrigin
	}
	return true
}

// frontendRedirectURL combines the configured frontend base URL with a normalized relative path.
func (a *API) frontendRedirectURL(target string, nextPath string) string {
	base := strings.TrimRight(strings.TrimSpace(a.config.App.FrontendURL), "/")
	if normalizeOAuthTarget(target) == oauthTargetAdmin {
		base = strings.TrimRight(strings.TrimSpace(a.config.App.AdminFrontendURL), "/")
	}
	path := security.NormalizePathOnly(nextPath)
	if base == "" {
		return path
	}
	return base + path
}

// defaultTrustedProxyCIDRs keeps tests and hand-built configs fail-closed even
// when they do not call config.Load. Only loopback hops are trusted by default.
var defaultTrustedProxyCIDRs = []netip.Prefix{
	netip.MustParsePrefix("127.0.0.1/32"),
	netip.MustParsePrefix("::1/128"),
}

// requestClientIP extracts the best available client IP from trusted reverse-
// proxy headers and finally falls back to RemoteAddr. Only explicitly allowed
// proxy CIDRs may influence the resolved address.
func requestClientIP(r *http.Request, trustedProxyCIDRs []netip.Prefix) string {
	if trustedProxyPeer(r.RemoteAddr, trustedProxyCIDRs) {
		if value, ok := parseHeaderIP(r.Header.Get("CF-Connecting-IP")); ok {
			return value
		}
		if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
			if value, ok := firstUntrustedForwardedIP(forwardedFor, trustedProxyCIDRs); ok {
				return value
			}
		}
		if value, ok := parseHeaderIP(r.Header.Get("X-Real-IP")); ok {
			return value
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// trustedProxyPeer reports whether the direct peer belongs to the configured
// reverse-proxy allowlist.
func trustedProxyPeer(remoteAddr string, trustedProxyCIDRs []netip.Prefix) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err != nil {
		host = strings.TrimSpace(remoteAddr)
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return false
	}

	effectiveCIDRs := trustedProxyCIDRs
	if len(effectiveCIDRs) == 0 {
		effectiveCIDRs = defaultTrustedProxyCIDRs
	}
	for _, prefix := range effectiveCIDRs {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}

// parseHeaderIP validates one single-value proxy header before it is used as a
// limiter key or audit attribute.
func parseHeaderIP(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}
	address, err := netip.ParseAddr(value)
	if err != nil {
		return "", false
	}
	return address.String(), true
}

// firstUntrustedForwardedIP walks the X-Forwarded-For chain from right to left
// and returns the first address that is not itself a trusted proxy. This keeps
// multi-hop deployments aligned with the real client instead of an inner
// reverse proxy.
func firstUntrustedForwardedIP(raw string, trustedProxyCIDRs []netip.Prefix) (string, bool) {
	parts := strings.Split(raw, ",")
	if len(parts) == 0 {
		return "", false
	}

	effectiveCIDRs := trustedProxyCIDRs
	if len(effectiveCIDRs) == 0 {
		effectiveCIDRs = defaultTrustedProxyCIDRs
	}

	for index := len(parts) - 1; index >= 0; index-- {
		candidate := strings.TrimSpace(parts[index])
		if candidate == "" {
			continue
		}
		address, err := netip.ParseAddr(candidate)
		if err != nil {
			continue
		}
		if !addressInTrustedCIDRs(address, effectiveCIDRs) {
			return address.String(), true
		}
	}

	for _, part := range parts {
		candidate := strings.TrimSpace(part)
		if address, err := netip.ParseAddr(candidate); err == nil {
			return address.String(), true
		}
	}
	return "", false
}

// addressInTrustedCIDRs reports whether the given address belongs to the
// configured trusted proxy set.
func addressInTrustedCIDRs(address netip.Addr, trustedProxyCIDRs []netip.Prefix) bool {
	for _, prefix := range trustedProxyCIDRs {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
