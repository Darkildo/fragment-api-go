package fragment

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Darkildo/fragment-api-go/internal/tonwallet"
	"github.com/Darkildo/fragment-api-go/internal/types"
	"github.com/xssnick/tonutils-go/ton"
)

func TestAPIError_Error_WithCause(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	e := &APIError{Message: "network failure", Cause: cause}

	got := e.Error()
	if !strings.Contains(got, "network failure") || !strings.Contains(got, "connection refused") {
		t.Errorf("Error() = %q, want both message and cause", got)
	}
}

func TestAPIError_Error_NoCause(t *testing.T) {
	e := &APIError{Message: "something went wrong"}
	if got := e.Error(); got != "something went wrong" {
		t.Errorf("Error() = %q, want %q", got, "something went wrong")
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("root cause")
	e := &APIError{Message: "wrapper", Cause: cause}
	if got := e.Unwrap(); got != cause {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}

	e2 := &APIError{Message: "no cause"}
	if got := e2.Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}

// --- errors.As matching ---

func TestErrorsAs_AuthenticationError(t *testing.T) {
	err := types.NewAuthenticationError("session expired", nil)
	var target *AuthenticationError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match *AuthenticationError")
	}
	if target.Message != "session expired" {
		t.Errorf("Message = %q", target.Message)
	}
}

func TestErrorsAs_UserNotFoundError(t *testing.T) {
	cause := fmt.Errorf("invalid username")
	err := types.NewUserNotFoundError("test_user", cause)
	var target *UserNotFoundError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match *UserNotFoundError")
	}
	if target.Username != "test_user" {
		t.Errorf("Username = %q, want %q", target.Username, "test_user")
	}
	if target.Unwrap() != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestErrorsAs_InvalidAmountError(t *testing.T) {
	cause := fmt.Errorf("out of range")
	err := types.NewInvalidAmountError(0, 1, 100, cause)
	var target *InvalidAmountError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
	if target.Amount != 0 || target.MinValue != 1 || target.MaxValue != 100 {
		t.Errorf("fields: Amount=%d Min=%d Max=%d", target.Amount, target.MinValue, target.MaxValue)
	}
	if target.Unwrap() != cause {
		t.Error("cause should be preserved")
	}
}

func TestErrorsAs_InsufficientBalanceError(t *testing.T) {
	err := types.NewInsufficientBalanceError(1.5, 0.3)
	var target *InsufficientBalanceError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
	if target.Required != 1.5 || target.Current != 0.3 {
		t.Errorf("Required=%f Current=%f", target.Required, target.Current)
	}
}

func TestErrorsAs_NetworkError(t *testing.T) {
	cause := fmt.Errorf("timeout")
	err := types.NewNetworkError("request failed", 503, cause)
	var target *NetworkError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
	if target.StatusCode != 503 {
		t.Errorf("StatusCode = %d, want 503", target.StatusCode)
	}
	if target.Unwrap() != cause {
		t.Error("cause should be preserved")
	}
}

func TestErrorsAs_RateLimitError(t *testing.T) {
	err := types.NewRateLimitError(60)
	var target *RateLimitError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
	if target.RetryAfter != 60 {
		t.Errorf("RetryAfter = %d, want 60", target.RetryAfter)
	}
}

func TestErrorsAs_TransactionError(t *testing.T) {
	cause := fmt.Errorf("boc decode failed")
	err := types.NewTransactionError("send failed", cause)
	var target *TransactionError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
	if target.Unwrap() != cause {
		t.Error("cause should be preserved")
	}
}

func TestErrorsAs_WalletError(t *testing.T) {
	err := types.NewWalletError("init failed", nil)
	var target *WalletError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
}

func TestErrorsAs_InvalidWalletVersionError(t *testing.T) {
	err := types.NewInvalidWalletVersionError("V99")
	var target *InvalidWalletVersionError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
	if target.Version != "V99" {
		t.Errorf("Version = %q, want %q", target.Version, "V99")
	}
	if len(target.SupportedVersions) == 0 {
		t.Error("SupportedVersions should not be empty")
	}
}

func TestErrorsAs_PaymentInitiationError(t *testing.T) {
	err := types.NewPaymentInitiationError("no req_id", nil)
	var target *PaymentInitiationError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
}

// --- TransactionNotConfirmedError ---

func TestErrorsAs_TransactionNotConfirmedError(t *testing.T) {
	cause := fmt.Errorf("some timeout")
	err := types.NewTransactionNotConfirmedError(cause)

	var target *TransactionNotConfirmedError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match *TransactionNotConfirmedError")
	}
	if target.Unwrap() != cause {
		t.Error("cause should be preserved")
	}
	if !strings.Contains(target.Error(), "not confirmed") {
		t.Errorf("Error() should mention 'not confirmed': %s", target.Error())
	}
}

func TestClassifyTxError_NotConfirmed(t *testing.T) {
	// Simulate the exact error tonutils-go returns.
	err := tonwallet.ClassifyTxError("send", ton.ErrTxWasNotConfirmed)

	var target *TransactionNotConfirmedError
	if !errors.As(err, &target) {
		t.Fatalf("expected *TransactionNotConfirmedError, got %T: %v", err, err)
	}
}

func TestClassifyTxError_OtherError(t *testing.T) {
	err := tonwallet.ClassifyTxError("send", fmt.Errorf("boc decode failed"))

	var target *TransactionNotConfirmedError
	if errors.As(err, &target) {
		t.Fatal("should NOT match *TransactionNotConfirmedError for other errors")
	}

	var txErr *TransactionError
	if !errors.As(err, &txErr) {
		t.Fatal("should match *TransactionError")
	}
}

func TestClassifyTxError_WrappedNotConfirmed(t *testing.T) {
	// tonutils-go might wrap ErrTxWasNotConfirmed in fmt.Errorf.
	wrapped := fmt.Errorf("wallet send: %w", ton.ErrTxWasNotConfirmed)
	err := tonwallet.ClassifyTxError("send", wrapped)

	var target *TransactionNotConfirmedError
	if !errors.As(err, &target) {
		t.Fatal("should match *TransactionNotConfirmedError even when cause is wrapped")
	}
}

// --- Error chain wrapping with fmt.Errorf %w ---

func TestErrorChain_FmtErrorfWrap(t *testing.T) {
	inner := types.NewNetworkError("timeout", 0, nil)
	wrapped := fmt.Errorf("purchase step 2: %w", inner)

	var target *NetworkError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should find NetworkError through fmt.Errorf wrapping")
	}
	if target.StatusCode != 0 {
		t.Errorf("StatusCode = %d", target.StatusCode)
	}
}
