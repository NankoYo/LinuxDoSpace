package linuxdo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"linuxdospace/backend/internal/model"
)

// Client is a small Linux Do OAuth / userinfo client backed by the standard library.
type Client struct {
	httpClient   *http.Client
	clientID     string
	clientSecret string
	redirectURL  string
	authorizeURL string
	tokenURL     string
	userInfoURL  string
	scope        string
	enablePKCE   bool
}

// TokenResponse captures the minimum token fields used by the application.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// userEnvelope handles deployments where the user payload is wrapped under `user`.
type userEnvelope struct {
	User model.LinuxDOProfile `json:"user"`
}

// NewClient constructs a Linux Do client and falls back to the documented `user` scope.
func NewClient(clientID string, clientSecret string, redirectURL string, authorizeURL string, tokenURL string, userInfoURL string, scope string, enablePKCE bool) *Client {
	normalizedScope := strings.TrimSpace(scope)
	if normalizedScope == "" {
		normalizedScope = "user"
	}

	return &Client{
		httpClient:   &http.Client{Timeout: 20 * time.Second},
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		redirectURL:  strings.TrimSpace(redirectURL),
		authorizeURL: strings.TrimSpace(authorizeURL),
		tokenURL:     strings.TrimSpace(tokenURL),
		userInfoURL:  strings.TrimSpace(userInfoURL),
		scope:        normalizedScope,
		enablePKCE:   enablePKCE,
	}
}

// Configured reports whether the client has the minimum settings required for OAuth.
func (c *Client) Configured() bool {
	return c.clientID != "" && c.clientSecret != "" && c.redirectURL != ""
}

// BuildAuthorizationURL creates the Linux Do authorization URL for one login attempt.
func (c *Client) BuildAuthorizationURL(state string, codeChallenge string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", c.clientID)
	values.Set("redirect_uri", c.redirectURL)
	values.Set("state", state)
	values.Set("scope", c.scope)
	if c.enablePKCE && codeChallenge != "" {
		values.Set("code_challenge", codeChallenge)
		values.Set("code_challenge_method", "S256")
	}
	return c.authorizeURL + "?" + values.Encode()
}

// basicAuthorizationHeader returns the RFC 7617 Basic Authorization header required by Linux Do.
func (c *Client) basicAuthorizationHeader() string {
	credentials := c.clientID + ":" + c.clientSecret
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
}

// ExchangeCode swaps an authorization code for an access token.
func (c *Client) ExchangeCode(ctx context.Context, code string, codeVerifier string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.redirectURL)
	if c.enablePKCE && codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("create linuxdo token request: %w", err)
	}
	request.Header.Set("Authorization", c.basicAuthorizationHeader())
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("perform linuxdo token request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("read linuxdo token response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return TokenResponse{}, fmt.Errorf("linuxdo token request failed with status %d", response.StatusCode)
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return TokenResponse{}, fmt.Errorf("decode linuxdo token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return TokenResponse{}, fmt.Errorf("linuxdo token response did not contain access_token")
	}

	return token, nil
}

// GetCurrentUser fetches the current Linux Do profile with the issued access token.
func (c *Client) GetCurrentUser(ctx context.Context, accessToken string) (model.LinuxDOProfile, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if err != nil {
		return model.LinuxDOProfile{}, fmt.Errorf("create linuxdo userinfo request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return model.LinuxDOProfile{}, fmt.Errorf("perform linuxdo userinfo request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return model.LinuxDOProfile{}, fmt.Errorf("read linuxdo userinfo response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return model.LinuxDOProfile{}, fmt.Errorf("linuxdo userinfo request failed with status %d", response.StatusCode)
	}

	var direct model.LinuxDOProfile
	if err := json.Unmarshal(body, &direct); err == nil && direct.Username != "" {
		return direct, nil
	}

	var envelope userEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return model.LinuxDOProfile{}, fmt.Errorf("decode linuxdo userinfo response: %w", err)
	}
	if envelope.User.Username == "" {
		return model.LinuxDOProfile{}, fmt.Errorf("linuxdo userinfo response did not contain username")
	}
	return envelope.User, nil
}
