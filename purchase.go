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
//  1. Init request  → get req_id
//  2. Get link      → get transaction message (address, amount, BOC)
//  3. Check balance
//  4. Send transaction
//  5. Return PurchaseResult
func (c *Client) executePurchase(ctx context.Context, user *UserInfo, p purchaseParams) (*PurchaseResult, error) {
	// Step 1: initiate purchase
	initData := map[string]string{
		"method":    p.initMethod,
		"recipient": user.Recipient,
	}
	for k, v := range p.extra {
		initData[k] = v
	}

	initResp, err := c.core.makeRequest(ctx, initData)
	if err != nil {
		return &PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	reqID, _ := extractString(initResp, "req_id")
	if reqID == "" {
		return &PurchaseResult{
			Success: false,
			Error:   fmt.Sprintf("no req_id from %s", p.initMethod),
			User:    user,
		}, nil
	}

	// Step 2: get transaction data
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
		return &PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	txMsg, err := extractTransactionMsg(linkResp)
	if err != nil {
		return &PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	// Step 3: send transaction via wallet
	txHash, err := c.wallet.sendTransaction(ctx, txMsg.Address, txMsg.Amount, txMsg.Payload)
	if err != nil {
		return &PurchaseResult{
			Success:        false,
			Error:          fmt.Sprintf("transaction failed: %v", err),
			User:           user,
			BalanceChecked: true,
		}, nil
	}

	cost, _ := nanoToTON(txMsg.Amount)

	return &PurchaseResult{
		Success:         true,
		TransactionHash: txHash,
		User:            user,
		BalanceChecked:  true,
		RequiredAmount:  cost,
	}, nil
}
