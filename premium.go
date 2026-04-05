package fragment

import (
	"context"
	"fmt"
)

// GiftPremium gifts a Telegram Premium subscription.
//
// Parameters:
//   - username:   recipient's Telegram username.
//   - months:     subscription duration — must be 3, 6, or 12.
//   - showSender: when true the sender's identity is visible to the recipient.
func (c *Client) GiftPremium(ctx context.Context, username string, months int, showSender bool) (*PurchaseResult, error) {
	clean, err := validateUsername(username)
	if err != nil {
		return nil, newUserNotFoundError(username, err)
	}
	if err := validatePremiumMonths(months); err != nil {
		return nil, newInvalidAmountError(months, 3, 12, err)
	}

	user, err := c.checkUser(ctx, clean, "searchPremiumGiftRecipient", nil)
	if err != nil {
		return nil, err
	}

	return c.executePurchase(ctx, user, purchaseParams{
		initMethod: "initGiftPremiumRequest",
		linkMethod: "getGiftPremiumLink",
		extra:      map[string]string{"months": fmt.Sprintf("%d", months)},
		showSender: showSender,
	})
}
