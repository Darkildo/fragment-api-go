// Package fragment provides a Go client library for the Fragment.com API.
//
// Fragment.com is a platform for purchasing Telegram Stars, Premium subscriptions,
// and managing TON-based payments. This library is a Go port of the Python
// fragment-api-py library (https://github.com/S1qwy/fragment-api-py).
//
// # Architecture
//
// The library is organized into the following packages:
//
//   - client:  High-level FragmentAPI client with all public methods.
//   - core:    Low-level HTTP client for Fragment.com API communication.
//   - wallet:  TON wallet management (balance, transactions, transfers).
//   - models:  Data structures (UserInfo, PurchaseResult, TransferResult, etc.).
//   - errors:  Error types hierarchy matching the Python library's exceptions.
//   - utils:   Utility functions (cookie parsing, validation, TON conversion).
//
// # Quick Start
//
//	import (
//	    "context"
//	    "fmt"
//	    "log"
//
//	    "github.com/Darkildo/fragment-api-go/client"
//	)
//
//	func main() {
//	    api, err := client.New(client.Config{
//	        Cookies:        "stel_ssid=...; stel_token=...",
//	        HashValue:      "your_hash",
//	        WalletMnemonic: "word1 word2 ... word24",
//	        WalletAPIKey:   "your-tonapi-key",
//	        WalletVersion:  "V4R2",
//	    })
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    defer api.Close()
//
//	    ctx := context.Background()
//
//	    // Look up a user
//	    user, err := api.GetRecipientStars(ctx, "jane_doe")
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    fmt.Printf("User: %s\n", user.Name)
//
//	    // Send Stars
//	    result, err := api.BuyStars(ctx, "jane_doe", 100, false)
//	    if err != nil {
//	        log.Fatal(err)
//	    }
//	    if result.Success {
//	        fmt.Printf("TX: %s\n", result.TransactionHash)
//	    }
//	}
//
// # Version
//
// Current version: 1.0.0 (Go port of fragment-api-py v3.2.0)
package fragment

const Version = "1.0.0"
