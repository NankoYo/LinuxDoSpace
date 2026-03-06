package linuxdo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"linuxdospace/backend/internal/model"
)

// Client 是 Linux Do OAuth / 用户信息接口的轻量级客户端。
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

// TokenResponse 表示 OAuth 令牌交换成功后的返回结果。
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// userEnvelope 用于兼容某些接口把用户对象包在 `user` 字段中的情况。
type userEnvelope struct {
	User model.LinuxDOProfile `json:"user"`
}

// NewClient 创建 Linux Do 客户端。
func NewClient(clientID string, clientSecret string, redirectURL string, authorizeURL string, tokenURL string, userInfoURL string, scope string, enablePKCE bool) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 20 * time.Second},
		clientID:     strings.TrimSpace(clientID),
		clientSecret: strings.TrimSpace(clientSecret),
		redirectURL:  strings.TrimSpace(redirectURL),
		authorizeURL: strings.TrimSpace(authorizeURL),
		tokenURL:     strings.TrimSpace(tokenURL),
		userInfoURL:  strings.TrimSpace(userInfoURL),
		scope:        strings.TrimSpace(scope),
		enablePKCE:   enablePKCE,
	}
}

// Configured 返回客户端是否具备完成 OAuth 授权码流程的最小配置。
func (c *Client) Configured() bool {
	return c.clientID != "" && c.clientSecret != "" && c.redirectURL != ""
}

// BuildAuthorizationURL 构造 Linux Do OAuth 登录跳转地址。
func (c *Client) BuildAuthorizationURL(state string, codeChallenge string) string {
	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", c.clientID)
	values.Set("redirect_uri", c.redirectURL)
	values.Set("state", state)
	if c.scope != "" {
		values.Set("scope", c.scope)
	}
	if c.enablePKCE && codeChallenge != "" {
		values.Set("code_challenge", codeChallenge)
		values.Set("code_challenge_method", "S256")
	}
	return c.authorizeURL + "?" + values.Encode()
}

// ExchangeCode 使用授权码换取访问令牌。
func (c *Client) ExchangeCode(ctx context.Context, code string, codeVerifier string) (TokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", c.clientID)
	form.Set("client_secret", c.clientSecret)
	form.Set("redirect_uri", c.redirectURL)
	if c.enablePKCE && codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return TokenResponse{}, fmt.Errorf("create linuxdo token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return TokenResponse{}, fmt.Errorf("perform linuxdo token request: %w", err)
	}
	defer response.Body.Close()

	var token TokenResponse
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil {
		return TokenResponse{}, fmt.Errorf("decode linuxdo token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return TokenResponse{}, fmt.Errorf("linuxdo token response did not contain access_token")
	}

	return token, nil
}

// GetCurrentUser 使用访问令牌获取当前 Linux Do 用户信息。
// 这里会先尝试解析直出结构，如果失败，再回退到包裹结构解析。
func (c *Client) GetCurrentUser(ctx context.Context, accessToken string) (model.LinuxDOProfile, error) {
	decodeDirect := func() (model.LinuxDOProfile, error) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
		if err != nil {
			return model.LinuxDOProfile{}, fmt.Errorf("create linuxdo userinfo request: %w", err)
		}
		request.Header.Set("Authorization", "Bearer "+accessToken)

		response, err := c.httpClient.Do(request)
		if err != nil {
			return model.LinuxDOProfile{}, fmt.Errorf("perform linuxdo userinfo request: %w", err)
		}
		defer response.Body.Close()

		var direct model.LinuxDOProfile
		if err := json.NewDecoder(response.Body).Decode(&direct); err != nil {
			return model.LinuxDOProfile{}, err
		}
		return direct, nil
	}

	direct, err := decodeDirect()
	if err == nil && direct.Username != "" {
		return direct, nil
	}

	request, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, c.userInfoURL, nil)
	if reqErr != nil {
		return model.LinuxDOProfile{}, fmt.Errorf("recreate linuxdo userinfo request: %w", reqErr)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return model.LinuxDOProfile{}, fmt.Errorf("repeat linuxdo userinfo request: %w", err)
	}
	defer response.Body.Close()

	var envelope userEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return model.LinuxDOProfile{}, fmt.Errorf("decode linuxdo userinfo response: %w", err)
	}
	if envelope.User.Username == "" {
		return model.LinuxDOProfile{}, fmt.Errorf("linuxdo userinfo response did not contain username")
	}
	return envelope.User, nil
}
