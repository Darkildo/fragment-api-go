package fragment

import (
	"context"
	"errors"
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

// executePurchase runs the common purchase flow against the Fragment API:
//
//  1. Init request  -> get req_id  (top-level "req_id" in response)
//  2. Get link      -> get transaction message from "transaction.messages[0]"
//  3. Send transaction via wallet
//  4. Return PurchaseResult
//
// Fragment API response format:
//
//	Success: {"req_id": "..."} for init, {"transaction": {"messages": [...]}} for link
//	Error:   {"error": "..."} at top level
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

	// Check for API error.
	if apiErr := extractAPIError(initResp); apiErr != nil {
		return &PurchaseResult{User: user},
			newPaymentInitiationError(fmt.Sprintf("%s: %s", p.initMethod, apiErr.Error()), apiErr)
	}

	// req_id is at top-level in Fragment response.
	reqID, _ := initResp["req_id"].(string)
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
		"transaction": "1",
	})
	if err != nil {
		return &PurchaseResult{User: user}, fmt.Errorf("get link %s: %w", p.linkMethod, err)
	}

	if apiErr := extractAPIError(linkResp); apiErr != nil {
		return &PurchaseResult{User: user},
			newTransactionError(fmt.Sprintf("%s: %s", p.linkMethod, apiErr.Error()), apiErr)
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

// extractAPIError checks if the Fragment API response contains an "error" field.
// Returns nil if no error is present.
func extractAPIError(resp map[string]interface{}) error {
	errVal, ok := resp["error"]
	if !ok {
		return nil
	}
	switch v := errVal.(type) {
	case string:
		if v == "Session expired" || v == "AUTH_SESSION_EXPIRED" {
			return newAuthenticationError(v, nil)
		}
		return errors.New(v)
	case map[string]interface{}:
		msg, _ := v["error"].(string)
		if msg == "" {
			msg = "unknown API error"
		}
		return errors.New(msg)
	default:
		return errors.New("unknown API error")
	}
}
