// Package models defines the data structures used by the Fragment API client.
package models

import "fmt"

// UserInfo contains information about a Telegram user retrieved from Fragment API.
type UserInfo struct {
	// Name is the user's display name on Telegram.
	Name string `json:"name"`

	// Recipient is the blockchain recipient address for the payment.
	Recipient string `json:"recipient"`

	// Found indicates whether the user was successfully found.
	Found bool `json:"found"`

	// Avatar is the URL or base64-encoded avatar image.
	Avatar string `json:"avatar,omitempty"`
}

func (u UserInfo) String() string {
	return fmt.Sprintf("UserInfo{Name: %q, Found: %v, Recipient: %q}", u.Name, u.Found, u.Recipient)
}

// TransactionMessage represents a single message in a TON transaction.
type TransactionMessage struct {
	// Address is the destination address for the message.
	Address string `json:"address"`

	// Amount is the amount in nanotons as a string.
	Amount string `json:"amount"`

	// Payload is the base64-encoded BOC payload.
	Payload string `json:"payload"`
}

func (t TransactionMessage) String() string {
	return fmt.Sprintf("TransactionMessage{Address: %q, Amount: %s}", t.Address, t.Amount)
}

// TransactionData contains the full transaction information returned by Fragment API.
type TransactionData struct {
	// Messages is the list of transaction messages.
	Messages []TransactionMessage `json:"messages"`

	// ReqID is the Fragment API request ID for the transaction.
	ReqID string `json:"req_id,omitempty"`
}

// GetFirstMessage returns the first message from the transaction data, or nil if empty.
func (t *TransactionData) GetFirstMessage() *TransactionMessage {
	if len(t.Messages) == 0 {
		return nil
	}
	return &t.Messages[0]
}

// PurchaseResult contains the result of a Stars/Premium/TON top-up purchase.
type PurchaseResult struct {
	// Success indicates whether the transaction was successful.
	Success bool `json:"success"`

	// TransactionHash is the blockchain transaction hash (empty on failure).
	TransactionHash string `json:"transaction_hash,omitempty"`

	// Error contains the error message if the transaction failed.
	Error string `json:"error,omitempty"`

	// User contains recipient information.
	User *UserInfo `json:"user,omitempty"`

	// BalanceChecked indicates whether the wallet balance was validated before sending.
	BalanceChecked bool `json:"balance_checked"`

	// RequiredAmount is the total TON amount required for the transaction (including fees).
	RequiredAmount float64 `json:"required_amount,omitempty"`
}

func (p PurchaseResult) String() string {
	if p.Success {
		return fmt.Sprintf("PurchaseResult{Success: true, TX: %q, Amount: %.6f TON}", p.TransactionHash, p.RequiredAmount)
	}
	return fmt.Sprintf("PurchaseResult{Success: false, Error: %q}", p.Error)
}

// TransferResult contains the result of a direct TON transfer.
type TransferResult struct {
	// Success indicates whether the transfer was successful.
	Success bool `json:"success"`

	// TransactionHash is the blockchain transaction hash (empty on failure).
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

	// Error contains the error message if the transfer failed.
	Error string `json:"error,omitempty"`
}

func (t TransferResult) String() string {
	if t.Success {
		return fmt.Sprintf("TransferResult{Success: true, TX: %q, Amount: %.6f TON}", t.TransactionHash, t.AmountTON)
	}
	return fmt.Sprintf("TransferResult{Success: false, Error: %q}", t.Error)
}

// WalletBalance contains the current wallet balance and metadata.
type WalletBalance struct {
	// BalanceNano is the balance in nanotons (1 TON = 1e9 nanotons).
	BalanceNano string `json:"balance_nano"`

	// BalanceTON is the balance in TON.
	BalanceTON float64 `json:"balance_ton"`

	// Address is the blockchain wallet address.
	Address string `json:"address"`

	// IsReady indicates whether the wallet is ready for transactions.
	IsReady bool `json:"is_ready"`

	// WalletVersion is the TON wallet contract version (e.g., "V4R2").
	WalletVersion string `json:"wallet_version"`
}

// HasSufficientBalance checks whether the wallet balance is enough for a transaction
// with the given required amount (in nanotons) plus an optional fee (in nanotons).
// If feeNano is 0, a default fee of 1_000_000 nanotons (0.001 TON) is used.
func (w *WalletBalance) HasSufficientBalance(requiredNano int64, feeNano int64) bool {
	if feeNano == 0 {
		feeNano = 1_000_000 // default 0.001 TON
	}
	// Parse balance_nano as int64
	var balanceNano int64
	_, _ = fmt.Sscanf(w.BalanceNano, "%d", &balanceNano)
	return balanceNano >= requiredNano+feeNano
}

func (w WalletBalance) String() string {
	return fmt.Sprintf("WalletBalance{%.6f TON, Address: %q, Version: %s, Ready: %v}",
		w.BalanceTON, w.Address, w.WalletVersion, w.IsReady)
}

// WalletVersion represents a supported TON wallet version.
type WalletVersion string

const (
	WalletV3R1 WalletVersion = "V3R1"
	WalletV3R2 WalletVersion = "V3R2"
	WalletV4R2 WalletVersion = "V4R2"
	WalletV5R1 WalletVersion = "V5R1"
	WalletW5   WalletVersion = "W5" // alias for V5R1
)
