package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// apiBaseURL 是 Cloudflare v4 API 的固定基础地址。
const apiBaseURL = "https://api.cloudflare.com/client/v4"

// Client 是一个轻量级 Cloudflare API 客户端。
// 我们使用标准库实现，避免在核心链路里引入不必要的外部框架。
type Client struct {
	httpClient *http.Client
	apiToken   string
}

// Zone 表示 Cloudflare 返回的 Zone 简要信息。
type Zone struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DNSRecord 表示 Cloudflare DNS 记录。
type DNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Proxied  bool   `json:"proxied"`
	Comment  string `json:"comment"`
	Priority *int   `json:"priority,omitempty"`
}

// CreateDNSRecordInput 表示创建 DNS 记录时需要的字段。
type CreateDNSRecordInput struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Proxied  bool   `json:"proxied"`
	Comment  string `json:"comment,omitempty"`
	Priority *int   `json:"priority,omitempty"`
}

// UpdateDNSRecordInput 表示更新 DNS 记录时需要的字段。
type UpdateDNSRecordInput = CreateDNSRecordInput

// listResponse 是 Cloudflare 分页结果的通用包装。
type listResponse[T any] struct {
	Result  []T  `json:"result"`
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
	ResultInfo struct {
		Page       int `json:"page"`
		PerPage    int `json:"per_page"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

// objectResponse 是 Cloudflare 单对象结果的通用包装。
type objectResponse[T any] struct {
	Result  T    `json:"result"`
	Success bool `json:"success"`
	Errors  []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// NewClient 创建一个 Cloudflare API 客户端。
func NewClient(apiToken string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 20 * time.Second},
		apiToken:   strings.TrimSpace(apiToken),
	}
}

// ResolveZoneID 通过根域名查找对应的 Zone ID。
func (c *Client) ResolveZoneID(ctx context.Context, rootDomain string) (string, error) {
	if strings.TrimSpace(rootDomain) == "" {
		return "", fmt.Errorf("root domain is required")
	}

	values := url.Values{}
	values.Set("name", rootDomain)

	var response listResponse[Zone]
	if err := c.doJSON(ctx, http.MethodGet, apiBaseURL+"/zones?"+values.Encode(), nil, &response); err != nil {
		return "", err
	}

	if len(response.Result) == 0 {
		return "", fmt.Errorf("zone %q not found", rootDomain)
	}

	return response.Result[0].ID, nil
}

// ListAllDNSRecords 把指定 Zone 下的所有 DNS 记录全部拉取回来。
// 该方法会自动分页，适合做命名空间冲突检查和权限校验。
func (c *Client) ListAllDNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error) {
	if strings.TrimSpace(zoneID) == "" {
		return nil, fmt.Errorf("zone id is required")
	}

	all := make([]DNSRecord, 0, 32)
	page := 1

	for {
		values := url.Values{}
		values.Set("page", strconv.Itoa(page))
		values.Set("per_page", "100")

		var response listResponse[DNSRecord]
		endpoint := fmt.Sprintf("%s/zones/%s/dns_records?%s", apiBaseURL, zoneID, values.Encode())
		if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
			return nil, err
		}

		all = append(all, response.Result...)
		if page >= response.ResultInfo.TotalPages || response.ResultInfo.TotalPages == 0 {
			break
		}
		page++
	}

	return all, nil
}

// GetDNSRecord 根据记录 ID 获取一条具体 DNS 记录。
func (c *Client) GetDNSRecord(ctx context.Context, zoneID string, recordID string) (DNSRecord, error) {
	var response objectResponse[DNSRecord]
	endpoint := fmt.Sprintf("%s/zones/%s/dns_records/%s", apiBaseURL, zoneID, recordID)
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return DNSRecord{}, err
	}
	return response.Result, nil
}

// CreateDNSRecord 在 Cloudflare 中创建一条 DNS 记录。
func (c *Client) CreateDNSRecord(ctx context.Context, zoneID string, input CreateDNSRecordInput) (DNSRecord, error) {
	var response objectResponse[DNSRecord]
	endpoint := fmt.Sprintf("%s/zones/%s/dns_records", apiBaseURL, zoneID)
	if err := c.doJSON(ctx, http.MethodPost, endpoint, input, &response); err != nil {
		return DNSRecord{}, err
	}
	return response.Result, nil
}

// UpdateDNSRecord 在 Cloudflare 中完整更新一条 DNS 记录。
func (c *Client) UpdateDNSRecord(ctx context.Context, zoneID string, recordID string, input UpdateDNSRecordInput) (DNSRecord, error) {
	var response objectResponse[DNSRecord]
	endpoint := fmt.Sprintf("%s/zones/%s/dns_records/%s", apiBaseURL, zoneID, recordID)
	if err := c.doJSON(ctx, http.MethodPut, endpoint, input, &response); err != nil {
		return DNSRecord{}, err
	}
	return response.Result, nil
}

// DeleteDNSRecord 删除一条 DNS 记录。
func (c *Client) DeleteDNSRecord(ctx context.Context, zoneID string, recordID string) error {
	var response objectResponse[map[string]any]
	endpoint := fmt.Sprintf("%s/zones/%s/dns_records/%s", apiBaseURL, zoneID, recordID)
	return c.doJSON(ctx, http.MethodDelete, endpoint, nil, &response)
}

// doJSON 负责统一处理 Cloudflare API 的认证、JSON 编码与错误判断。
func (c *Client) doJSON(ctx context.Context, method string, endpoint string, requestBody any, responseBody any) error {
	var bodyReader *bytes.Reader
	if requestBody == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal cloudflare request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("create cloudflare request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+c.apiToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("perform cloudflare request: %w", err)
	}
	defer response.Body.Close()

	if err := json.NewDecoder(response.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("decode cloudflare response: %w", err)
	}

	switch typed := responseBody.(type) {
	case *listResponse[Zone]:
		if !typed.Success {
			return fmt.Errorf("cloudflare api error: %s", firstCloudflareError(typed.Errors))
		}
	case *listResponse[DNSRecord]:
		if !typed.Success {
			return fmt.Errorf("cloudflare api error: %s", firstCloudflareError(typed.Errors))
		}
	case *objectResponse[DNSRecord]:
		if !typed.Success {
			return fmt.Errorf("cloudflare api error: %s", firstCloudflareError(typed.Errors))
		}
	case *objectResponse[map[string]any]:
		if !typed.Success {
			return fmt.Errorf("cloudflare api error: %s", firstCloudflareError(typed.Errors))
		}
	default:
		return fmt.Errorf("unsupported cloudflare response type")
	}

	return nil
}

// firstCloudflareError 把 Cloudflare 错误列表收敛成一条便于日志和 API 返回的消息。
func firstCloudflareError(errors []struct {
	Message string `json:"message"`
}) string {
	if len(errors) == 0 {
		return "unknown error"
	}
	if strings.TrimSpace(errors[0].Message) == "" {
		return "unknown error"
	}
	return errors[0].Message
}
