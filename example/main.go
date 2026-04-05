// Example demonstrates basic usage of the Fragment API Go client.
//
// Before running, set environment variables:
//
//	export FRAGMENT_COOKIES="stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=..."
//	export FRAGMENT_HASH="your_hash_value"
//	export WALLET_MNEMONIC="word1 word2 ... word24"
//	export WALLET_VERSION="V4R2"
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"

	fragment "github.com/Darkildo/fragment-api-go"
)

func main() {
	// Use slog for structured logging (optional — pass nil to disable).
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))

	api, err := fragment.New(fragment.Config{
		Cookies:        os.Getenv("FRAGMENT_COOKIES"),
		HashValue:      os.Getenv("FRAGMENT_HASH"),
		WalletMnemonic: os.Getenv("WALLET_MNEMONIC"),
		WalletVersion:  envOrDefault("WALLET_VERSION", "V4R2"),
		Logger:         logger,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer api.Close()

	ctx := context.Background()

	// --- Look up a user ---
	fmt.Println("=== Looking up user ===")
	user, err := api.GetRecipientStars(ctx, "VladimIRKish")
	if err != nil {
		var uerr *fragment.UserNotFoundError
		if errors.As(err, &uerr) {
			log.Printf("User %q not found", uerr.Username)
		} else {
			log.Printf("Lookup error: %v", err)
		}
	} else {
		fmt.Printf("Name:      %s\n", user.Name)
		fmt.Printf("Recipient: %s\n", user.Recipient)
		fmt.Printf("Found:     %v\n", user.Found)
	}

	// --- Send Stars (anonymous) ---
	fmt.Println("\n=== Sending 100 Stars (anonymous) ===")
	result, err := api.BuyStars(ctx, "VladimIRKish", 100, false)
	if err != nil {
		var txErr *fragment.TransactionError
		if errors.As(err, &txErr) {
			log.Printf("Transaction failed: %v (cause: %v)", txErr, errors.Unwrap(txErr))
		} else {
			log.Printf("BuyStars error: %v", err)
		}
	} else if result.Success {
		fmt.Printf("Transaction: %s\n", result.TransactionHash)
		fmt.Printf("Cost:        %.6f TON\n", result.RequiredAmount)
	}

	// --- Gift Premium ---
	fmt.Println("\n=== Gifting 3 months Premium ===")
	premResult, err := api.GiftPremium(ctx, "VladimIRKish", 3, false)
	if err != nil {
		log.Printf("GiftPremium error: %v", err)
	} else if premResult.Success {
		fmt.Printf("Transaction: %s\n", premResult.TransactionHash)
	}

	// --- Direct TON transfer ---
	fmt.Println("\n=== Transferring 0.5 TON ===")
	transfer, err := api.TransferTON(ctx, "UQAq1cXMjGoz5fB9xoZlf0H6hHGtbS6tEIcQ3U7l_Oyk9fT2", 0.5, "Test Payment")
	if err != nil {
		var balErr *fragment.InsufficientBalanceError
		if errors.As(err, &balErr) {
			log.Printf("Need %.6f TON, have %.6f TON", balErr.Required, balErr.Current)
		} else {
			log.Printf("TransferTON error: %v", err)
		}
	} else if transfer.Success {
		fmt.Printf("TX:     %s\n", transfer.TransactionHash)
		fmt.Printf("Amount: %.6f TON\n", transfer.AmountTON)
	}

	// --- Check wallet balance ---
	fmt.Println("\n=== Wallet Balance ===")
	balance, err := api.WalletBalance(ctx)
	if err != nil {
		log.Printf("WalletBalance error: %v", err)
	} else {
		fmt.Printf("Balance: %.6f TON (%d nano)\n", balance.BalanceTON, balance.BalanceNano)
		fmt.Printf("Address: %s\n", balance.Address)
		fmt.Printf("Version: %s\n", balance.Version)
		fmt.Printf("Ready:   %v\n", balance.IsReady)
	}

	// --- Wallet info ---
	fmt.Println("\n=== Wallet Info ===")
	info := api.WalletInfo()
	fmt.Printf("Version:   %s\n", info.Version)
	fmt.Printf("Supported: %v\n", info.SupportedVersions)
	fmt.Printf("Address:   %s\n", info.Address)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
