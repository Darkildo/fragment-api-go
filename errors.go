package fragment

import "fmt"

// APIError is the base error type for all Fragment API errors.
// Use [errors.As] to match specific subtypes.
type APIError struct {
	Message string
	Cause   error
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *APIError) Unwrap() error { return e.Cause }

// AuthenticationError indicates that the session has expired or the
// cookies / hash are invalid.
type AuthenticationError struct{ APIError }

// UserNotFoundError indicates that the target user does not exist on Telegram
// or cannot receive the specified payment type.
type UserNotFoundError struct {
	APIError
	Username string
}

// InvalidAmountError indicates that the quantity or amount is outside
// the valid range.
type InvalidAmountError struct {
	APIError
	Amount   interface{}
	MinValue int
	MaxValue int
}

// InsufficientBalanceError indicates that the wallet does not have enough
// TON to complete the transaction.
type InsufficientBalanceError struct {
	APIError
	Required float64
	Current  float64
}

// PaymentInitiationError indicates that the Fragment API rejected the
// payment initiation request.
type PaymentInitiationError struct{ APIError }

// TransactionError indicates that the blockchain transaction execution failed.
type TransactionError struct{ APIError }

// NetworkError indicates that an HTTP request to Fragment.com failed.
type NetworkError struct {
	APIError
	StatusCode int
}

// RateLimitError indicates that the API rate limit has been exceeded.
type RateLimitError struct {
	APIError
	RetryAfter int // seconds
}

// WalletError indicates a generic wallet operation failure.
type WalletError struct{ APIError }

// InvalidWalletVersionError indicates that the specified wallet version
// is not supported.
type InvalidWalletVersionError struct {
	WalletError
	Version           string
	SupportedVersions map[string]string
}

// supportedWalletVersions enumerates all recognised TON wallet versions.
var supportedWalletVersions = map[string]string{
	"V4R2": "WalletV4R2 — most common wallet version (recommended)",
	"V5R1": "WalletV5R1 — latest wallet version (also known as W5)",
	"W5":   "WalletV5R1 — alias for V5R1",
	"V3R2": "WalletV3R2 — legacy wallet version",
	"V3R1": "WalletV3R1 — legacy wallet version",
}

// --- constructors (unexported, used by internal code) ---

func newAuthenticationError(msg string, cause error) *AuthenticationError {
	return &AuthenticationError{APIError{Message: msg, Cause: cause}}
}

func newUserNotFoundError(username string, cause error) *UserNotFoundError {
	return &UserNotFoundError{
		APIError: APIError{Message: fmt.Sprintf("user not found: %s", username), Cause: cause},
		Username: username,
	}
}

func newInvalidAmountError(amount interface{}, min, max int) *InvalidAmountError {
	return &InvalidAmountError{
		APIError: APIError{Message: fmt.Sprintf("invalid amount %v: must be between %d and %d", amount, min, max)},
		Amount:   amount,
		MinValue: min,
		MaxValue: max,
	}
}

func newInsufficientBalanceError(required, current float64) *InsufficientBalanceError {
	return &InsufficientBalanceError{
		APIError: APIError{Message: fmt.Sprintf("insufficient balance: need %.6f TON, have %.6f TON", required, current)},
		Required: required,
		Current:  current,
	}
}

func newPaymentInitiationError(msg string, cause error) *PaymentInitiationError {
	return &PaymentInitiationError{APIError{Message: msg, Cause: cause}}
}

func newTransactionError(msg string, cause error) *TransactionError {
	return &TransactionError{APIError{Message: msg, Cause: cause}}
}

func newNetworkError(msg string, statusCode int, cause error) *NetworkError {
	return &NetworkError{APIError: APIError{Message: msg, Cause: cause}, StatusCode: statusCode}
}

func newRateLimitError(retryAfter int) *RateLimitError {
	return &RateLimitError{
		APIError:   APIError{Message: fmt.Sprintf("rate limit exceeded, retry after %d seconds", retryAfter)},
		RetryAfter: retryAfter,
	}
}

func newWalletError(msg string, cause error) *WalletError {
	return &WalletError{APIError{Message: msg, Cause: cause}}
}

func newInvalidWalletVersionError(version string) *InvalidWalletVersionError {
	msg := fmt.Sprintf("invalid wallet version: %q\nSupported wallet versions:", version)
	for v, desc := range supportedWalletVersions {
		msg += fmt.Sprintf("\n  - %s: %s", v, desc)
	}
	return &InvalidWalletVersionError{
		WalletError:       WalletError{APIError{Message: msg, Cause: nil}},
		Version:           version,
		SupportedVersions: supportedWalletVersions,
	}
}
