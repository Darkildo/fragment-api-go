# Fragment API Go

[![Go 1.21+](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/Darkildo/fragment-api-go)

**Go client for the Fragment.com API — Telegram Stars, Premium subscriptions, and TON transfers.**

Go port of [fragment-api-py](https://github.com/S1qwy/fragment-api-py) (Python v3.2.0).

[README on Russian](README_ru.md)

---

## Installation

```bash
go get github.com/Darkildo/fragment-api-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    fragment "github.com/Darkildo/fragment-api-go"
)

func main() {
    api, err := fragment.New(fragment.Config{
        Cookies:        "stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=...",
        HashValue:      "your_hash_from_network_tab",
        WalletMnemonic: "word1 word2 ... word24",
        WalletAPIKey:   "your_tonapi_key",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer api.Close()

    ctx := context.Background()

    // Look up a user
    user, err := api.GetRecipientStars(ctx, "jane_doe")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("User: %s\n", user.Name)

    // Send 100 Stars anonymously
    result, err := api.BuyStars(ctx, "jane_doe", 100, false)
    if err != nil {
        log.Fatal(err)
    }
    if result.Success {
        fmt.Printf("TX: %s\n", result.TransactionHash)
    }
}
```

## Features

- **Telegram Stars** — send Stars to any Telegram user
- **Premium Gifts** — gift 3/6/12-month Telegram Premium subscriptions
- **TON Ads Top-up** — top up TON Ads accounts
- **Direct TON Transfers** — send TON to any address with optional memo
- **Multi-Wallet Support** — V3R1, V3R2, V4R2 (default), V5R1/W5
- **Sender Visibility** — anonymous or visible payments
- **Automatic Retries** — exponential backoff on transient network errors
- **Typed Errors** — specific error type for each failure scenario
- **Context Support** — all operations accept `context.Context`

## Project Structure

The library follows Go best practice: a single flat package with file-level
organisation. Users import one path and get all public types.

```
fragment-api-go/
  go.mod
  fragment.go    — Client, New(), Config, Close, WalletBalance, WalletInfo
  types.go       — UserInfo, PurchaseResult, TransferResult, WalletBalance
  errors.go      — APIError, AuthenticationError, UserNotFoundError, ...
  recipient.go   — GetRecipientStars, GetRecipientPremium, GetRecipientTON
  stars.go       — BuyStars
  premium.go     — GiftPremium
  topup.go       — TopupTON
  transfer.go    — TransferTON
  purchase.go    — shared purchase flow (unexported)
  core.go        — HTTP transport (unexported)
  wallet.go      — TON wallet management (unexported)
  helpers.go     — validation, parsing, conversion (unexported)
  example/
    main.go      — runnable usage example
```

| File | Contents |
|------|----------|
| `fragment.go` | `Client` struct, `New()` constructor, `Config`, `Close()` |
| `types.go` | `UserInfo`, `PurchaseResult`, `TransferResult`, `WalletBalance` |
| `errors.go` | `APIError` (base), `AuthenticationError`, `UserNotFoundError`, `InvalidAmountError`, `InsufficientBalanceError`, `PaymentInitiationError`, `TransactionError`, `NetworkError`, `RateLimitError`, `WalletError`, `InvalidWalletVersionError` |
| `recipient.go` | `GetRecipientStars`, `GetRecipientPremium`, `GetRecipientTON` |
| `stars.go` | `BuyStars` |
| `premium.go` | `GiftPremium` |
| `topup.go` | `TopupTON` |
| `transfer.go` | `TransferTON` |
| `purchase.go` | Shared purchase flow used by Stars/Premium/TopUp (unexported) |
| `core.go` | `httpCore` — HTTP client, cookies, retries (unexported) |
| `wallet.go` | `walletManager` — TON wallet operations (unexported) |
| `helpers.go` | Cookie parsing, username/amount validation, TON conversion (unexported) |

---

## Setup Guide

### 1. Extract Fragment Cookies

1. Visit [fragment.com](https://fragment.com), press `F12`
2. `Application` > `Cookies` > `fragment.com`
3. Copy: `stel_ssid`, `stel_token`, `stel_dt`, `stel_ton_token`
4. Combine: `"stel_ssid=abc; stel_token=xyz; stel_dt=-180; stel_ton_token=uvw"`

### 2. Get Hash Value

1. DevTools > `Network` tab, refresh fragment.com
2. Find any request to `fragment.com/api`
3. Copy the `hash` query parameter

### 3. Prepare TON Wallet

Export the 24-word mnemonic from your wallet app.

| App | Default Version |
|-----|----------------|
| Tonkeeper | V4R2 |
| MyTonWallet | V4R2 |
| TonHub | V5R1 |

### 4. Get TonAPI Key

1. Visit [tonconsole.com](https://tonconsole.com)
2. Create a project, copy the API Key

---

## API Reference

```go
// Create client (WalletVersion defaults to "V4R2")
api, err := fragment.New(fragment.Config{ ... })
defer api.Close()

// Recipient lookup
user, err := api.GetRecipientStars(ctx, "username")
user, err := api.GetRecipientPremium(ctx, "username")
user, err := api.GetRecipientTON(ctx, "username")

// Purchases
result, err := api.BuyStars(ctx, "username", 100, false)
result, err := api.GiftPremium(ctx, "username", 3, false)
result, err := api.TopupTON(ctx, "username", 10, false)

// Direct transfer
transfer, err := api.TransferTON(ctx, "addr.t.me", 0.5, "memo")

// Wallet
balance, err := api.WalletBalance(ctx)
info := api.WalletInfo()
```

### Parameters

| Method | Parameter | Type | Description |
|--------|-----------|------|-------------|
| `BuyStars` | `username` | `string` | Telegram username (5-32 chars) |
| | `quantity` | `int` | Stars count (1-999999) |
| | `showSender` | `bool` | Show sender identity |
| `GiftPremium` | `months` | `int` | Duration: 3, 6, or 12 |
| `TopupTON` | `amount` | `int` | TON amount (1-999999) |
| `TransferTON` | `toAddress` | `string` | TON address or `user.t.me` |
| | `amountTON` | `float64` | Amount in TON |
| | `memo` | `string` | Transaction comment ("" for none) |

---

## Wallet Versions

| Version | Status | Notes |
|---------|--------|-------|
| **V4R2** | **Default** | Most compatible |
| **V5R1** | Latest | Modern features |
| **W5** | Alias | Maps to V5R1 |
| **V3R2** | Legacy | Older wallets |
| **V3R1** | Legacy | Older wallets |

Case-insensitive: `"v4r2"`, `"V4R2"`, `"V4r2"` all work.

---

## Error Handling

All error types embed `APIError` and implement the `error` interface.
Use `errors.As` to match specific types:

```go
import "errors"

result, err := api.BuyStars(ctx, "user", 100, false)
if err != nil {
    var authErr *fragment.AuthenticationError
    var userErr *fragment.UserNotFoundError
    var balErr  *fragment.InsufficientBalanceError

    switch {
    case errors.As(err, &authErr):
        log.Println("Session expired, update cookies")
    case errors.As(err, &userErr):
        log.Printf("User not found: %s", userErr.Username)
    case errors.As(err, &balErr):
        log.Printf("Need %.6f TON, have %.6f", balErr.Required, balErr.Current)
    default:
        log.Printf("Error: %v", err)
    }
}
```

### Error Hierarchy

```
APIError (base)
├── AuthenticationError
├── UserNotFoundError
├── InvalidAmountError
├── InsufficientBalanceError
├── PaymentInitiationError
├── TransactionError
├── NetworkError
├── RateLimitError
└── WalletError
    └── InvalidWalletVersionError
```

---

## Development Status

This is a **structural skeleton** (v1.0.0). The HTTP client, types, errors,
validation, and full high-level API are implemented and compile cleanly.

**Requires implementation:** the `wallet.go` methods (`getBalance`,
`sendTransaction`, `transferTON`) return stubs. Integrate a Go TON SDK:

- [xssnick/tonutils-go](https://github.com/xssnick/tonutils-go)
- [tonkeeper/tongo](https://github.com/tonkeeper/tongo)

Each stub method contains detailed TODO pseudocode.

---

## License

MIT. Based on [fragment-api-py](https://github.com/S1qwy/fragment-api-py) by [S1qwy](https://github.com/S1qwy).
