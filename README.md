# Fragment API Go

[![Go 1.26+](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/Darkildo/fragment-api-go)

**Go client for the Fragment.com API — Telegram Stars, Premium subscriptions, and TON transfers.**

Go port of [fragment-api-py](https://github.com/S1qwy/fragment-api-py) (Python v3.2.0).  
TON blockchain integration via [tonutils-go](https://github.com/xssnick/tonutils-go).

[README on Russian](README_ru.md)

---

## Installation

```bash
go get github.com/Darkildo/fragment-api-go
```

## Quick Start

```go
import fragment "github.com/Darkildo/fragment-api-go"

api, err := fragment.New(fragment.Config{
    Cookies:        "stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=...",
    HashValue:      "your_hash_from_network_tab",
    WalletMnemonic: "word1 word2 ... word24",
})
if err != nil {
    log.Fatal(err)
}
defer api.Close()

ctx := context.Background()

// Send 100 Stars
result, err := api.BuyStars(ctx, "jane_doe", 100, false)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("TX: %s\n", result.TransactionHash)
```

## Features

- **Telegram Stars** — send Stars to any Telegram user
- **Premium Gifts** — gift 3/6/12-month Telegram Premium subscriptions
- **TON Ads Top-up** — top up TON Ads accounts
- **Direct TON Transfers** — send TON to any address with optional memo
- **Multi-Wallet Support** — V3R1, V3R2, V4R2 (default), V5R1/W5 as typed enum
- **Sender Visibility** — anonymous or visible payments
- **Automatic Retries** — context-aware exponential backoff
- **Transaction Serialization** — safe concurrent calls from multiple goroutines; transactions are queued and executed one at a time via channel semaphore, with context-aware cancellation
- **Typed Errors** — full error chain preserved via `errors.Is` / `errors.As`; dedicated `TransactionNotConfirmedError` for timeout scenarios
- **Structured Logging** — optional `log/slog` integration (stdlib, no deps)
- **Zero External Deps** — only [tonutils-go](https://github.com/xssnick/tonutils-go) + stdlib

## Project Structure

```
fragment-api-go/
  go.mod           module definition
  fragment.go      Client, New(), Config, Close, WalletBalance, WalletInfo
  types.go         UserInfo, PurchaseResult, TransferResult, WalletBalance, WalletVersion enum, WalletInfo
  errors.go        APIError hierarchy (10 typed errors with Unwrap chains)
  recipient.go     GetRecipientStars, GetRecipientPremium, GetRecipientTON
  stars.go         BuyStars
  premium.go       GiftPremium
  topup.go         TopupTON
  transfer.go      TransferTON
  purchase.go      shared purchase flow (unexported)
  core.go          HTTP transport (unexported)
  wallet.go        TON wallet via tonutils-go (unexported)
  helpers.go       validation, parsing, conversion (unexported)
  LICENSE          MIT
  example/main.go  runnable usage example
```

---

## API Reference

```go
api, err := fragment.New(fragment.Config{ ... })
defer api.Close()

// Recipient lookup
user, err := api.GetRecipientStars(ctx, "username")
user, err := api.GetRecipientPremium(ctx, "username")
user, err := api.GetRecipientTON(ctx, "username")

// Purchases — return (*PurchaseResult, error)
result, err := api.BuyStars(ctx, "username", 100, false)
result, err := api.GiftPremium(ctx, "username", 3, false)
result, err := api.TopupTON(ctx, "username", 10, false)

// Direct transfer — return (*TransferResult, error)
transfer, err := api.TransferTON(ctx, "EQ...", 0.5, "memo")

// Wallet
balance, err := api.WalletBalance(ctx)   // *WalletBalance
info := api.WalletInfo()                 // WalletInfo (typed struct)
```

---

## Wallet Versions (Typed Enum)

```go
fragment.WalletV3R1  // "V3R1" — legacy
fragment.WalletV3R2  // "V3R2" — legacy
fragment.WalletV4R2  // "V4R2" — default, recommended
fragment.WalletV5R1  // "V5R1" — latest
fragment.WalletW5    // "W5"   — alias for V5R1
```

Config accepts case-insensitive strings: `"v4r2"`, `"V4R2"`, `"w5"`.

---

## Error Handling

All errors form a chain. Use `errors.As` / `errors.Is` for typed matching.
Errors are never swallowed into string fields — always returned as Go errors.

```go
result, err := api.BuyStars(ctx, "user", 100, false)
if err != nil {
    var notConfirmed *fragment.TransactionNotConfirmedError
    var txErr  *fragment.TransactionError
    var balErr *fragment.InsufficientBalanceError
    var userErr *fragment.UserNotFoundError

    switch {
    case errors.As(err, &notConfirmed):
        // TX was sent but not confirmed — may still confirm later!
        // Check blockchain state before retrying to avoid double-spend.
        log.Printf("TX pending: %v", notConfirmed)
    case errors.As(err, &txErr):
        log.Printf("TX failed: %v", txErr)
    case errors.As(err, &balErr):
        log.Printf("Need %.6f TON, have %.6f", balErr.Required, balErr.Current)
    case errors.As(err, &userErr):
        log.Printf("User %q not found", userErr.Username)
    default:
        log.Printf("Error: %v", err)
    }
}
```

### Error Hierarchy

```
APIError (base, has Unwrap)
├── AuthenticationError
├── UserNotFoundError              — .Username
├── InvalidAmountError             — .Amount, .MinValue, .MaxValue
├── InsufficientBalanceError       — .Required, .Current
├── PaymentInitiationError
├── TransactionError
│   └── TransactionNotConfirmedError — tx sent but not confirmed in deadline
├── NetworkError                   — .StatusCode
├── RateLimitError                 — .RetryAfter
└── WalletError
    └── InvalidWalletVersionError — .Version, .SupportedVersions
```

---

## Logging

Pass a `*slog.Logger` to enable structured logging (stdlib `log/slog`).
Nil disables logging completely (no-op handler, zero overhead).

```go
logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

api, _ := fragment.New(fragment.Config{
    // ...
    Logger: logger,
})
```

---

## Concurrency

All methods on `Client` are safe to call from multiple goroutines.
Transactions (Stars, Premium, TON transfers) are automatically serialized
via an internal semaphore — only one blockchain transaction is in-flight
at a time. Callers waiting for the slot can cancel via context.

```go
api, _ := fragment.New(fragment.Config{...})
defer api.Close()

var wg sync.WaitGroup
for _, user := range []string{"alice_t", "bob_smith", "charlie_99"} {
    wg.Add(1)
    go func(username string) {
        defer wg.Done()
        ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
        defer cancel()

        // Safe: concurrent calls are queued internally.
        // Each transaction waits for the previous one to complete.
        result, err := api.BuyStars(ctx, username, 100, false)
        if err != nil {
            log.Printf("%s: %v", username, err)
            return
        }
        log.Printf("%s: TX %s", username, result.TransactionHash)
    }(user)
}
wg.Wait()
```

**Why serialization?** TON wallet contracts use a sequence number (seqno)
that increments with each outgoing transaction. Parallel sends from the same
wallet would use the same seqno, causing one transaction to be rejected by
the network. The semaphore ensures sequential execution at the wallet level.

**Timeout behavior:** if a transaction hangs (e.g. network issues), the
semaphore is held until the context deadline or a 180-second fallback from
tonutils-go. Other goroutines waiting for the slot receive
`context.DeadlineExceeded` from their own context. A hung transaction never
blocks the semaphore permanently.

---

## Config Reference

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `Cookies` | `string` | required | Fragment.com session cookies |
| `HashValue` | `string` | required | API hash from DevTools |
| `WalletMnemonic` | `string` | required | 24-word TON seed phrase |
| `WalletVersion` | `string` | `"V4R2"` | Wallet version (case-insensitive) |
| `Testnet` | `bool` | `false` | Use TON testnet |
| `Timeout` | `time.Duration` | `15s` | HTTP timeout for Fragment API |
| `Logger` | `*slog.Logger` | `nil` (disabled) | Structured logger |

---

## License

MIT. See [LICENSE](LICENSE).

Based on [fragment-api-py](https://github.com/S1qwy/fragment-api-py) by [S1qwy](https://github.com/S1qwy).
