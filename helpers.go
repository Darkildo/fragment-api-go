package fragment

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// --- cookie helpers ---

// parseCookies splits "k1=v1; k2=v2" into a map.
func parseCookies(raw string) map[string]string {
	out := make(map[string]string)
	for _, pair := range strings.Split(raw, ";") {
		pair = strings.TrimSpace(pair)
		if idx := strings.Index(pair, "="); idx > 0 {
			out[strings.TrimSpace(pair[:idx])] = strings.TrimSpace(pair[idx+1:])
		}
	}
	return out
}

// cookiesToHTTP converts a raw cookie string into []*http.Cookie.
func cookiesToHTTP(raw string) []*http.Cookie {
	m := parseCookies(raw)
	out := make([]*http.Cookie, 0, len(m))
	for k, v := range m {
		out = append(out, &http.Cookie{Name: k, Value: v})
	}
	return out
}

// --- validation helpers ---

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{5,32}$`)

// validateUsername strips a leading '@' and checks format.
// Returns the clean username or an error.
func validateUsername(username string) (string, error) {
	u := strings.TrimPrefix(username, "@")
	if !usernameRe.MatchString(u) {
		return "", fmt.Errorf("invalid username %q: must be 5-32 alphanumeric characters or underscores", u)
	}
	return u, nil
}

// validateAmount checks that amount is in [min, max].
func validateAmount(amount, min, max int) error {
	if amount < min || amount > max {
		return fmt.Errorf("invalid amount %d: must be between %d and %d", amount, min, max)
	}
	return nil
}

// validatePremiumMonths checks that months is 3, 6, or 12.
func validatePremiumMonths(months int) error {
	switch months {
	case 3, 6, 12:
		return nil
	default:
		return fmt.Errorf("invalid premium duration %d months: must be 3, 6, or 12", months)
	}
}

// --- TON conversion ---

// nanoToTON converts nanotons (string) to TON (float64).
func nanoToTON(nano string) (float64, error) {
	n, err := strconv.ParseInt(nano, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse nanotons %q: %w", nano, err)
	}
	return float64(n) / 1e9, nil
}

// tonToNano converts TON (float64) to nanotons (string).
func tonToNano(ton float64) string {
	return strconv.FormatInt(int64(ton*1e9), 10)
}

// --- HTTP defaults ---

// defaultHeaders returns browser-like headers for Fragment.com requests.
func defaultHeaders() http.Header {
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

// extractAvatarURL extracts the image URL from an HTML <img> tag.
func extractAvatarURL(html string) string {
	m := avatarSrcRe.FindStringSubmatch(html)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// extractString looks up a string value in data[key] or data["result"][key].
func extractString(data map[string]interface{}, key string) (string, bool) {
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

// extractTransactionMessage extracts the first transaction message from a
// Fragment API response.
func extractTransactionMsg(data map[string]interface{}) (*transactionMessage, error) {
	result, ok := data["result"].(map[string]interface{})
	if !ok {
		return nil, newPaymentInitiationError("no result in response", nil)
	}

	messages, ok := result["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return nil, newPaymentInitiationError("no transaction messages in response", nil)
	}

	msg, ok := messages[0].(map[string]interface{})
	if !ok {
		return nil, newPaymentInitiationError("invalid transaction message format", nil)
	}

	addr, _ := msg["address"].(string)
	amount, _ := msg["amount"].(string)
	payload, _ := msg["payload"].(string)

	if addr == "" || amount == "" {
		return nil, newPaymentInitiationError("missing address or amount in transaction", nil)
	}

	return &transactionMessage{Address: addr, Amount: amount, Payload: payload}, nil
}
