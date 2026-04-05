// Package fragment provides a Go client for the Fragment.com API.
//
// Fragment.com is a platform for purchasing Telegram Stars, Premium subscriptions,
// and managing TON-based payments. This library is a Go port of the Python
// fragment-api-py library (https://github.com/S1qwy/fragment-api-py).
//
// # Quick Start
//
//	api, err := fragment.New(fragment.Config{
//	    Cookies:        "stel_ssid=...; stel_token=...",
//	    HashValue:      "your_hash",
//	    WalletMnemonic: "word1 word2 ... word24",
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer api.Close()
//
//	result, err := api.BuyStars(ctx, "jane_doe", 100, false)
//
// # Architecture
//
// All public types and methods live in this package. Internal details
// (HTTP transport, HTML parsing, wallet signing) are in internal/ packages.
package fragment

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Darkildo/fragment-api-go/internal/httpcore"
	"github.com/Darkildo/fragment-api-go/internal/tonwallet"
)

// Version is the library version. Go port of fragment-api-py v3.2.0.
const Version = "1.3.0"

// Config contains all parameters needed to create a [Client].
type Config struct {
	// Cookies is the raw cookie string from a Fragment.com browser session.
	// Required cookies: stel_ssid, stel_token, stel_dt, stel_ton_token.
	Cookies string

	// HashValue is the API hash parameter extracted from Fragment.com
	// network requests (DevTools -> Network -> any /api request -> "hash" query param).
	HashValue string

	// WalletMnemonic is the 24-word TON wallet seed phrase (space-separated).
	WalletMnemonic string

	// WalletVersion is the TON wallet contract version.
	// Supported: WalletV3R1, WalletV3R2, WalletV4R2 (default), WalletV5R1, WalletW5.
	// Case-insensitive string is also accepted (e.g. "v4r2").
	// Empty defaults to WalletV4R2.
	WalletVersion string

	// Testnet connects to the TON testnet instead of mainnet.
	// Default is false (mainnet).
	Testnet bool

	// Timeout is the HTTP request timeout for Fragment.com API calls.
	// Zero means 15 seconds.
	Timeout time.Duration

	// Logger is an optional structured logger ([log/slog.Logger]).
	// If nil, a no-op logger is used (no output).
	Logger *slog.Logger
}

// Client is the main entry point for the Fragment.com API.
//
// Create one with [New]. The underlying http.Client is safe for concurrent
// use; however, wallet operations (balance, send) use a shared LiteClient
// connection pool and do not guard against concurrent nonce conflicts.
// If you need to send multiple transactions concurrently, use separate
// Client instances.
type Client struct {
	core   *httpcore.Core
	wallet *tonwallet.Manager
	log    *slog.Logger
}

// New creates a new Fragment API [Client].
//
// It validates the configuration, initialises the HTTP transport and wallet
// manager, and returns a ready-to-use client. The connection to the TON
// network is established lazily on the first wallet operation.
func New(cfg Config) (*Client, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.New(discardHandler{})
	}

	core, err := httpcore.New(cfg.Cookies, cfg.HashValue, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("fragment: %w", err)
	}

	wm, err := tonwallet.New(cfg.WalletMnemonic, cfg.WalletVersion, cfg.Testnet)
	if err != nil {
		core.Close()
		return nil, fmt.Errorf("fragment: %w", err)
	}

	logger.Info("fragment client created",
		"wallet_version", string(wm.Version),
		"testnet", cfg.Testnet,
	)

	return &Client{core: core, wallet: wm, log: logger}, nil
}

// Close releases all resources held by the client, including the
// TON LiteClient connection pool. Safe to call multiple times.
func (c *Client) Close() {
	if c.core != nil {
		c.core.Close()
	}
	if c.wallet != nil && c.wallet.Pool != nil {
		c.wallet.Pool.Stop()
	}
	c.log.Debug("fragment client closed")
}

// WalletBalance retrieves the current TON wallet balance and metadata.
// The first call triggers the LiteClient connection to the TON network.
func (c *Client) WalletBalance(ctx context.Context) (*WalletBalance, error) {
	return c.wallet.GetBalance(ctx)
}

// WalletInfo returns metadata about the wallet configuration.
func (c *Client) WalletInfo() WalletInfo {
	return c.wallet.Info()
}

// discardHandler is a [slog.Handler] that discards all log records.
type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (d discardHandler) WithAttrs([]slog.Attr) slog.Handler      { return d }
func (d discardHandler) WithGroup(string) slog.Handler           { return d }
