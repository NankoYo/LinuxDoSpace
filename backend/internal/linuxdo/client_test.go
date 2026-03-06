package linuxdo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBuildAuthorizationURLDefaultsToUserScope(t *testing.T) {
	client := NewClient(
		"client-id",
		"client-secret",
		"https://api.linuxdo.space/v1/auth/callback",
		"https://connect.linux.do/oauth2/authorize",
		"https://connect.linux.do/oauth2/token",
		"https://connect.linux.do/api/user",
		"",
		false,
	)

	parsed, err := url.Parse(client.BuildAuthorizationURL("state-value", ""))
	if err != nil {
		t.Fatalf("parse authorization url: %v", err)
	}

	query := parsed.Query()
	if got := query.Get("scope"); got != "user" {
		t.Fatalf("expected default scope user, got %q", got)
	}
}

func TestExchangeCodeSendsFormAndAcceptHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected Accept application/json, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/x-www-form-urlencoded") {
			t.Fatalf("expected form content type, got %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("client_id"); got != "client-id" {
			t.Fatalf("unexpected client_id %q", got)
		}
		if got := r.Form.Get("client_secret"); got != "client-secret" {
			t.Fatalf("unexpected client_secret %q", got)
		}
		if got := r.Form.Get("grant_type"); got != "authorization_code" {
			t.Fatalf("unexpected grant_type %q", got)
		}
		if got := r.Form.Get("redirect_uri"); got != "https://api.linuxdo.space/v1/auth/callback" {
			t.Fatalf("unexpected redirect_uri %q", got)
		}

		_ = json.NewEncoder(w).Encode(TokenResponse{
			AccessToken: "token-value",
			TokenType:   "Bearer",
		})
	}))
	defer server.Close()

	client := NewClient(
		"client-id",
		"client-secret",
		"https://api.linuxdo.space/v1/auth/callback",
		"https://connect.linux.do/oauth2/authorize",
		server.URL,
		"https://connect.linux.do/api/user",
		"user",
		false,
	)

	token, err := client.ExchangeCode(context.Background(), "code-value", "")
	if err != nil {
		t.Fatalf("exchange code: %v", err)
	}
	if token.AccessToken != "token-value" {
		t.Fatalf("unexpected access token %q", token.AccessToken)
	}
}

func TestGetCurrentUserAcceptsDirectPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected Accept application/json, got %q", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-value" {
			t.Fatalf("unexpected authorization header %q", got)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":              42,
			"username":        "alice",
			"name":            "Alice",
			"avatar_template": "/user_avatar/linux.do/alice/{size}/1.png",
			"trust_level":     3,
		})
	}))
	defer server.Close()

	client := NewClient(
		"client-id",
		"client-secret",
		"https://api.linuxdo.space/v1/auth/callback",
		"https://connect.linux.do/oauth2/authorize",
		"https://connect.linux.do/oauth2/token",
		server.URL,
		"user",
		false,
	)

	profile, err := client.GetCurrentUser(context.Background(), "token-value")
	if err != nil {
		t.Fatalf("get current user: %v", err)
	}
	if profile.Username != "alice" {
		t.Fatalf("unexpected username %q", profile.Username)
	}
}
