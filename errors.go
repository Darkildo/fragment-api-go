package fragment

import (
	"github.com/Darkildo/fragment-api-go/internal/types"
)

// Error type aliases re-export all error types from internal/types.
// External consumers use fragment.APIError, fragment.AuthenticationError, etc.

// APIError is the base error type for all Fragment API errors.
// Use [errors.As] to match specific subtypes.
type APIError = types.APIError

// AuthenticationError indicates that the session has expired or the
// cookies / hash are invalid.
type AuthenticationError = types.AuthenticationError

// UserNotFoundError indicates that the target user does not exist on Telegram
// or cannot receive the specified payment type.
type UserNotFoundError = types.UserNotFoundError

// InvalidAmountError indicates that the quantity or amount is outside
// the valid range.
type InvalidAmountError = types.InvalidAmountError

// InsufficientBalanceError indicates that the wallet does not have enough
// TON to complete the transaction.
type InsufficientBalanceError = types.InsufficientBalanceError

// PaymentInitiationError indicates that the Fragment API rejected the
// payment initiation request.
type PaymentInitiationError = types.PaymentInitiationError

// TransactionError indicates that the blockchain transaction execution failed.
type TransactionError = types.TransactionError

// TransactionNotConfirmedError indicates that a transaction was sent to the
// TON network but was not confirmed within the context deadline.
type TransactionNotConfirmedError = types.TransactionNotConfirmedError

// NetworkError indicates that an HTTP request to Fragment.com failed.
type NetworkError = types.NetworkError

// RateLimitError indicates that the API rate limit has been exceeded.
type RateLimitError = types.RateLimitError

// WalletError indicates a generic wallet operation failure.
type WalletError = types.WalletError

// InvalidWalletVersionError indicates that the specified wallet version
// is not supported.
type InvalidWalletVersionError = types.InvalidWalletVersionError
