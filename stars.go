package fragment

import (
	"context"
	"fmt"

	"github.com/Darkildo/fragment-api-go/internal/helpers"
	"github.com/Darkildo/fragment-api-go/internal/purchase"
	"github.com/Darkildo/fragment-api-go/internal/types"
)

// BuyStars sends Telegram Stars to a user.
//
// Parameters:
//   - username:   recipient's Telegram username (5-32 characters, '@' optional).
//   - quantity:   number of stars to send (1-999 999).
//   - showSender: when true the sender's identity is visible to the recipient.
func (c *Client) BuyStars(ctx context.Context, username string, quantity int, showSender bool) (*PurchaseResult, error) {
	clean, err := helpers.ValidateUsername(username)
	if err != nil {
		return nil, types.NewUserNotFoundError(username, err)
	}
	if err := helpers.ValidateAmount(quantity, 50, 999999); err != nil {
		return nil, types.NewInvalidAmountError(quantity, 50, 999999, err)
	}

	user, err := c.checkUser(ctx, clean, "searchStarsRecipient", map[string]string{"quantity": ""})
	if err != nil {
		return nil, err
	}

	return purchase.Execute(ctx, c.core, c.wallet, c.log, user, purchase.Params{
		InitMethod: "initBuyStarsRequest",
		LinkMethod: "getBuyStarsLink",
		Extra:      map[string]string{"quantity": fmt.Sprintf("%d", quantity)},
		ShowSender: showSender,
	})
}
