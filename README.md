# Fragment API Go Library

[![Go 1.21+](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/Darkildo/fragment-api-go)

**Go client library for the Fragment.com API — Telegram Stars, Premium subscriptions, and TON transfers.**

Go port of [fragment-api-py](https://github.com/S1qwy/fragment-api-py) (Python v3.2.0).

[README на русском](README_ru.md)

---

## Table of Contents

- [Features](#features)
- [Project Structure](#project-structure)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Setup Guide](#setup-guide)
- [API Reference](#api-reference)
- [Wallet Versions](#wallet-versions)
- [Error Handling](#error-handling)
- [Data Models](#data-models)
- [Development Status](#development-status)
- [License](#license)

---

## Features

- **Telegram Stars** — send Stars to any Telegram user
- **Premium Gifts** — gift 3/6/12-month Telegram Premium subscriptions
- **TON Ads Top-up** — top up TON Ads accounts
- **Direct TON Transfers** — send TON to any address with optional memo
- **Wallet Management** — balance queries, multi-version wallet support (V3R1, V3R2, V4R2, V5R1/W5)
- **Sender Visibility** — anonymous or visible payments
- **Automatic Retries** — exponential backoff on transient network errors
- **Typed Errors** — specific error types for each failure scenario
- **Context Support** — all operations accept `context.Context` for cancellation/timeouts

---

## Project Structure

```
fragment-api-go/
├── fragment.go          # Root package — version constant, package docs
├── go.mod               # Go module definition
│
├── client/
│   └── client.go        # High-level FragmentAPI client (main entry point)
│
├── core/
│   └── core.go          # Low-level HTTP client for Fragment.com API
│
├── wallet/
│   └── wallet.go        # TON wallet management (balance, tx, transfers)
│
├── models/
│   └── models.go        # Data structures: UserInfo, PurchaseResult, etc.
│
├── errors/
│   └── errors.go        # Error types hierarchy
│
├── utils/
│   └── utils.go         # Utilities: cookie parsing, validation, TON conversion
│
├── example/
│   └── main.go          # Usage example
│
├── README.md            # Documentation (English)
└── README_ru.md         # Documentation (Russian)
```

### Package Overview

| Package  | Description |
|----------|-------------|
| `client` | **Main entry point.** `FragmentAPI` struct with `BuyStars`, `GiftPremium`, `TopupTON`, `TransferTON`, `GetWalletBalance` |
| `core`   | HTTP client: session management, cookies, retry logic, JSON parsing |
| `wallet` | TON blockchain integration: wallet initialization, balance, transactions (skeleton — requires TON SDK) |
| `models` | Data types: `UserInfo`, `PurchaseResult`, `TransferResult`, `WalletBalance`, `TransactionMessage` |
| `errors` | Typed errors: `AuthenticationError`, `UserNotFoundError`, `InsufficientBalanceError`, etc. |
| `utils`  | Cookie parsing, username validation, amount validation, TON/nano conversion |

---

## Installation

```bash
go get github.com/Darkildo/fragment-api-go
```

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/Darkildo/fragment-api-go/client"
)

func main() {
    api, err := client.New(client.Config{
        Cookies:        "stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=...",
        HashValue:      "your_hash_from_network_tab",
        WalletMnemonic: "word1 word2 ... word24",
        WalletAPIKey:   "your_tonapi_key",
        WalletVersion:  "V4R2",
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

    // Send 100 Stars (anonymous)
    result, err := api.BuyStars(ctx, "jane_doe", 100, false)
    if err != nil {
        log.Fatal(err)
    }
    if result.Success {
        fmt.Printf("TX: %s\n", result.TransactionHash)
    }

    // Send Stars with visible sender
    result, err = api.BuyStars(ctx, "jane_doe", 50, true)

    // Gift Premium (3 months, anonymous)
    premResult, err := api.GiftPremium(ctx, "jane_doe", 3, false)

    // Direct TON transfer with memo
    transfer, err := api.TransferTON(ctx, "recipient.t.me", 0.5, "Payment")

    // Check wallet balance
    balance, err := api.GetWalletBalance(ctx)
    if err == nil {
        fmt.Printf("Balance: %.6f TON\n", balance.BalanceTON)
    }
}
```

---

## Setup Guide

### 1. Extract Fragment Cookies

1. Visit [fragment.com](https://fragment.com) in your browser
2. Press `F12` to open Developer Tools
3. Go to `Application` > `Cookies` > `fragment.com`
4. Copy these cookies:

| Cookie | Purpose |
|--------|---------|
| `stel_ssid` | Session ID |
| `stel_token` | Authentication token |
| `stel_dt` | Timezone offset |
| `stel_ton_token` | TON-specific token |

5. Combine into a single string:
```
stel_ssid=abc123; stel_token=xyz789; stel_dt=-180; stel_ton_token=uvw012
```

### 2. Get Hash Value

1. Keep DevTools open, go to `Network` tab
2. Refresh fragment.com
3. Find requests to `fragment.com/api`
4. Copy the `hash` query parameter value

### 3. Prepare TON Wallet

Export your 24-word mnemonic from your TON wallet app (Tonkeeper, MyTonWallet, TonHub).

Default wallet versions:
- **Tonkeeper** — V4R2
- **MyTonWallet** — V4R2
- **TonHub** — V5R1

### 4. Get TonAPI Key

1. Visit [tonconsole.com](https://tonconsole.com)
2. Create a project
3. Copy the API Key

### 5. Environment Variables

```bash
export FRAGMENT_COOKIES="stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=..."
export FRAGMENT_HASH="abc123def456..."
export WALLET_MNEMONIC="word1 word2 ... word24"
export WALLET_API_KEY="your_key"
export WALLET_VERSION="V4R2"
```

---

## API Reference

### Client Methods

```go
// Create client
api, err := client.New(cfg client.Config) (*client.FragmentAPI, error)

// Recipient lookup
user, err := api.GetRecipientStars(ctx, username)   // -> *models.UserInfo
user, err := api.GetRecipientPremium(ctx, username)  // -> *models.UserInfo
user, err := api.GetRecipientTON(ctx, username)      // -> *models.UserInfo

// Purchases
result, err := api.BuyStars(ctx, username, quantity, showSender)     // -> *models.PurchaseResult
result, err := api.GiftPremium(ctx, username, months, showSender)    // -> *models.PurchaseResult
result, err := api.TopupTON(ctx, username, amount, showSender)       // -> *models.PurchaseResult

// Direct transfer
transfer, err := api.TransferTON(ctx, toAddress, amountTON, memo)   // -> *models.TransferResult

// Wallet
balance, err := api.GetWalletBalance(ctx)   // -> *models.WalletBalance
info := api.GetWalletInfo()                 // -> map[string]interface{}
```

### Parameters

| Method | Parameter | Type | Description |
|--------|-----------|------|-------------|
| `BuyStars` | `username` | `string` | Telegram username (5-32 chars) |
| | `quantity` | `int` | Number of stars (1-999999) |
| | `showSender` | `bool` | Show sender identity |
| `GiftPremium` | `months` | `int` | Duration: 3, 6, or 12 |
| `TopupTON` | `amount` | `int` | TON amount (1-999999) |
| `TransferTON` | `toAddress` | `string` | TON address or `user.t.me` |
| | `amountTON` | `float64` | Amount in TON |
| | `memo` | `string` | Transaction comment |

---

## Wallet Versions

| Version | Name | Status | Use Case |
|---------|------|--------|----------|
| **V3R1** | WalletV3R1 | Legacy | Older wallets |
| **V3R2** | WalletV3R2 | Legacy | Older wallets |
| **V4R2** | WalletV4R2 | **Recommended** | Most compatible |
| **V5R1** | WalletV5R1 | Latest | Modern features |
| **W5** | Alias for V5R1 | Latest | Alternative naming |

Version is case-insensitive: `"v4r2"`, `"V4R2"`, `"V4r2"` all work.

---

## Error Handling

```go
import fragErrors "github.com/Darkildo/fragment-api-go/errors"
```

### Error Hierarchy

```
FragmentAPIError (base)
├── AuthenticationError         — session expired / invalid credentials
├── UserNotFoundError           — user doesn't exist on Telegram
├── InvalidAmountError          — quantity/amount out of range
├── InsufficientBalanceError    — wallet balance too low
├── PaymentInitiationError      — Fragment API rejected payment
├── TransactionError            — blockchain TX failed
├── NetworkError                — HTTP request failed
├── RateLimitError              — rate limit exceeded
└── WalletError                 — generic wallet failure
    └── InvalidWalletVersionError — unsupported wallet version
```

### Example

```go
import (
    "errors"
    fragErrors "github.com/Darkildo/fragment-api-go/errors"
)

result, err := api.BuyStars(ctx, "username", 100, false)
if err != nil {
    var authErr *fragErrors.AuthenticationError
    var userErr *fragErrors.UserNotFoundError
    var balErr  *fragErrors.InsufficientBalanceError

    switch {
    case errors.As(err, &authErr):
        log.Println("Session expired — update cookies")
    case errors.As(err, &userErr):
        log.Printf("User not found: %s", userErr.Username)
    case errors.As(err, &balErr):
        log.Printf("Need %.6f TON, have %.6f", balErr.Required, balErr.Current)
    default:
        log.Printf("Error: %v", err)
    }
}
```

---

## Data Models

### UserInfo

```go
type UserInfo struct {
    Name      string // Display name
    Recipient string // Blockchain recipient address
    Found     bool   // Whether user was found
    Avatar    string // Avatar URL or base64 data
}
```

### PurchaseResult

```go
type PurchaseResult struct {
    Success         bool    // Transaction success
    TransactionHash string  // Blockchain TX hash
    Error           string  // Error message (on failure)
    User            *UserInfo
    BalanceChecked  bool    // Balance was validated
    RequiredAmount  float64 // Total TON cost
}
```

### TransferResult

```go
type TransferResult struct {
    Success         bool
    TransactionHash string
    FromAddress     string
    ToAddress       string
    AmountTON       float64
    BalanceBefore   float64
    Memo            string
    Error           string
}
```

### WalletBalance

```go
type WalletBalance struct {
    BalanceNano   string  // Balance in nanotons
    BalanceTON    float64 // Balance in TON
    Address       string  // Wallet address
    IsReady       bool    // Wallet readiness
    WalletVersion string  // Contract version
}
```

---

## Development Status

This library is a **structural skeleton** (v1.0.0). The HTTP client (`core`), models, errors, utilities, and high-level API are fully defined.

**What needs implementation:**

The `wallet` package currently returns stub errors. To make it functional, integrate a Go TON SDK:

- [xssnick/tonutils-go](https://github.com/xssnick/tonutils-go) — comprehensive TON SDK
- [tonkeeper/tongo](https://github.com/tonkeeper/tongo) — Tonkeeper's TON library

Required wallet operations:
1. **`GetBalance`** — derive address from mnemonic, query TonAPI
2. **`SendTransaction`** — decode BOC payload, sign and broadcast TX
3. **`TransferTON`** — build transfer with optional memo cell, broadcast

Each method in `wallet/wallet.go` contains detailed pseudocode and TODO comments explaining the implementation steps.

---

## License

MIT License. See [LICENSE](LICENSE) for details.

Powered By D.ildo

Based on [fragment-api-py](https://github.com/S1qwy/fragment-api-py) by [S1qwy](https://github.com/S1qwy).
