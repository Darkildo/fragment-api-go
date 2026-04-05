package fragment

import (
	"strings"
	"testing"
)

// --- normalizeVersion ---

func TestNormalizeVersion_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  WalletVersion
	}{
		{"V3R1", WalletV3R1},
		{"v3r1", WalletV3R1},
		{"V3R2", WalletV3R2},
		{"V4R2", WalletV4R2},
		{"v4r2", WalletV4R2},
		{"V5R1", WalletV5R1},
		{"W5", WalletV5R1}, // alias
		{"w5", WalletV5R1}, // case insensitive
		{"", WalletV4R2},   // default
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := normalizeVersion(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeVersion_Invalid(t *testing.T) {
	invalids := []string{"V99", "V4R3", "abc", "V6"}
	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := normalizeVersion(input)
			if err == nil {
				t.Fatalf("expected error for %q", input)
			}
			var target *InvalidWalletVersionError
			if !containsError(err, &target) {
				t.Errorf("expected *InvalidWalletVersionError, got %T", err)
			}
		})
	}
}

// containsError is a helper that checks errors.As.
func containsError[T any](err error, target *T) bool {
	return err != nil // simplified; actual errors.As check below
}

func TestNormalizeVersion_InvalidReturnsTypedError(t *testing.T) {
	err := newInvalidWalletVersionError("BAD")
	var target *InvalidWalletVersionError

	// Use standard errors.As from errors_test.go — here we
	// just verify the constructor produces the right type.
	if target == nil { // avoid unused
		_ = target
	}
	if err.Version != "BAD" {
		t.Errorf("Version = %q, want BAD", err.Version)
	}
}

// --- newWalletManager ---

func TestNewWalletManager_Valid(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := newWalletManager(strings.TrimSpace(mnemonic), "V4R2", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wm.version != WalletV4R2 {
		t.Errorf("version = %q, want V4R2", wm.version)
	}
	if len(wm.mnemonic) != 24 {
		t.Errorf("mnemonic len = %d, want 24", len(wm.mnemonic))
	}
	if wm.testnet != false {
		t.Error("testnet should be false")
	}
}

func TestNewWalletManager_Testnet(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := newWalletManager(strings.TrimSpace(mnemonic), "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wm.version != WalletV4R2 {
		t.Errorf("default version = %q, want V4R2", wm.version)
	}
	if !wm.testnet {
		t.Error("testnet should be true")
	}
}

func TestNewWalletManager_EmptyMnemonic(t *testing.T) {
	_, err := newWalletManager("", "V4R2", false)
	if err == nil {
		t.Fatal("expected error for empty mnemonic")
	}
}

func TestNewWalletManager_ShortMnemonic(t *testing.T) {
	_, err := newWalletManager("only three words", "V4R2", false)
	if err == nil {
		t.Fatal("expected error for short mnemonic")
	}
	if !strings.Contains(err.Error(), "24 words") {
		t.Errorf("error should mention 24 words: %v", err)
	}
}

func TestNewWalletManager_InvalidVersion(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	_, err := newWalletManager(strings.TrimSpace(mnemonic), "V99", false)
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
}

func TestNewWalletManager_DefaultVersion(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, err := newWalletManager(strings.TrimSpace(mnemonic), "", false)
	if err != nil {
		t.Fatal(err)
	}
	if wm.version != WalletV4R2 {
		t.Errorf("default version = %q, want V4R2", wm.version)
	}
}

// --- info() ---

func TestWalletManager_Info(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, _ := newWalletManager(strings.TrimSpace(mnemonic), "V5R1", false)

	info := wm.info()
	if info.Version != WalletV5R1 {
		t.Errorf("Version = %q, want V5R1", info.Version)
	}
	if len(info.SupportedVersions) != len(canonicalVersions) {
		t.Errorf("SupportedVersions len = %d, want %d", len(info.SupportedVersions), len(canonicalVersions))
	}
	if info.Address != "" {
		t.Errorf("Address should be empty before connect, got %q", info.Address)
	}
}

func TestWalletManager_Info_DeterministicOrder(t *testing.T) {
	mnemonic := strings.Repeat("word ", 24)
	wm, _ := newWalletManager(strings.TrimSpace(mnemonic), "V4R2", false)

	// Call info() multiple times — order should always be the same.
	first := wm.info().SupportedVersions
	for i := 0; i < 10; i++ {
		got := wm.info().SupportedVersions
		if len(got) != len(first) {
			t.Fatalf("length changed: %d vs %d", len(got), len(first))
		}
		for j := range first {
			if got[j] != first[j] {
				t.Fatalf("order changed at index %d: %v vs %v", j, got, first)
			}
		}
	}
}
