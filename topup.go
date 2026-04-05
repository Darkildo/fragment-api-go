package fragment

import (
	"context"
	"fmt"
)

// TopupTON tops up TON balance for a Telegram Ads account.
//
// Parameters:
//   - username:   target username or ads account.
//   - amount:     amount of TON to transfer (1-999 999).
//   - showSender: when true the sender's identity is visible to the recipient.
func (c *Client) TopupTON(ctx context.Context, username string, amount int, showSender bool) (*PurchaseResult, error) {
	clean, err := validateUsername(username)
	if err != nil {
		return nil, newUserNotFoundError(username, err)
	}
	if err := validateAmount(amount, 1, 999999); err != nil {
		return nil, newInvalidAmountError(amount, 1, 999999, err)
	}

	user, err := c.checkUser(ctx, clean, "searchAdsTopupRecipient", nil)
	if err != nil {
		return nil, err
	}

	return c.executePurchase(ctx, user, purchaseParams{
		initMethod: "initAdsTopupRequest",
		linkMethod: "getAdsTopupLink",
		extra:      map[string]string{"amount": fmt.Sprintf("%d", amount)},
		showSender: showSender,
	})
}
