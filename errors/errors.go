// Package errors defines the error types for the Fragment API client.
//
// Error hierarchy:
//
//	FragmentAPIError (base)
//	├── AuthenticationError       - session expired or invalid credentials
//	├── UserNotFoundError         - user/recipient not found on Telegram
//	├── InvalidAmountError        - quantity/amount out of valid range
//	├── InsufficientBalanceError  - wallet balance too low for transaction
//	├── PaymentInitiationError    - Fragment API rejected payment initiation
//	├── TransactionError          - blockchain transaction execution failed
//	├── NetworkError              - HTTP request failed
//	├── RateLimitError            - rate limit exceeded
//	└── WalletError               - generic wallet operation failure
//	    └── InvalidWalletVersionError - unsupported TON wallet version
package errors

import "fmt"

// FragmentAPIError is the base error type for all Fragment API errors.
type FragmentAPIError struct {
	Message string
	Cause   error
}

func (e *FragmentAPIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *FragmentAPIError) Unwrap() error {
	return e.Cause
}

// NewFragmentAPIError creates a new base Fragment API error.
func NewFragmentAPIError(message string, cause error) *FragmentAPIError {
	return &FragmentAPIError{Message: message, Cause: cause}
}

// AuthenticationError is returned when session expires or credentials are invalid.
type AuthenticationError struct {
	FragmentAPIError
}

func NewAuthenticationError(message string, cause error) *AuthenticationError {
	return &AuthenticationError{FragmentAPIError{Message: message, Cause: cause}}
}

// UserNotFoundError is returned when the target user doesn't exist on Telegram
// or cannot receive the specified payment type.
type UserNotFoundError struct {
	FragmentAPIError
	Username string
}

func NewUserNotFoundError(username string, cause error) *UserNotFoundError {
	return &UserNotFoundError{
		FragmentAPIError: FragmentAPIError{
			Message: fmt.Sprintf("user not found: %s", username),
			Cause:   cause,
		},
		Username: username,
	}
}

// InvalidAmountError is returned when the quantity or amount is out of valid range.
type InvalidAmountError struct {
	FragmentAPIError
	Amount   interface{}
	MinValue int
	MaxValue int
}

func NewInvalidAmountError(amount interface{}, minVal, maxVal int) *InvalidAmountError {
	return &InvalidAmountError{
		FragmentAPIError: FragmentAPIError{
			Message: fmt.Sprintf("invalid amount %v: must be between %d and %d", amount, minVal, maxVal),
		},
		Amount:   amount,
		MinValue: minVal,
		MaxValue: maxVal,
	}
}

// InsufficientBalanceError is returned when the wallet balance is too low.
type InsufficientBalanceError struct {
	FragmentAPIError
	Required float64
	Current  float64
}

func NewInsufficientBalanceError(required, current float64) *InsufficientBalanceError {
	return &InsufficientBalanceError{
		FragmentAPIError: FragmentAPIError{
			Message: fmt.Sprintf("insufficient balance: required %.6f TON, current %.6f TON", required, current),
		},
		Required: required,
		Current:  current,
	}
}

// PaymentInitiationError is returned when Fragment API rejects payment initiation.
type PaymentInitiationError struct {
	FragmentAPIError
}

func NewPaymentInitiationError(message string, cause error) *PaymentInitiationError {
	return &PaymentInitiationError{FragmentAPIError{Message: message, Cause: cause}}
}

// TransactionError is returned when blockchain transaction execution fails.
type TransactionError struct {
	FragmentAPIError
}

func NewTransactionError(message string, cause error) *TransactionError {
	return &TransactionError{FragmentAPIError{Message: message, Cause: cause}}
}

// NetworkError is returned when an HTTP request fails.
type NetworkError struct {
	FragmentAPIError
	StatusCode int
}

func NewNetworkError(message string, statusCode int, cause error) *NetworkError {
	return &NetworkError{
		FragmentAPIError: FragmentAPIError{Message: message, Cause: cause},
		StatusCode:       statusCode,
	}
}

// RateLimitError is returned when the rate limit is exceeded.
type RateLimitError struct {
	FragmentAPIError
	RetryAfter int // seconds to wait before retrying
}

func NewRateLimitError(retryAfter int) *RateLimitError {
	return &RateLimitError{
		FragmentAPIError: FragmentAPIError{
			Message: fmt.Sprintf("rate limit exceeded, retry after %d seconds", retryAfter),
		},
		RetryAfter: retryAfter,
	}
}

// WalletError is returned when a generic wallet operation fails.
type WalletError struct {
	FragmentAPIError
}

func NewWalletError(message string, cause error) *WalletError {
	return &WalletError{FragmentAPIError{Message: message, Cause: cause}}
}

// InvalidWalletVersionError is returned when an unsupported wallet version is specified.
type InvalidWalletVersionError struct {
	WalletError
	Version           string
	SupportedVersions map[string]string
}

// SupportedWalletVersions is the list of supported TON wallet versions.
var SupportedWalletVersions = map[string]string{
	"V4R2": "WalletV4R2 - Most common wallet version (recommended)",
	"V5R1": "WalletV5R1 - Latest wallet version (also known as W5)",
	"W5":   "WalletV5R1 - Alias for V5R1",
	"V3R2": "WalletV3R2 - Legacy wallet version",
	"V3R1": "WalletV3R1 - Legacy wallet version",
}

func NewInvalidWalletVersionError(version string) *InvalidWalletVersionError {
	msg := fmt.Sprintf("invalid wallet version: '%s'\nSupported wallet versions:", version)
	for v, desc := range SupportedWalletVersions {
		msg += fmt.Sprintf("\n  - %s: %s", v, desc)
	}
	return &InvalidWalletVersionError{
		WalletError:       WalletError{FragmentAPIError{Message: msg}},
		Version:           version,
		SupportedVersions: SupportedWalletVersions,
	}
}

// IsFragmentAPIError checks whether an error is a Fragment API error.
func IsFragmentAPIError(err error) bool {
	_, ok := err.(*FragmentAPIError)
	return ok
}
