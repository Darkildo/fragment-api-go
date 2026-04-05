package fragment

import (
	"fmt"
	"strconv"
)

// WalletVersion represents a supported TON wallet contract version.
type WalletVersion string

const (
	WalletV3R1 WalletVersion = "V3R1"
	WalletV3R2 WalletVersion = "V3R2"
	WalletV4R2 WalletVersion = "V4R2" // recommended default
	WalletV5R1 WalletVersion = "V5R1"
	WalletW5   WalletVersion = "W5" // alias for V5R1
)

// String returns the version string.
func (v WalletVersion) String() string { return string(v) }

// UserInfo contains information about a Telegram user retrieved from
// the Fragment API.
type UserInfo struct {
	// Name is the user's display name.
	Name string `json:"name"`

	// Recipient is the blockchain recipient address for the payment.
	Recipient string `json:"recipient"`

	// Found indicates whether the user was successfully found.
	Found bool `json:"found"`

	// Avatar is the URL or base64-encoded avatar image.
	Avatar string `json:"avatar,omitempty"`
}

func (u UserInfo) String() string {
	return fmt.Sprintf("UserInfo{Name: %q, Found: %v}", u.Name, u.Found)
}

// PurchaseResult contains the result of a Stars / Premium / TON top-up
// purchase operation.
//
// When a purchase fails during the multi-step flow (user lookup, payment
// initiation, transaction broadcast), the Go error is returned alongside
// the result so that callers can use [errors.Is] and [errors.As].
type PurchaseResult struct {
	// Success is true when the blockchain transaction succeeded.
	Success bool `json:"success"`

	// TransactionHash is the blockchain transaction hash (empty on failure).
	TransactionHash string `json:"transaction_hash,omitempty"`

	// User is the resolved recipient information (may be nil on early failures).
	User *UserInfo `json:"user,omitempty"`

	// BalanceChecked is true when the wallet balance was validated
	// before sending.
	BalanceChecked bool `json:"balance_checked"`

	// RequiredAmount is the total TON required (including fees).
	RequiredAmount float64 `json:"required_amount,omitempty"`
}

func (p PurchaseResult) String() string {
	if p.Success {
		return fmt.Sprintf("PurchaseResult{OK, TX: %s, Cost: %.6f TON}", p.TransactionHash, p.RequiredAmount)
	}
	return "PurchaseResult{FAIL}"
}

// TransferResult contains the result of a direct TON transfer.
type TransferResult struct {
	// Success is true when the transfer succeeded.
	Success bool `json:"success"`

	// TransactionHash is the blockchain transaction hash.
	TransactionHash string `json:"transaction_hash,omitempty"`

	// FromAddress is the sender's wallet address.
	FromAddress string `json:"from_address,omitempty"`

	// ToAddress is the recipient's address.
	ToAddress string `json:"to_address,omitempty"`

	// AmountTON is the amount transferred in TON.
	AmountTON float64 `json:"amount_ton,omitempty"`

	// BalanceBefore is the wallet balance before the transfer.
	BalanceBefore float64 `json:"balance_before,omitempty"`

	// Memo is the text comment included in the transaction.
	Memo string `json:"memo,omitempty"`
}

func (t TransferResult) String() string {
	if t.Success {
		return fmt.Sprintf("TransferResult{OK, TX: %s, %.6f TON}", t.TransactionHash, t.AmountTON)
	}
	return "TransferResult{FAIL}"
}

// WalletBalance contains the current wallet balance and metadata.
type WalletBalance struct {
	// BalanceNano is the balance in nanotons (1 TON = 1e9 nanotons).
	BalanceNano uint64 `json:"balance_nano"`

	// BalanceTON is the balance in TON.
	BalanceTON float64 `json:"balance_ton"`

	// Address is the blockchain wallet address.
	Address string `json:"address"`

	// IsReady indicates whether the wallet is ready for transactions.
	IsReady bool `json:"is_ready"`

	// Version is the TON wallet contract version.
	Version WalletVersion `json:"wallet_version"`
}

// HasSufficientBalance reports whether the wallet can cover requiredNano
// plus feeNano nanotons. If feeNano is 0 a default of 1 000 000 (0.001 TON)
// is used.
func (w *WalletBalance) HasSufficientBalance(requiredNano, feeNano uint64) bool {
	if feeNano == 0 {
		feeNano = 1_000_000
	}
	return w.BalanceNano >= requiredNano+feeNano
}

func (w WalletBalance) String() string {
	return fmt.Sprintf("WalletBalance{%.6f TON, %s, %s}", w.BalanceTON, w.Address, w.Version)
}

// WalletInfo contains metadata about the wallet configuration.
type WalletInfo struct {
	// Version is the active wallet contract version.
	Version WalletVersion `json:"version"`

	// SupportedVersions lists all recognised version strings.
	SupportedVersions []WalletVersion `json:"supported_versions"`

	// Address is the wallet's blockchain address.
	// Empty until the first connection to the TON network.
	Address string `json:"address,omitempty"`
}

func (i WalletInfo) String() string {
	return fmt.Sprintf("WalletInfo{Version: %s, Address: %s}", i.Version, i.Address)
}

// transactionMessage is an internal representation of a single TON
// transaction message returned by the Fragment API.
type transactionMessage struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
	Payload string `json:"payload"`
}

// amountNano returns the amount as uint64.
func (t *transactionMessage) amountNano() (uint64, error) {
	return strconv.ParseUint(t.Amount, 10, 64)
}
