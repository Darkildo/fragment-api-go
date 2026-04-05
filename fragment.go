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
//	    WalletAPIKey:   "your-tonapi-key",
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
// All public types and methods live in this single package. Internal details
// (HTTP transport, HTML parsing, wallet signing) are unexported.
//
// Files:
//   - fragment.go  — Client, New(), Config, Close
//   - types.go     — UserInfo, PurchaseResult, TransferResult, WalletBalance
//   - errors.go    — Error type hierarchy
//   - stars.go     — BuyStars
//   - premium.go   — GiftPremium
//   - topup.go     — TopupTON
//   - transfer.go  — TransferTON
//   - recipient.go — GetRecipientStars/Premium/TON
//   - wallet.go    — walletManager (unexported)
//   - core.go      — httpCore (unexported)
//   - helpers.go   — validation, parsing, conversion utilities (unexported)
package fragment

import (
	"context"
	"fmt"
	"time"
)

// Version is the library version. Go port of fragment-api-py v3.2.0.
const Version = "1.0.0"

// Config contains all parameters needed to create a [Client].
type Config struct {
	// Cookies is the raw cookie string from a Fragment.com browser session.
	// Required cookies: stel_ssid, stel_token, stel_dt, stel_ton_token.
	Cookies string

	// HashValue is the API hash parameter extracted from Fragment.com
	// network requests (DevTools → Network → any /api request → "hash" query param).
	HashValue string

	// WalletMnemonic is the 24-word TON wallet seed phrase (space-separated).
	WalletMnemonic string

	// WalletAPIKey is the TonAPI key from https://tonconsole.com.
	WalletAPIKey string

	// WalletVersion is the TON wallet contract version.
	// Supported: "V3R1", "V3R2", "V4R2" (default), "V5R1", "W5".
	// Case-insensitive. Empty defaults to "V4R2".
	WalletVersion string

	// Timeout is the HTTP request timeout. Zero means 15 seconds.
	Timeout time.Duration
}

// Client is the main entry point for the Fragment.com API.
//
// Create one with [New]. All methods are safe for concurrent use.
type Client struct {
	core   *httpCore
	wallet *walletManager
}

// New creates a new Fragment API [Client].
//
// It validates the configuration, initialises the HTTP transport and wallet
// manager, and returns a ready-to-use client.
func New(cfg Config) (*Client, error) {
	core, err := newHTTPCore(cfg.Cookies, cfg.HashValue, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("fragment: %w", err)
	}

	wm, err := newWalletManager(cfg.WalletMnemonic, cfg.WalletAPIKey, cfg.WalletVersion)
	if err != nil {
		core.close()
		return nil, fmt.Errorf("fragment: %w", err)
	}

	return &Client{core: core, wallet: wm}, nil
}

// Close releases all resources held by the client.
// It is safe to call Close multiple times.
func (c *Client) Close() {
	if c.core != nil {
		c.core.close()
	}
}

// WalletBalance retrieves the current TON wallet balance and metadata.
func (c *Client) WalletBalance(ctx context.Context) (*WalletBalance, error) {
	return c.wallet.getBalance(ctx)
}

// WalletInfo returns metadata about the wallet configuration:
// current version, supported versions, and the version alias mapping.
func (c *Client) WalletInfo() map[string]interface{} {
	return c.wallet.info()
}
