package fragment

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
)

// --- New() validation ---

func TestNew_MissingCookies(t *testing.T) {
	_, err := New(Config{
		HashValue:      "h",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
	})
	if err == nil {
		t.Fatal("expected error for missing cookies")
	}
	var target *AuthenticationError
	if !errors.As(err, &target) {
		t.Errorf("expected AuthenticationError, got: %v", err)
	}
}

func TestNew_MissingHash(t *testing.T) {
	_, err := New(Config{
		Cookies:        "a=1",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
	})
	if err == nil {
		t.Fatal("expected error for missing hash")
	}
	var target *AuthenticationError
	if !errors.As(err, &target) {
		t.Errorf("expected AuthenticationError, got: %v", err)
	}
}

func TestNew_MissingMnemonic(t *testing.T) {
	_, err := New(Config{
		Cookies:   "a=1",
		HashValue: "h",
	})
	if err == nil {
		t.Fatal("expected error for missing mnemonic")
	}
	var target *WalletError
	if !errors.As(err, &target) {
		t.Errorf("expected WalletError, got: %v", err)
	}
}

func TestNew_InvalidMnemonic(t *testing.T) {
	_, err := New(Config{
		Cookies:        "a=1",
		HashValue:      "h",
		WalletMnemonic: "only three words",
	})
	if err == nil {
		t.Fatal("expected error for short mnemonic")
	}
}

func TestNew_InvalidWalletVersion(t *testing.T) {
	_, err := New(Config{
		Cookies:        "a=1",
		HashValue:      "h",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
		WalletVersion:  "INVALID",
	})
	if err == nil {
		t.Fatal("expected error for invalid version")
	}
	var target *InvalidWalletVersionError
	if !errors.As(err, &target) {
		t.Errorf("expected InvalidWalletVersionError, got: %v", err)
	}
}

func TestNew_Valid(t *testing.T) {
	client, err := New(Config{
		Cookies:        "stel_ssid=abc; stel_token=xyz",
		HashValue:      "hash123",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer client.Close()

	if client.core == nil {
		t.Error("core should not be nil")
	}
	if client.wallet == nil {
		t.Error("wallet should not be nil")
	}
}

func TestNew_WithLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client, err := New(Config{
		Cookies:        "a=1",
		HashValue:      "h",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
		Logger:         logger,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if client.log != logger {
		t.Error("logger should be set")
	}
}

func TestNew_NilLogger_UsesDiscard(t *testing.T) {
	client, err := New(Config{
		Cookies:        "a=1",
		HashValue:      "h",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	// Verify the discard handler doesn't panic.
	client.log.Info("should not appear anywhere")
	client.log.Debug("also silent")
}

func TestNew_DefaultVersion(t *testing.T) {
	client, err := New(Config{
		Cookies:        "a=1",
		HashValue:      "h",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	info := client.WalletInfo()
	if info.Version != WalletV4R2 {
		t.Errorf("default version = %q, want V4R2", info.Version)
	}
}

func TestNew_Testnet(t *testing.T) {
	client, err := New(Config{
		Cookies:        "a=1",
		HashValue:      "h",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
		Testnet:        true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if !client.wallet.Testnet {
		t.Error("testnet should be true")
	}
}

// --- Close() idempotency ---

func TestClose_Multiple(t *testing.T) {
	client, err := New(Config{
		Cookies:        "a=1",
		HashValue:      "h",
		WalletMnemonic: strings.TrimSpace(strings.Repeat("word ", 24)),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Should not panic.
	client.Close()
	client.Close()
	client.Close()
}

// --- discardHandler ---

func TestDiscardHandler(t *testing.T) {
	h := discardHandler{}

	if h.Enabled(context.Background(), slog.LevelDebug) {
		t.Error("Enabled should return false")
	}
	if h.Enabled(context.Background(), slog.LevelError) {
		t.Error("Enabled should return false even for Error level")
	}
	if err := h.Handle(context.Background(), slog.Record{}); err != nil {
		t.Errorf("Handle should return nil, got %v", err)
	}

	h2 := h.WithAttrs([]slog.Attr{slog.String("key", "val")})
	if _, ok := h2.(discardHandler); !ok {
		t.Errorf("WithAttrs should return discardHandler, got %T", h2)
	}

	h3 := h.WithGroup("group")
	if _, ok := h3.(discardHandler); !ok {
		t.Errorf("WithGroup should return discardHandler, got %T", h3)
	}
}
