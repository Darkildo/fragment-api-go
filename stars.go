package fragment

import (
	"context"
	"fmt"
)

// BuyStars sends Telegram Stars to a user.
//
// Parameters:
//   - username:   recipient's Telegram username (5-32 characters, '@' optional).
//   - quantity:   number of stars to send (1-999 999).
//   - showSender: when true the sender's identity is visible to the recipient.
func (c *Client) BuyStars(ctx context.Context, username string, quantity int, showSender bool) (*PurchaseResult, error) {
	clean, err := validateUsername(username)
	if err != nil {
		return nil, newUserNotFoundError(username, err)
	}
	if err := validateAmount(quantity, 50, 999999); err != nil {
		return nil, newInvalidAmountError(quantity, 50, 999999, err)
	}

	user, err := c.checkUser(ctx, clean, "searchStarsRecipient", map[string]string{"quantity": ""})
	if err != nil {
		return nil, err
	}

	return c.executePurchase(ctx, user, purchaseParams{
		initMethod: "initBuyStarsRequest",
		linkMethod: "getBuyStarsLink",
		extra:      map[string]string{"quantity": fmt.Sprintf("%d", quantity)},
		showSender: showSender,
	})
}
