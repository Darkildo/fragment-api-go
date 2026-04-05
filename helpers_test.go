package fragment

import (
	"testing"
)

// --- validateUsername ---

func TestValidateUsername_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"jane_doe", "jane_doe"},
		{"@jane_doe", "jane_doe"},
		{"user1", "user1"},
		{"ABCDE", "ABCDE"},
		{"a_b_c_d_e", "a_b_c_d_e"},
		{"user_1234567890123456789012345", "user_1234567890123456789012345"}, // 32 chars
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := validateUsername(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateUsername_Invalid(t *testing.T) {
	tests := []string{
		"",
		"ab",          // too short
		"abcd",        // 4 chars
		"a b c d e f", // spaces
		"user!name",   // special chars
		"пользователь",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := validateUsername(input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// --- validateAmount ---

func TestValidateAmount_Valid(t *testing.T) {
	tests := []struct {
		amount, min, max int
	}{
		{1, 1, 999999},
		{500, 1, 999999},
		{999999, 1, 999999},
		{3, 3, 12},
		{12, 3, 12},
	}
	for _, tt := range tests {
		if err := validateAmount(tt.amount, tt.min, tt.max); err != nil {
			t.Errorf("validateAmount(%d, %d, %d) = %v, want nil", tt.amount, tt.min, tt.max, err)
		}
	}
}

func TestValidateAmount_Invalid(t *testing.T) {
	tests := []struct {
		amount, min, max int
	}{
		{0, 1, 999999},
		{1000000, 1, 999999},
		{-1, 1, 999999},
		{2, 3, 12},
		{13, 3, 12},
	}
	for _, tt := range tests {
		if err := validateAmount(tt.amount, tt.min, tt.max); err == nil {
			t.Errorf("validateAmount(%d, %d, %d) = nil, want error", tt.amount, tt.min, tt.max)
		}
	}
}

// --- validatePremiumMonths ---

func TestValidatePremiumMonths(t *testing.T) {
	valid := []int{3, 6, 12}
	for _, m := range valid {
		if err := validatePremiumMonths(m); err != nil {
			t.Errorf("validatePremiumMonths(%d) = %v, want nil", m, err)
		}
	}

	invalid := []int{0, 1, 2, 4, 5, 7, 11, 13, 24, -1}
	for _, m := range invalid {
		if err := validatePremiumMonths(m); err == nil {
			t.Errorf("validatePremiumMonths(%d) = nil, want error", m)
		}
	}
}

// --- parseCookies ---

func TestParseCookies(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "standard",
			input: "stel_ssid=abc; stel_token=xyz; stel_dt=-180",
			want:  map[string]string{"stel_ssid": "abc", "stel_token": "xyz", "stel_dt": "-180"},
		},
		{
			name:  "no_spaces",
			input: "a=1;b=2",
			want:  map[string]string{"a": "1", "b": "2"},
		},
		{
			name:  "empty",
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "value_with_equals",
			input: "token=abc=def",
			want:  map[string]string{"token": "abc=def"},
		},
		{
			name:  "trailing_semicolon",
			input: "a=1; ",
			want:  map[string]string{"a": "1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCookies(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d cookies, want %d: %v", len(got), len(tt.want), got)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("cookie %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestCookiesToHTTP(t *testing.T) {
	cookies := cookiesToHTTP("a=1; b=2")
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies, want 2", len(cookies))
	}
	names := map[string]string{}
	for _, c := range cookies {
		names[c.Name] = c.Value
	}
	if names["a"] != "1" || names["b"] != "2" {
		t.Errorf("unexpected cookies: %v", names)
	}
}

// --- nanoToTON ---

func TestNanoToTON(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"0", 0},
		{"1000000000", 1.0},
		{"500000000", 0.5},
		{"1", 1e-9},
		{"1500000000", 1.5},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := nanoToTON(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("nanoToTON(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestNanoToTON_Invalid(t *testing.T) {
	for _, input := range []string{"", "abc", "1.5", "99999999999999999999999"} {
		_, err := nanoToTON(input)
		if err == nil {
			t.Errorf("nanoToTON(%q) = nil error, want error", input)
		}
	}
}

// --- tonToNano / roundToNano ---

func TestTonToNano(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		{1.0, "1000000000"},
		{0.5, "500000000"},
		{1.23, "1230000000"}, // tests math.Round precision
		{0.001, "1000000"},
	}
	for _, tt := range tests {
		got := tonToNano(tt.input)
		if got != tt.want {
			t.Errorf("tonToNano(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRoundToNano_Precision(t *testing.T) {
	// This is the critical float precision test.
	// Without math.Round, 1.23 * 1e9 = 1229999999 (truncation).
	got := roundToNano(1.23)
	if got != 1_230_000_000 {
		t.Errorf("roundToNano(1.23) = %d, want 1230000000", got)
	}

	got = roundToNano(0.1)
	if got != 100_000_000 {
		t.Errorf("roundToNano(0.1) = %d, want 100000000", got)
	}
}

// --- extractAvatarURL ---

func TestExtractAvatarURL(t *testing.T) {
	tests := []struct {
		name string
		html string
		want string
	}{
		{"with_src", `<img src="https://example.com/avatar.jpg" />`, "https://example.com/avatar.jpg"},
		{"no_src", `<div class="photo"></div>`, ""},
		{"empty", "", ""},
		{"multiple", `<img src="first.jpg"><img src="second.jpg">`, "first.jpg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractAvatarURL(tt.html)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// --- extractString ---

func TestExtractString(t *testing.T) {
	data := map[string]interface{}{
		"top_level": "hello",
		"result": map[string]interface{}{
			"nested": "world",
		},
	}

	if v, ok := extractString(data, "top_level"); !ok || v != "hello" {
		t.Errorf("top_level: got %q/%v, want hello/true", v, ok)
	}
	if v, ok := extractString(data, "nested"); !ok || v != "world" {
		t.Errorf("nested: got %q/%v, want world/true", v, ok)
	}
	if _, ok := extractString(data, "missing"); ok {
		t.Error("missing key: expected ok=false")
	}
}

// --- extractTransactionMsg ---

func TestExtractTransactionMsg_Valid(t *testing.T) {
	data := map[string]interface{}{
		"transaction": map[string]interface{}{
			"messages": []interface{}{
				map[string]interface{}{
					"address": "EQ123",
					"amount":  "1000000000",
					"payload": "base64boc",
				},
			},
		},
	}

	msg, err := extractTransactionMsg(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Address != "EQ123" {
		t.Errorf("address: got %q, want EQ123", msg.Address)
	}
	if msg.Amount != "1000000000" {
		t.Errorf("amount: got %q, want 1000000000", msg.Amount)
	}
	if msg.Payload != "base64boc" {
		t.Errorf("payload: got %q, want base64boc", msg.Payload)
	}
}

func TestExtractTransactionMsg_FallbackToResult(t *testing.T) {
	data := map[string]interface{}{
		"result": map[string]interface{}{
			"messages": []interface{}{
				map[string]interface{}{
					"address": "EQfallback",
					"amount":  "500",
					"payload": "",
				},
			},
		},
	}
	msg, err := extractTransactionMsg(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Address != "EQfallback" {
		t.Errorf("address: got %q, want EQfallback", msg.Address)
	}
}

func TestExtractTransactionMsg_NoResult(t *testing.T) {
	_, err := extractTransactionMsg(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing result")
	}
}

func TestExtractTransactionMsg_EmptyMessages(t *testing.T) {
	data := map[string]interface{}{
		"transaction": map[string]interface{}{
			"messages": []interface{}{},
		},
	}
	_, err := extractTransactionMsg(data)
	if err == nil {
		t.Fatal("expected error for empty messages")
	}
}

func TestExtractTransactionMsg_MissingAddress(t *testing.T) {
	data := map[string]interface{}{
		"transaction": map[string]interface{}{
			"messages": []interface{}{
				map[string]interface{}{
					"amount": "100",
				},
			},
		},
	}
	_, err := extractTransactionMsg(data)
	if err == nil {
		t.Fatal("expected error for missing address")
	}
}
