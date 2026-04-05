// Package httpcore is the low-level HTTP client for the Fragment.com API.
package httpcore

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Darkildo/fragment-api-go/internal/helpers"
	"github.com/Darkildo/fragment-api-go/internal/types"
)

const (
	// BaseURL is the Fragment API endpoint.
	BaseURL = "https://fragment.com/api"
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 15 * time.Second
	// MaxRetries is the maximum number of retry attempts for transient errors.
	MaxRetries = 3
)

// Core is the low-level HTTP client for the Fragment.com API.
type Core struct {
	Client  *http.Client
	Cookies []*http.Cookie
	Hash    string
	Headers http.Header
}

// New creates a configured Core.
func New(cookies, hash string, timeout time.Duration) (*Core, error) {
	if cookies == "" {
		return nil, types.NewAuthenticationError("cookies are required", nil)
	}
	if hash == "" {
		return nil, types.NewAuthenticationError("hash value is required", nil)
	}
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return &Core{
		Client:  &http.Client{Timeout: timeout},
		Cookies: helpers.CookiesToHTTP(cookies),
		Hash:    hash,
		Headers: helpers.DefaultHeaders(),
	}, nil
}

// MakeRequest sends a POST to the Fragment API and returns the parsed JSON.
// Transient network errors are retried up to [MaxRetries] times with
// exponential back-off that respects context cancellation.
func (h *Core) MakeRequest(ctx context.Context, data map[string]string) (map[string]interface{}, error) {
	return h.doWithRetry(ctx, data, 0)
}

func (h *Core) doWithRetry(ctx context.Context, data map[string]string, attempt int) (map[string]interface{}, error) {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}

	reqURL := fmt.Sprintf("%s?hash=%s", BaseURL, h.Hash)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, types.NewNetworkError("create request", 0, err)
	}

	// Copy headers (all current headers are single-value, so clone is safe).
	req.Header = h.Headers.Clone()
	for _, c := range h.Cookies {
		req.AddCookie(c)
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		if attempt < MaxRetries {
			// Exponential backoff: 1s, 2s, 4s — but respect context cancellation.
			delay := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, types.NewNetworkError("request cancelled during retry backoff", 0, ctx.Err())
			case <-time.After(delay):
			}
			return h.doWithRetry(ctx, data, attempt+1)
		}
		return nil, types.NewNetworkError("request failed after retries", 0, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, types.NewRateLimitError(60)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, types.NewNetworkError(fmt.Sprintf("HTTP %d", resp.StatusCode), resp.StatusCode, nil)
	}

	// Limit response body to 10 MB to prevent memory exhaustion.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, types.NewNetworkError("read response body", resp.StatusCode, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, types.NewNetworkError("parse JSON response", resp.StatusCode, err)
	}
	return result, nil
}

// Close releases idle connections.
func (h *Core) Close() {
	h.Client.CloseIdleConnections()
}
