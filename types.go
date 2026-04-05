package fragment

import (
	"github.com/Darkildo/fragment-api-go/internal/types"
)

// Type aliases re-export all public types from internal/types.
// This allows external consumers to use fragment.WalletVersion etc.
// while the actual definitions live in internal/types (shared by all
// internal packages without circular imports).

// WalletVersion represents a supported TON wallet contract version.
type WalletVersion = types.WalletVersion

const (
	// WalletV3R1 is the legacy V3R1 wallet contract.
	WalletV3R1 = types.WalletV3R1
	// WalletV3R2 is the legacy V3R2 wallet contract.
	WalletV3R2 = types.WalletV3R2
	// WalletV4R2 is the most common wallet contract (recommended default).
	WalletV4R2 = types.WalletV4R2
	// WalletV5R1 is the latest wallet contract with modern features.
	WalletV5R1 = types.WalletV5R1
	// WalletW5 is an alias for [WalletV5R1].
	WalletW5 = types.WalletW5
)

// UserInfo contains information about a Telegram user retrieved from
// the Fragment API.
type UserInfo = types.UserInfo

// PurchaseResult contains the result of a Stars / Premium / TON top-up
// purchase operation.
type PurchaseResult = types.PurchaseResult

// TransferResult contains the result of a direct TON transfer.
type TransferResult = types.TransferResult

// WalletBalance contains the current wallet balance and metadata.
type WalletBalance = types.WalletBalance

// WalletInfo contains metadata about the wallet configuration.
type WalletInfo = types.WalletInfo
