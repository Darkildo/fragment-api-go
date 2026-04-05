// Example demonstrates basic usage of the Fragment API Go client.
//
// Before running, set environment variables:
//
//	export FRAGMENT_COOKIES="stel_ssid=...; stel_token=...; stel_dt=...; stel_ton_token=..."
//	export FRAGMENT_HASH="your_hash_value"
//	export WALLET_MNEMONIC="word1 word2 ... word24"
//	export WALLET_API_KEY="your_tonapi_key"
//	export WALLET_VERSION="V4R2"
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	fragment "github.com/Darkildo/fragment-api-go"
)

func main() {
	api, err := fragment.New(fragment.Config{
		Cookies:        os.Getenv("FRAGMENT_COOKIES"),
		HashValue:      os.Getenv("FRAGMENT_HASH"),
		WalletMnemonic: os.Getenv("WALLET_MNEMONIC"),
		WalletAPIKey:   os.Getenv("WALLET_API_KEY"),
		WalletVersion:  envOrDefault("WALLET_VERSION", "V4R2"),
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer api.Close()

	ctx := context.Background()

	// --- Look up a user ---
	fmt.Println("=== Looking up user ===")
	user, err := api.GetRecipientStars(ctx, "jane_doe")
	if err != nil {
		log.Printf("User lookup failed: %v", err)
	} else {
		fmt.Printf("Name:      %s\n", user.Name)
		fmt.Printf("Recipient: %s\n", user.Recipient)
		fmt.Printf("Found:     %v\n", user.Found)
	}

	// --- Send Stars (anonymous) ---
	fmt.Println("\n=== Sending 100 Stars (anonymous) ===")
	result, err := api.BuyStars(ctx, "jane_doe", 100, false)
	if err != nil {
		log.Printf("BuyStars error: %v", err)
	} else if result.Success {
		fmt.Printf("Transaction: %s\n", result.TransactionHash)
		fmt.Printf("Cost:        %.6f TON\n", result.RequiredAmount)
	} else {
		fmt.Printf("Failed: %s\n", result.Error)
	}

	// --- Send Stars (visible sender) ---
	fmt.Println("\n=== Sending 50 Stars (visible sender) ===")
	result, err = api.BuyStars(ctx, "jane_doe", 50, true)
	if err != nil {
		log.Printf("BuyStars error: %v", err)
	} else if result.Success {
		fmt.Printf("Transaction: %s\n", result.TransactionHash)
	} else {
		fmt.Printf("Failed: %s\n", result.Error)
	}

	// --- Gift Premium ---
	fmt.Println("\n=== Gifting 3 months Premium ===")
	premResult, err := api.GiftPremium(ctx, "jane_doe", 3, false)
	if err != nil {
		log.Printf("GiftPremium error: %v", err)
	} else if premResult.Success {
		fmt.Printf("Transaction: %s\n", premResult.TransactionHash)
	} else {
		fmt.Printf("Failed: %s\n", premResult.Error)
	}

	// --- Direct TON transfer ---
	fmt.Println("\n=== Transferring 0.5 TON ===")
	transfer, err := api.TransferTON(ctx, "recipient.t.me", 0.5, "Payment for services")
	if err != nil {
		log.Printf("TransferTON error: %v", err)
	} else if transfer.Success {
		fmt.Printf("From:   %s\n", transfer.FromAddress)
		fmt.Printf("To:     %s\n", transfer.ToAddress)
		fmt.Printf("Amount: %.6f TON\n", transfer.AmountTON)
		fmt.Printf("Memo:   %s\n", transfer.Memo)
		fmt.Printf("TX:     %s\n", transfer.TransactionHash)
	} else {
		fmt.Printf("Failed: %s\n", transfer.Error)
	}

	// --- Check wallet balance ---
	fmt.Println("\n=== Wallet Balance ===")
	balance, err := api.WalletBalance(ctx)
	if err != nil {
		log.Printf("WalletBalance error: %v", err)
	} else {
		fmt.Printf("Balance: %.6f TON\n", balance.BalanceTON)
		fmt.Printf("Address: %s\n", balance.Address)
		fmt.Printf("Version: %s\n", balance.WalletVersion)
		fmt.Printf("Ready:   %v\n", balance.IsReady)
	}

	// --- Wallet info ---
	fmt.Println("\n=== Wallet Info ===")
	info := api.WalletInfo()
	fmt.Printf("Version:   %s\n", info["version"])
	fmt.Printf("Supported: %v\n", info["supported_versions"])
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
