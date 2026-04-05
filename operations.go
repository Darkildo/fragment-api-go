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

// TransferTON sends TON directly to any wallet address or Telegram username.
//
// Parameters:
//   - toAddress: destination — a TON address or "username.t.me" format.
//   - amountTON: amount to transfer in TON.
//   - memo:      text comment attached to the transaction (pass "" for none).
func (c *Client) TransferTON(ctx context.Context, toAddress string, amountTON float64, memo string) (*TransferResult, error) {
	return c.wallet.TransferTON(ctx, toAddress, amountTON, memo)
}
