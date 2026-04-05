package fragment

import (
	"context"
	"fmt"
	"strings"
)

const (
	transferFeeNano int64   = 1_000_000 // 0.001 TON
	transferFeeTON  float64 = 0.05
	defaultVersion          = "V4R2"
)

// versionAliases maps every accepted version string (upper-case) to its
// canonical form.
var versionAliases = map[string]string{
	"V3R1": "V3R1",
	"V3R2": "V3R2",
	"V4R2": "V4R2",
	"V5R1": "V5R1",
	"W5":   "V5R1",
}

// walletManager handles TON wallet operations.
//
// This is a skeleton: actual TON blockchain integration requires a Go TON
// SDK such as github.com/xssnick/tonutils-go or github.com/tonkeeper/tongo.
// Every method that touches the blockchain contains a TODO with pseudocode.
type walletManager struct {
	mnemonic []string
	apiKey   string
	version  string
}

// newWalletManager creates and validates a walletManager.
func newWalletManager(mnemonic, apiKey, version string) (*walletManager, error) {
	if mnemonic == "" {
		return nil, newWalletError("wallet mnemonic is required", nil)
	}
	if apiKey == "" {
		return nil, newWalletError("wallet API key is required", nil)
	}

	if version == "" {
		version = defaultVersion
	}
	ver, err := normalizeVersion(version)
	if err != nil {
		return nil, err
	}

	words := strings.Fields(mnemonic)
	if len(words) != 24 {
		return nil, newWalletError(fmt.Sprintf("mnemonic must be 24 words, got %d", len(words)), nil)
	}

	return &walletManager{mnemonic: words, apiKey: apiKey, version: ver}, nil
}

// getBalance retrieves the current wallet balance from the TON blockchain.
//
// TODO: Implement with a Go TON SDK.
//  1. Derive wallet address from mnemonic + version.
//  2. Query TonAPI (https://tonapi.io/v2/accounts/{address}).
//  3. Return *WalletBalance.
func (w *walletManager) getBalance(ctx context.Context) (*WalletBalance, error) {
	return nil, newWalletError("getBalance not yet implemented — requires TON SDK integration", nil)
}

// sendTransaction signs and broadcasts a transaction to the TON network.
//
// Parameters:
//   - destination: TON address of the recipient.
//   - amountNano:  amount in nanotons.
//   - bocPayload:  base64-encoded BOC payload from Fragment API.
//
// TODO: Implement with a Go TON SDK.
//  1. Check wallet balance >= amountNano + fee.
//  2. Decode BOC payload into a Cell.
//  3. Build, sign, and broadcast transfer message.
//  4. Return transaction hash.
func (w *walletManager) sendTransaction(ctx context.Context, destination, amountNano, bocPayload string) (string, error) {
	return "", newWalletError("sendTransaction not yet implemented — requires TON SDK integration", nil)
}

// transferTON sends TON directly to any address or Telegram username.
//
// Parameters:
//   - toAddress: TON address or "username.t.me" format.
//   - amountTON: amount in TON.
//   - memo:      optional text comment (empty string for none).
//
// TODO: Implement with a Go TON SDK.
//  1. Validate address and amount.
//  2. Check balance >= amountTON + transferFeeTON.
//  3. If memo != "", build Cell: store_uint(0, 32) + store_snake_string(memo).
//  4. Sign and broadcast.
//  5. Return *TransferResult.
func (w *walletManager) transferTON(ctx context.Context, toAddress string, amountTON float64, memo string) (*TransferResult, error) {
	if toAddress == "" {
		return nil, newWalletError("destination address is required", nil)
	}
	if amountTON <= 0 {
		return nil, newWalletError("amount must be greater than 0", nil)
	}
	return nil, newWalletError("transferTON not yet implemented — requires TON SDK integration", nil)
}

// info returns wallet metadata.
func (w *walletManager) info() map[string]interface{} {
	names := make([]string, 0, len(versionAliases))
	for v := range versionAliases {
		names = append(names, v)
	}
	return map[string]interface{}{
		"version":            w.version,
		"supported_versions": names,
		"version_mapping":    versionAliases,
	}
}

// normalizeVersion validates and normalises a version string (case-insensitive,
// resolves aliases like "w5" → "V5R1").
func normalizeVersion(v string) (string, error) {
	canonical, ok := versionAliases[strings.ToUpper(v)]
	if !ok {
		return "", newInvalidWalletVersionError(v)
	}
	return canonical, nil
}
