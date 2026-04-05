package fragment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	baseURL        = "https://fragment.com/api"
	defaultTimeout = 15 * time.Second
	maxRetries     = 3
)

// httpCore is the low-level HTTP client for the Fragment.com API.
// It is unexported; users interact with the public [Client] type.
type httpCore struct {
	client  *http.Client
	cookies []*http.Cookie
	hash    string
	headers http.Header
}

// newHTTPCore creates a configured httpCore.
func newHTTPCore(cookies, hash string, timeout time.Duration) (*httpCore, error) {
	if cookies == "" {
		return nil, newAuthenticationError("cookies are required", nil)
	}
	if hash == "" {
		return nil, newAuthenticationError("hash value is required", nil)
	}
	if timeout == 0 {
		timeout = defaultTimeout
	}
	return &httpCore{
		client:  &http.Client{Timeout: timeout},
		cookies: cookiesToHTTP(cookies),
		hash:    hash,
		headers: defaultHeaders(),
	}, nil
}

// makeRequest sends a POST to the Fragment API and returns the parsed JSON.
// Transient network errors are retried up to [maxRetries] times with
// exponential back-off that respects context cancellation.
func (h *httpCore) makeRequest(ctx context.Context, data map[string]string) (map[string]interface{}, error) {
	return h.doWithRetry(ctx, data, 0)
}

func (h *httpCore) doWithRetry(ctx context.Context, data map[string]string, attempt int) (map[string]interface{}, error) {
	form := url.Values{}
	for k, v := range data {
		form.Set(k, v)
	}

	reqURL := fmt.Sprintf("%s?hash=%s", baseURL, h.hash)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, newNetworkError("create request", 0, err)
	}

	// Copy headers (all current headers are single-value, so clone is safe).
	req.Header = h.headers.Clone()
	for _, c := range h.cookies {
		req.AddCookie(c)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		if attempt < maxRetries {
			// Exponential backoff: 1s, 2s, 4s — but respect context cancellation.
			delay := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, newNetworkError("request cancelled during retry backoff", 0, ctx.Err())
			case <-time.After(delay):
			}
			return h.doWithRetry(ctx, data, attempt+1)
		}
		return nil, newNetworkError("request failed after retries", 0, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, newRateLimitError(60)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newNetworkError(fmt.Sprintf("HTTP %d", resp.StatusCode), resp.StatusCode, nil)
	}

	// Limit response body to 10 MB to prevent memory exhaustion.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, newNetworkError("read response body", resp.StatusCode, err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, newNetworkError("parse JSON response", resp.StatusCode, err)
	}
	return result, nil
}

// close releases idle connections.
func (h *httpCore) close() {
	h.client.CloseIdleConnections()
}
