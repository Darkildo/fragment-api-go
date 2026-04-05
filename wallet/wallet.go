// Package wallet provides TON wallet management for the Fragment API client.
//
// WalletManager handles wallet initialization, balance queries, transaction signing,
// and direct TON transfers. It supports multiple TON wallet contract versions:
// V3R1, V3R2, V4R2 (recommended), V5R1/W5.
//
// This package is a skeleton — actual TON blockchain integration requires
// a Go TON SDK (e.g., tonutils-go, tongo, or xssnick/tonutils-go).
package wallet

import (
	"context"
	"fmt"
	"strings"

	fragErrors "github.com/Darkildo/fragment-api-go/errors"
	"github.com/Darkildo/fragment-api-go/models"
)

const (
	// TransferFeeNano is the default transaction fee in nanotons (0.001 TON).
	TransferFeeNano int64 = 1_000_000

	// TransferFeeTON is the default fee for direct TON transfers.
	TransferFeeTON float64 = 0.05

	// DefaultVersion is the default wallet contract version.
	DefaultVersion = "V4R2"
)

// versionAliases maps wallet version aliases to their canonical names.
var versionAliases = map[string]string{
	"V3R1": "V3R1",
	"V3R2": "V3R2",
	"V4R2": "V4R2",
	"V5R1": "V5R1",
	"W5":   "V5R1",
}

// Config contains the configuration for creating a WalletManager.
type Config struct {
	// Mnemonic is the 24-word seed phrase separated by spaces.
	Mnemonic string

	// APIKey is the TonAPI key for blockchain queries.
	APIKey string

	// Version is the TON wallet contract version (e.g., "V4R2", "V5R1", "W5").
	// Defaults to "V4R2" if empty.
	Version string
}

// Manager handles TON wallet operations: balance queries, transaction signing,
// and direct transfers.
type Manager struct {
	mnemonic []string
	apiKey   string
	version  string
}

// NewManager creates a new WalletManager with the given configuration.
// Returns an error if the mnemonic, API key, or wallet version is invalid.
func NewManager(cfg Config) (*Manager, error) {
	if cfg.Mnemonic == "" {
		return nil, fragErrors.NewWalletError("wallet mnemonic is required", nil)
	}
	if cfg.APIKey == "" {
		return nil, fragErrors.NewWalletError("wallet API key is required", nil)
	}

	version := cfg.Version
	if version == "" {
		version = DefaultVersion
	}

	normalizedVersion, err := normalizeVersion(version)
	if err != nil {
		return nil, err
	}

	words := strings.Fields(cfg.Mnemonic)
	if len(words) != 24 {
		return nil, fragErrors.NewWalletError(
			fmt.Sprintf("mnemonic must be 24 words, got %d", len(words)), nil,
		)
	}

	return &Manager{
		mnemonic: words,
		apiKey:   cfg.APIKey,
		version:  normalizedVersion,
	}, nil
}

// Version returns the normalized wallet version string.
func (m *Manager) Version() string {
	return m.version
}

// GetBalance retrieves the current wallet balance from the TON blockchain.
//
// TODO: Implement using a Go TON SDK (e.g., tonutils-go).
// This should:
//  1. Derive the wallet address from the mnemonic and version.
//  2. Query the TonAPI (https://tonapi.io/v2/accounts/{address}) for the balance.
//  3. Return a WalletBalance struct with the balance, address, and readiness status.
func (m *Manager) GetBalance(ctx context.Context) (*models.WalletBalance, error) {
	// TODO: Implement TON blockchain integration.
	//
	// Pseudocode:
	//   client := tonapi.NewClient(m.apiKey)
	//   walletClass := getWalletClass(m.version)
	//   wallet, _ := walletClass.FromMnemonic(client, m.mnemonic)
	//   address := wallet.Address()
	//   account, _ := client.GetAccount(ctx, address)
	//   balance := account.Balance
	//   return &models.WalletBalance{
	//       BalanceNano:   fmt.Sprintf("%d", balance),
	//       BalanceTON:    float64(balance) / 1e9,
	//       Address:       address.String(),
	//       IsReady:       account.Status == "active",
	//       WalletVersion: m.version,
	//   }, nil

	return nil, fragErrors.NewWalletError("GetBalance not yet implemented — requires TON SDK integration", nil)
}

// SendTransaction sends a signed transaction to the TON blockchain.
//
// Parameters:
//   - destination: TON address of the recipient.
//   - amountNano: amount to send in nanotons.
//   - bocPayload: base64-encoded BOC payload from Fragment API.
//
// TODO: Implement using a Go TON SDK.
// This should:
//  1. Check wallet balance is sufficient (amountNano + fee).
//  2. Decode the BOC payload into a Cell.
//  3. Create and sign a transfer message with the wallet.
//  4. Broadcast the transaction to the TON network.
//  5. Return the transaction hash.
func (m *Manager) SendTransaction(ctx context.Context, destination string, amountNano string, bocPayload string) (string, error) {
	// TODO: Implement TON blockchain integration.
	//
	// Pseudocode:
	//   client := tonapi.NewClient(m.apiKey)
	//   wallet, _ := walletClass.FromMnemonic(client, m.mnemonic)
	//   balance, _ := m.GetBalance(ctx)
	//   if !balance.HasSufficientBalance(amountNanoInt, TransferFeeNano) {
	//       return "", errors.NewInsufficientBalanceError(...)
	//   }
	//   cell, _ := boc.DeserializeBoc(base64Decode(bocPayload))
	//   txHash, _ := wallet.Transfer(ctx, destination, amountNano, cell)
	//   return txHash, nil

	return "", fragErrors.NewWalletError("SendTransaction not yet implemented — requires TON SDK integration", nil)
}

// TransferTON sends TON directly to any wallet address or Telegram username.
//
// Parameters:
//   - toAddress: destination address (TON address or "username.t.me" format).
//   - amountTON: amount to transfer in TON.
//   - memo: optional text comment for the transaction (can be empty).
//
// TODO: Implement using a Go TON SDK.
// This should:
//  1. Validate the address and amount.
//  2. Check wallet balance is sufficient (amountTON + TransferFeeTON).
//  3. If memo is provided, build a Cell with: store_uint(0, 32) + store_snake_string(memo).
//  4. Create and sign a transfer message.
//  5. Broadcast and return TransferResult.
func (m *Manager) TransferTON(ctx context.Context, toAddress string, amountTON float64, memo string) (*models.TransferResult, error) {
	if toAddress == "" {
		return nil, fragErrors.NewWalletError("destination address is required", nil)
	}
	if amountTON <= 0 {
		return nil, fragErrors.NewWalletError("amount must be greater than 0", nil)
	}

	// TODO: Implement TON blockchain integration.
	//
	// Pseudocode:
	//   balance, _ := m.GetBalance(ctx)
	//   if balance.BalanceTON < amountTON + TransferFeeTON {
	//       return &models.TransferResult{Success: false, Error: "insufficient balance"}, nil
	//   }
	//   var body *cell.Cell
	//   if memo != "" {
	//       body = cell.BeginCell().StoreUInt(0, 32).StoreSnakeString(memo).EndCell()
	//   }
	//   txHash, _ := wallet.Transfer(ctx, toAddress, tonToNano(amountTON), body)
	//   return &models.TransferResult{
	//       Success:         true,
	//       TransactionHash: txHash,
	//       FromAddress:     wallet.Address().String(),
	//       ToAddress:       toAddress,
	//       AmountTON:       amountTON,
	//       BalanceBefore:   balance.BalanceTON,
	//       Memo:            memo,
	//   }, nil

	return nil, fragErrors.NewWalletError("TransferTON not yet implemented — requires TON SDK integration", nil)
}

// GetWalletInfo returns metadata about the wallet configuration.
func (m *Manager) GetWalletInfo() map[string]interface{} {
	return map[string]interface{}{
		"version":            m.version,
		"supported_versions": getSupportedVersions(),
		"version_mapping":    versionAliases,
	}
}

// normalizeVersion validates and normalizes a wallet version string.
// It is case-insensitive and resolves aliases (e.g., "w5" -> "V5R1").
func normalizeVersion(version string) (string, error) {
	upper := strings.ToUpper(version)
	canonical, ok := versionAliases[upper]
	if !ok {
		return "", fragErrors.NewInvalidWalletVersionError(version)
	}
	return canonical, nil
}

// getSupportedVersions returns a list of supported wallet version strings.
func getSupportedVersions() []string {
	versions := make([]string, 0, len(versionAliases))
	for v := range versionAliases {
		versions = append(versions, v)
	}
	return versions
}
