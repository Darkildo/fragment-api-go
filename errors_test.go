package fragment

import (
	"errors"
	"fmt"
	"strings"
	"testing"
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
	err := newAuthenticationError("session expired", nil)
	var target *AuthenticationError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match *AuthenticationError")
	}
	if target.Message != "session expired" {
		t.Errorf("Message = %q", target.Message)
	}

	// Note: errors.As does NOT traverse struct embedding.
	// *AuthenticationError embeds APIError by value, so errors.As
	// with *APIError won't match. The correct way to match the base
	// is via the concrete type. This is standard Go errors behaviour.
}

func TestErrorsAs_UserNotFoundError(t *testing.T) {
	cause := fmt.Errorf("invalid username")
	err := newUserNotFoundError("test_user", cause)
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
	err := newInvalidAmountError(0, 1, 100, cause)
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
	err := newInsufficientBalanceError(1.5, 0.3)
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
	err := newNetworkError("request failed", 503, cause)
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
	err := newRateLimitError(60)
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
	err := newTransactionError("send failed", cause)
	var target *TransactionError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
	if target.Unwrap() != cause {
		t.Error("cause should be preserved")
	}
}

func TestErrorsAs_WalletError(t *testing.T) {
	err := newWalletError("init failed", nil)
	var target *WalletError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
}

func TestErrorsAs_InvalidWalletVersionError(t *testing.T) {
	err := newInvalidWalletVersionError("V99")
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

	// Note: errors.As does NOT traverse struct embedding.
	// *InvalidWalletVersionError embeds WalletError by value, so
	// errors.As with *WalletError won't match through embedding alone.
	// This is standard Go errors behaviour — match the concrete type.
}

func TestErrorsAs_PaymentInitiationError(t *testing.T) {
	err := newPaymentInitiationError("no req_id", nil)
	var target *PaymentInitiationError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match")
	}
}

// --- Error chain wrapping with fmt.Errorf %w ---

func TestErrorChain_FmtErrorfWrap(t *testing.T) {
	inner := newNetworkError("timeout", 0, nil)
	wrapped := fmt.Errorf("purchase step 2: %w", inner)

	var target *NetworkError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should find NetworkError through fmt.Errorf wrapping")
	}
	if target.StatusCode != 0 {
		t.Errorf("StatusCode = %d", target.StatusCode)
	}
}
