package fragment

import (
	"strings"
	"testing"

	"github.com/Darkildo/fragment-api-go/internal/types"
)

// --- WalletVersion ---

func TestWalletVersion_String(t *testing.T) {
	tests := []struct {
		v    WalletVersion
		want string
	}{
		{WalletV3R1, "V3R1"},
		{WalletV3R2, "V3R2"},
		{WalletV4R2, "V4R2"},
		{WalletV5R1, "V5R1"},
		{WalletW5, "W5"},
	}
	for _, tt := range tests {
		if got := tt.v.String(); got != tt.want {
			t.Errorf("%v.String() = %q, want %q", tt.v, got, tt.want)
		}
	}
}

// --- UserInfo ---

func TestUserInfo_String(t *testing.T) {
	u := UserInfo{Name: "Alice", Found: true}
	s := u.String()
	if !strings.Contains(s, "Alice") || !strings.Contains(s, "true") {
		t.Errorf("String() = %q, want to contain name and found", s)
	}
}

// --- PurchaseResult ---

func TestPurchaseResult_String_Success(t *testing.T) {
	p := PurchaseResult{
		Success:         true,
		TransactionHash: "abc123",
		RequiredAmount:  1.5,
	}
	s := p.String()
	if !strings.Contains(s, "abc123") || !strings.Contains(s, "1.5") {
		t.Errorf("String() = %q, want TX and cost", s)
	}
}

func TestPurchaseResult_String_Pending_WithUser(t *testing.T) {
	p := PurchaseResult{
		User: &UserInfo{Name: "Bob"},
	}
	s := p.String()
	if !strings.Contains(s, "Pending") || !strings.Contains(s, "Bob") {
		t.Errorf("String() = %q, want Pending and user name", s)
	}
}

func TestPurchaseResult_String_Pending_NoUser(t *testing.T) {
	p := PurchaseResult{}
	s := p.String()
	if s != "PurchaseResult{Pending}" {
		t.Errorf("String() = %q, want PurchaseResult{Pending}", s)
	}
}

// --- TransferResult ---

func TestTransferResult_String(t *testing.T) {
	tr := TransferResult{
		Success:         true,
		TransactionHash: "txhash",
		FromAddress:     "EQfrom",
		ToAddress:       "EQto",
		AmountTON:       0.5,
	}
	s := tr.String()
	if !strings.Contains(s, "txhash") || !strings.Contains(s, "0.5") {
		t.Errorf("String() = %q, want TX and amount", s)
	}
	if !strings.Contains(s, "EQfrom") || !strings.Contains(s, "EQto") {
		t.Errorf("String() = %q, want addresses", s)
	}
}

// --- WalletBalance ---

func TestWalletBalance_HasSufficientBalance(t *testing.T) {
	bal := &WalletBalance{BalanceNano: 2_000_000_000} // 2 TON

	// 1 TON + default fee (0.001 TON) = 1.001 TON < 2 TON
	if !bal.HasSufficientBalance(1_000_000_000, 0) {
		t.Error("should have sufficient balance for 1 TON + default fee")
	}

	// 1 TON + 1 TON fee = 2 TON == 2 TON
	if !bal.HasSufficientBalance(1_000_000_000, 1_000_000_000) {
		t.Error("should have sufficient balance for exactly 2 TON total")
	}

	// 1 TON + 1.1 TON fee = 2.1 TON > 2 TON
	if bal.HasSufficientBalance(1_000_000_000, 1_100_000_000) {
		t.Error("should NOT have sufficient balance for 2.1 TON total")
	}
}

func TestWalletBalance_HasSufficientBalance_DefaultFee(t *testing.T) {
	bal := &WalletBalance{BalanceNano: 1_001_000} // just above 1M + 1M default

	// requiredNano=0, feeNano=0 -> uses default 1_000_000
	if !bal.HasSufficientBalance(0, 0) {
		t.Error("should have sufficient balance for 0 + default fee")
	}

	// requiredNano=1_000, feeNano=0 -> 1_000 + 1_000_000 = 1_001_000 <= 1_001_000
	if !bal.HasSufficientBalance(1_000, 0) {
		t.Error("should have sufficient balance for 1000 + default fee")
	}

	// requiredNano=2_000, feeNano=0 -> 2_000 + 1_000_000 = 1_002_000 > 1_001_000
	if bal.HasSufficientBalance(2_000, 0) {
		t.Error("should NOT have sufficient for 2000 + default fee")
	}
}

func TestWalletBalance_String(t *testing.T) {
	bal := WalletBalance{
		BalanceTON: 1.5,
		Address:    "EQxxx",
		Version:    WalletV4R2,
	}
	s := bal.String()
	if !strings.Contains(s, "1.5") || !strings.Contains(s, "EQxxx") || !strings.Contains(s, "V4R2") {
		t.Errorf("String() = %q", s)
	}
}

// --- WalletInfo ---

func TestWalletInfo_String(t *testing.T) {
	wi := WalletInfo{
		Version: WalletV4R2,
		Address: "EQaddr",
	}
	s := wi.String()
	if !strings.Contains(s, "V4R2") || !strings.Contains(s, "EQaddr") {
		t.Errorf("String() = %q", s)
	}
}

// --- TransactionMessage ---

func TestTransactionMessage_AmountNano(t *testing.T) {
	msg := &types.TransactionMessage{Amount: "1500000000"}
	got, err := msg.AmountNano()
	if err != nil {
		t.Fatal(err)
	}
	if got != 1_500_000_000 {
		t.Errorf("AmountNano() = %d, want 1500000000", got)
	}
}

func TestTransactionMessage_AmountNano_Invalid(t *testing.T) {
	msg := &types.TransactionMessage{Amount: "not_a_number"}
	_, err := msg.AmountNano()
	if err == nil {
		t.Fatal("expected error for invalid amount")
	}
}
