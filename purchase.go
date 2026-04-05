package fragment

import (
	"context"
	"fmt"
)

// purchaseParams groups the parameters that differ between BuyStars,
// GiftPremium, and TopupTON.
type purchaseParams struct {
	initMethod string            // e.g. "initBuyStarsRequest"
	linkMethod string            // e.g. "getBuyStarsLink"
	extra      map[string]string // method-specific fields (quantity, months, amount)
	showSender bool
}

// executePurchase runs the common purchase flow:
//
//  1. Init request  -> get req_id
//  2. Get link      -> get transaction message (address, amount, BOC)
//  3. Send transaction
//  4. Return PurchaseResult
//
// All errors are returned as typed Go errors (never swallowed into strings).
func (c *Client) executePurchase(ctx context.Context, user *UserInfo, p purchaseParams) (*PurchaseResult, error) {
	c.log.Debug("initiating purchase",
		"method", p.initMethod,
		"recipient", user.Recipient,
	)

	// Step 1: initiate purchase.
	initData := map[string]string{
		"method":    p.initMethod,
		"recipient": user.Recipient,
	}
	for k, v := range p.extra {
		initData[k] = v
	}

	initResp, err := c.core.makeRequest(ctx, initData)
	if err != nil {
		return &PurchaseResult{User: user}, fmt.Errorf("init %s: %w", p.initMethod, err)
	}

	reqID, _ := extractString(initResp, "req_id")
	if reqID == "" {
		return &PurchaseResult{User: user},
			newPaymentInitiationError(fmt.Sprintf("no req_id from %s", p.initMethod), nil)
	}

	// Step 2: get transaction data.
	showSenderStr := "0"
	if p.showSender {
		showSenderStr = "1"
	}
	linkResp, err := c.core.makeRequest(ctx, map[string]string{
		"method":      p.linkMethod,
		"id":          reqID,
		"show_sender": showSenderStr,
	})
	if err != nil {
		return &PurchaseResult{User: user}, fmt.Errorf("get link %s: %w", p.linkMethod, err)
	}

	txMsg, err := extractTransactionMsg(linkResp)
	if err != nil {
		return &PurchaseResult{User: user}, fmt.Errorf("extract tx message: %w", err)
	}

	// Step 3: send transaction via wallet.
	c.log.Info("sending transaction",
		"destination", txMsg.Address,
		"amount_nano", txMsg.Amount,
	)

	txHash, err := c.wallet.sendTransaction(ctx, txMsg.Address, txMsg.Amount, txMsg.Payload)
	if err != nil {
		return &PurchaseResult{User: user, BalanceChecked: true},
			newTransactionError("send fragment transaction", err)
	}

	cost, _ := nanoToTON(txMsg.Amount)

	c.log.Info("purchase completed",
		"tx_hash", txHash,
		"cost_ton", cost,
	)

	return &PurchaseResult{
		Success:         true,
		TransactionHash: txHash,
		User:            user,
		BalanceChecked:  true,
		RequiredAmount:  cost,
	}, nil
}
