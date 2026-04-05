package fragment

import "context"

// TransferTON sends TON directly to any wallet address or Telegram username.
//
// Parameters:
//   - toAddress: destination — a TON address or "username.t.me" format.
//   - amountTON: amount to transfer in TON.
//   - memo:      text comment attached to the transaction (pass "" for none).
func (c *Client) TransferTON(ctx context.Context, toAddress string, amountTON float64, memo string) (*TransferResult, error) {
	return c.wallet.transferTON(ctx, toAddress, amountTON, memo)
}
