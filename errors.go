package fragment

import "fmt"

// APIError is the base error type for all Fragment API errors.
// Use [errors.As] to match specific subtypes.
type APIError struct {
	// Message is a human-readable description of the error.
	Message string
	// Cause is the underlying error, if any. Accessible via [errors.Unwrap].
	Cause error
}

// Error returns the error message, including the cause if present.
func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause for use with [errors.Is] and [errors.As].
func (e *APIError) Unwrap() error { return e.Cause }

// AuthenticationError indicates that the session has expired or the
// cookies / hash are invalid.
type AuthenticationError struct{ APIError }

// UserNotFoundError indicates that the target user does not exist on Telegram
// or cannot receive the specified payment type.
type UserNotFoundError struct {
	APIError
	// Username is the Telegram username that was not found.
	Username string
}

// InvalidAmountError indicates that the quantity or amount is outside
// the valid range.
type InvalidAmountError struct {
	APIError
	// Amount is the value that was rejected.
	Amount int
	// MinValue is the minimum allowed value (inclusive).
	MinValue int
	// MaxValue is the maximum allowed value (inclusive).
	MaxValue int
}

// InsufficientBalanceError indicates that the wallet does not have enough
// TON to complete the transaction.
type InsufficientBalanceError struct {
	APIError
	// Required is the total TON needed (amount + fees).
	Required float64
	// Current is the wallet's current balance in TON.
	Current float64
}

// PaymentInitiationError indicates that the Fragment API rejected the
// payment initiation request.
type PaymentInitiationError struct{ APIError }

// TransactionError indicates that the blockchain transaction execution failed.
type TransactionError struct{ APIError }

// TransactionNotConfirmedError indicates that a transaction was sent to the
// TON network but was not confirmed within the context deadline.
// The transaction may still be confirmed later — callers should check the
// blockchain state before retrying to avoid double-spending.
type TransactionNotConfirmedError struct {
	TransactionError
}

// NetworkError indicates that an HTTP request to Fragment.com failed.
type NetworkError struct {
	APIError
	// StatusCode is the HTTP status code (0 if the request did not complete).
	StatusCode int
}

// RateLimitError indicates that the API rate limit has been exceeded.
type RateLimitError struct {
	APIError
	// RetryAfter is the recommended wait time in seconds before retrying.
	RetryAfter int
}

// WalletError indicates a generic wallet operation failure.
type WalletError struct{ APIError }

// InvalidWalletVersionError indicates that the specified wallet version
// is not supported.
type InvalidWalletVersionError struct {
	WalletError
	// Version is the unsupported version string that was provided.
	Version string
	// SupportedVersions lists all valid wallet versions.
	SupportedVersions []WalletVersion
}

// supportedWalletVersionsList enumerates all recognised TON wallet versions.
var supportedWalletVersionsList = []WalletVersion{
	WalletV4R2, WalletV5R1, WalletW5, WalletV3R2, WalletV3R1,
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

func newInvalidAmountError(amount, min, max int, cause error) *InvalidAmountError {
	return &InvalidAmountError{
		APIError: APIError{
			Message: fmt.Sprintf("invalid amount %d: must be between %d and %d", amount, min, max),
			Cause:   cause,
		},
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

func newTransactionNotConfirmedError(cause error) *TransactionNotConfirmedError {
	return &TransactionNotConfirmedError{
		TransactionError: TransactionError{APIError{
			Message: "transaction sent but not confirmed within deadline (may still confirm later)",
			Cause:   cause,
		}},
	}
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
	for _, v := range supportedWalletVersionsList {
		msg += fmt.Sprintf("\n  - %s", v)
	}
	return &InvalidWalletVersionError{
		WalletError:       WalletError{APIError{Message: msg, Cause: nil}},
		Version:           version,
		SupportedVersions: supportedWalletVersionsList,
	}
}
