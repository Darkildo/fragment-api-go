// Package helpers provides validation, parsing, and conversion utilities
// used internally by the fragment library.
package helpers

import (
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/Darkildo/fragment-api-go/internal/types"
)

// --- cookie helpers ---

// ParseCookies splits "k1=v1; k2=v2" into a map.
func ParseCookies(raw string) map[string]string {
	out := make(map[string]string)
	for _, pair := range strings.Split(raw, ";") {
		pair = strings.TrimSpace(pair)
		if idx := strings.Index(pair, "="); idx > 0 {
			out[strings.TrimSpace(pair[:idx])] = strings.TrimSpace(pair[idx+1:])
		}
	}
	return out
}

// CookiesToHTTP converts a raw cookie string into []*http.Cookie.
func CookiesToHTTP(raw string) []*http.Cookie {
	m := ParseCookies(raw)
	out := make([]*http.Cookie, 0, len(m))
	for k, v := range m {
		out = append(out, &http.Cookie{Name: k, Value: v})
	}
	return out
}

// --- validation helpers ---

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{5,32}$`)

// ValidateUsername strips a leading '@' and checks format.
// Returns the clean username or an error.
func ValidateUsername(username string) (string, error) {
	u := strings.TrimPrefix(username, "@")
	if !usernameRe.MatchString(u) {
		return "", fmt.Errorf("invalid username %q: must be 5-32 alphanumeric characters or underscores", u)
	}
	return u, nil
}

// ValidateAmount checks that amount is in [min, max].
func ValidateAmount(amount, min, max int) error {
	if amount < min || amount > max {
		return fmt.Errorf("invalid amount %d: must be between %d and %d", amount, min, max)
	}
	return nil
}

// ValidatePremiumMonths checks that months is 3, 6, or 12.
func ValidatePremiumMonths(months int) error {
	switch months {
	case 3, 6, 12:
		return nil
	default:
		return fmt.Errorf("invalid premium duration %d months: must be 3, 6, or 12", months)
	}
}

// --- TON conversion ---

// NanoToTON converts nanotons (string) to TON (float64).
func NanoToTON(nano string) (float64, error) {
	n, err := strconv.ParseInt(nano, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse nanotons %q: %w", nano, err)
	}
	return float64(n) / 1e9, nil
}

// TonToNano converts TON (float64) to nanotons (string).
// Uses math.Round to avoid floating-point truncation issues.
func TonToNano(ton float64) string {
	return strconv.FormatInt(RoundToNano(ton), 10)
}

// RoundToNano converts TON to nanotons as int64, rounding to the nearest
// integer to avoid floating-point precision loss (e.g. 1.23 * 1e9 = 1230000000).
func RoundToNano(ton float64) int64 {
	return int64(math.Round(ton * 1e9))
}

// --- HTTP defaults ---

// DefaultHeaders returns browser-like headers for Fragment.com requests.
func DefaultHeaders() http.Header {
	h := http.Header{}
	h.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	h.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	h.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	h.Set("Origin", "https://fragment.com")
	h.Set("Referer", "https://fragment.com/")
	h.Set("X-Requested-With", "XMLHttpRequest")
	return h
}

// --- response parsing ---

var avatarSrcRe = regexp.MustCompile(`src="([^"]+)"`)

// ExtractAvatarURL extracts the image URL from an HTML <img> tag.
func ExtractAvatarURL(html string) string {
	m := avatarSrcRe.FindStringSubmatch(html)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// ExtractString looks up a string value in data[key] or data["result"][key].
func ExtractString(data map[string]interface{}, key string) (string, bool) {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s, true
		}
	}
	if result, ok := data["result"].(map[string]interface{}); ok {
		if v, ok := result[key]; ok {
			if s, ok := v.(string); ok {
				return s, true
			}
		}
	}
	return "", false
}

// ExtractTransactionMsg extracts the first transaction message from a
// Fragment API response.
//
// Fragment API response format:
//
//	{"transaction": {"messages": [{"address": "...", "amount": "...", "payload": "..."}]}}
func ExtractTransactionMsg(data map[string]interface{}) (*types.TransactionMessage, error) {
	// Try "transaction" key (Fragment API standard).
	tx, ok := data["transaction"].(map[string]interface{})
	if !ok {
		// Fallback: try "result" key for compatibility.
		tx, ok = data["result"].(map[string]interface{})
		if !ok {
			return nil, types.NewPaymentInitiationError("no 'transaction' or 'result' in response", nil)
		}
	}

	messages, ok := tx["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return nil, types.NewPaymentInitiationError("no transaction messages in response", nil)
	}

	msg, ok := messages[0].(map[string]interface{})
	if !ok {
		return nil, types.NewPaymentInitiationError("invalid transaction message format", nil)
	}

	addr, _ := msg["address"].(string)
	amount, _ := msg["amount"].(string)
	payload, _ := msg["payload"].(string)

	if addr == "" || amount == "" {
		return nil, types.NewPaymentInitiationError("missing address or amount in transaction", nil)
	}

	return &types.TransactionMessage{Address: addr, Amount: amount, Payload: payload}, nil
}
