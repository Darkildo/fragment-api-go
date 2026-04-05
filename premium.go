package fragment

import (
	"context"
	"fmt"

	"github.com/Darkildo/fragment-api-go/internal/helpers"
	"github.com/Darkildo/fragment-api-go/internal/purchase"
	"github.com/Darkildo/fragment-api-go/internal/types"
)

// GiftPremium gifts a Telegram Premium subscription.
//
// Parameters:
//   - username:   recipient's Telegram username.
//   - months:     subscription duration — must be 3, 6, or 12.
//   - showSender: when true the sender's identity is visible to the recipient.
func (c *Client) GiftPremium(ctx context.Context, username string, months int, showSender bool) (*PurchaseResult, error) {
	clean, err := helpers.ValidateUsername(username)
	if err != nil {
		return nil, types.NewUserNotFoundError(username, err)
	}
	if err := helpers.ValidatePremiumMonths(months); err != nil {
		return nil, types.NewInvalidAmountError(months, 3, 12, err)
	}

	user, err := c.checkUser(ctx, clean, "searchPremiumGiftRecipient", nil)
	if err != nil {
		return nil, err
	}

	return purchase.Execute(ctx, c.core, c.wallet, c.log, user, purchase.Params{
		InitMethod: "initGiftPremiumRequest",
		LinkMethod: "getGiftPremiumLink",
		Extra:      map[string]string{"months": fmt.Sprintf("%d", months)},
		ShowSender: showSender,
	})
}
