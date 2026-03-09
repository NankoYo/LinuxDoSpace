package httpapi

import (
	"net/http/httptest"
	"testing"

	"linuxdospace/backend/internal/config"
)

// TestOAuthCookiesSupportConcurrentFlows verifies that one browser can keep
// multiple pending OAuth callbacks alive at the same time without app/admin
// flows overwriting each other's state or redirect target cookies.
func TestOAuthCookiesSupportConcurrentFlows(t *testing.T) {
	api := &API{
		config: config.Config{
			App: config.AppConfig{
				SessionSecure: false,
			},
		},
	}

	recorder := httptest.NewRecorder()
	api.setOAuthStateCookie(recorder, "state-app")
	api.setOAuthTargetCookie(recorder, "state-app", oauthTargetApp)
	api.setOAuthStateCookie(recorder, "state-admin")
	api.setOAuthTargetCookie(recorder, "state-admin", oauthTargetAdmin)

	request := httptest.NewRequest("GET", "/v1/auth/callback", nil)
	for _, cookie := range recorder.Result().Cookies() {
		request.AddCookie(cookie)
	}

	if got := api.currentOAuthStateCookie(request, "state-app"); got != "state-app" {
		t.Fatalf("expected app state cookie to remain readable, got %q", got)
	}
	if got := api.currentOAuthTargetCookie(request, "state-app"); got != oauthTargetApp {
		t.Fatalf("expected app target cookie %q, got %q", oauthTargetApp, got)
	}
	if got := api.currentOAuthStateCookie(request, "state-admin"); got != "state-admin" {
		t.Fatalf("expected admin state cookie to remain readable, got %q", got)
	}
	if got := api.currentOAuthTargetCookie(request, "state-admin"); got != oauthTargetAdmin {
		t.Fatalf("expected admin target cookie %q, got %q", oauthTargetAdmin, got)
	}
}

// TestShouldClearOAuthCookiesKeepsRetryableErrors verifies that upstream
// outages do not burn the local browser flow while terminal callback failures do.
func TestShouldClearOAuthCookiesKeepsRetryableErrors(t *testing.T) {
	if shouldClearOAuthCookies("service_unavailable") {
		t.Fatalf("expected service_unavailable to preserve oauth cookies for retry")
	}
	if shouldClearOAuthCookies("internal_error") {
		t.Fatalf("expected internal_error to preserve oauth cookies for retry")
	}
	if !shouldClearOAuthCookies("unauthorized") {
		t.Fatalf("expected unauthorized callbacks to clear oauth cookies")
	}
}
