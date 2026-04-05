package fragment

import (
	"context"
	"fmt"

	"github.com/Darkildo/fragment-api-go/internal/helpers"
	"github.com/Darkildo/fragment-api-go/internal/purchase"
	"github.com/Darkildo/fragment-api-go/internal/types"
)

// TopupTON tops up TON balance for a Telegram Ads account.
//
// Parameters:
//   - username:   target username or ads account.
//   - amount:     amount of TON to transfer (1-999 999).
//   - showSender: when true the sender's identity is visible to the recipient.
func (c *Client) TopupTON(ctx context.Context, username string, amount int, showSender bool) (*PurchaseResult, error) {
	clean, err := helpers.ValidateUsername(username)
	if err != nil {
		return nil, types.NewUserNotFoundError(username, err)
	}
	if err := helpers.ValidateAmount(amount, 1, 999999); err != nil {
		return nil, types.NewInvalidAmountError(amount, 1, 999999, err)
	}

	user, err := c.checkUser(ctx, clean, "searchAdsTopupRecipient", nil)
	if err != nil {
		return nil, err
	}

	return purchase.Execute(ctx, c.core, c.wallet, c.log, user, purchase.Params{
		InitMethod: "initAdsTopupRequest",
		LinkMethod: "getAdsTopupLink",
		Extra:      map[string]string{"amount": fmt.Sprintf("%d", amount)},
		ShowSender: showSender,
	})
}
