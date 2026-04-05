// Package core provides the low-level HTTP client for communicating with the Fragment.com API.
//
// It handles session management, cookie authentication, request retries,
// and JSON response parsing.
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	fragErrors "github.com/Darkildo/fragment-api-go/errors"
	"github.com/Darkildo/fragment-api-go/utils"
)

const (
	// BaseURL is the Fragment.com API endpoint.
	BaseURL = "https://fragment.com/api"

	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 15 * time.Second

	// MaxRetries is the maximum number of automatic retries on transient errors.
	MaxRetries = 3
)

// Config contains the configuration for creating a Core client.
type Config struct {
	// Cookies is the raw cookie string from Fragment.com
	// (e.g., "stel_ssid=abc; stel_token=xyz; stel_dt=-180; stel_ton_token=uvw").
	Cookies string

	// HashValue is the hash parameter extracted from Fragment.com API requests.
	HashValue string

	// Timeout is the HTTP request timeout. Defaults to DefaultTimeout if zero.
	Timeout time.Duration
}

// Client is the low-level HTTP client for Fragment.com API.
type Client struct {
	httpClient *http.Client
	cookies    []*http.Cookie
	hashValue  string
	headers    http.Header
}

// NewClient creates a new core HTTP client for the Fragment API.
// Returns an error if cookies or hash value are empty.
func NewClient(cfg Config) (*Client, error) {
	if cfg.Cookies == "" {
		return nil, fragErrors.NewAuthenticationError("cookies are required", nil)
	}
	if cfg.HashValue == "" {
		return nil, fragErrors.NewAuthenticationError("hash value is required", nil)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		cookies:    utils.CookiesToHTTP(cfg.Cookies),
		hashValue:  cfg.HashValue,
		headers:    utils.DefaultHeaders(),
	}, nil
}

// APIResponse represents a raw JSON response from the Fragment API.
type APIResponse struct {
	OK     bool                   `json:"ok"`
	Error  string                 `json:"error,omitempty"`
	Result map[string]interface{} `json:"result,omitempty"`
}

// MakeRequest sends a POST request to the Fragment API with the given form data.
// It automatically retries on transient network errors up to MaxRetries times.
//
// The data parameter should contain the form fields (e.g., "method", "query", etc.).
// The hash value is automatically appended as a query parameter.
func (c *Client) MakeRequest(ctx context.Context, data map[string]string) (map[string]interface{}, error) {
	return c.makeRequestWithRetry(ctx, data, 0)
}

func (c *Client) makeRequestWithRetry(ctx context.Context, data map[string]string, retryCount int) (map[string]interface{}, error) {
	// Build form data
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}

	// Build request URL with hash parameter
	reqURL := fmt.Sprintf("%s?hash=%s", BaseURL, c.hashValue)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fragErrors.NewNetworkError("failed to create request", 0, err)
	}

	// Set headers
	for k, vals := range c.headers {
		for _, v := range vals {
			req.Header.Set(k, v)
		}
	}

	// Set cookies
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if retryCount < MaxRetries {
			// Exponential backoff: 1s, 2s, 4s
			time.Sleep(time.Duration(1<<retryCount) * time.Second)
			return c.makeRequestWithRetry(ctx, data, retryCount+1)
		}
		return nil, fragErrors.NewNetworkError("request failed after retries", 0, err)
	}
	defer resp.Body.Close()

	// Check HTTP status
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fragErrors.NewRateLimitError(60)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fragErrors.NewNetworkError(
			fmt.Sprintf("unexpected status code: %d", resp.StatusCode),
			resp.StatusCode, nil,
		)
	}

	// Read and parse body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fragErrors.NewNetworkError("failed to read response body", resp.StatusCode, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fragErrors.NewNetworkError("failed to parse JSON response", resp.StatusCode, err)
	}

	return result, nil
}

// Close releases any resources held by the client.
func (c *Client) Close() {
	c.httpClient.CloseIdleConnections()
}
