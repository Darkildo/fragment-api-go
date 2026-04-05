// Package utils provides utility functions for the Fragment API client:
// cookie parsing, username/amount validation, and TON unit conversion.
package utils

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// ParseCookies parses a cookie string in the format "k1=v1; k2=v2" into
// a map of key-value pairs.
func ParseCookies(cookieString string) map[string]string {
	cookies := make(map[string]string)
	pairs := strings.Split(cookieString, ";")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		idx := strings.Index(pair, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(pair[:idx])
		value := strings.TrimSpace(pair[idx+1:])
		cookies[key] = value
	}
	return cookies
}

// CookiesToHTTP converts a cookie string into a slice of *http.Cookie
// suitable for use with an HTTP client.
func CookiesToHTTP(cookieString string) []*http.Cookie {
	parsed := ParseCookies(cookieString)
	cookies := make([]*http.Cookie, 0, len(parsed))
	for k, v := range parsed {
		cookies = append(cookies, &http.Cookie{Name: k, Value: v})
	}
	return cookies
}

var usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]{5,32}$`)

// ValidateUsername validates a Telegram username.
// It strips a leading '@' if present and checks that the username is 5-32 characters
// consisting only of letters, digits, and underscores.
// Returns the cleaned username and an error if validation fails.
func ValidateUsername(username string) (string, error) {
	username = strings.TrimPrefix(username, "@")
	if !usernameRegexp.MatchString(username) {
		return "", fmt.Errorf("invalid username %q: must be 5-32 alphanumeric characters or underscores", username)
	}
	return username, nil
}

// ValidateAmount checks that an integer amount is within the given range [minVal, maxVal].
func ValidateAmount(amount, minVal, maxVal int) error {
	if amount < minVal || amount > maxVal {
		return fmt.Errorf("invalid amount %d: must be between %d and %d", amount, minVal, maxVal)
	}
	return nil
}

// ValidatePremiumMonths checks that the premium subscription duration is valid (3, 6, or 12).
func ValidatePremiumMonths(months int) error {
	switch months {
	case 3, 6, 12:
		return nil
	default:
		return fmt.Errorf("invalid premium duration %d months: must be 3, 6, or 12", months)
	}
}

// NanoToTON converts nanotons (as a string) to TON as a float64.
// 1 TON = 1e9 nanotons.
func NanoToTON(nano string) (float64, error) {
	n, err := strconv.ParseInt(nano, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse nanotons %q: %w", nano, err)
	}
	return float64(n) / 1e9, nil
}

// TONToNano converts TON (float64) to nanotons as a string.
func TONToNano(ton float64) string {
	nano := int64(ton * 1e9)
	return strconv.FormatInt(nano, 10)
}

// DefaultHeaders returns the default HTTP headers mimicking a browser request
// to Fragment.com API.
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
