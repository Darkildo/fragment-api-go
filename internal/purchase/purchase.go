// Package purchase implements the common purchase flow for Stars, Premium,
// and TON Ads top-up operations via the Fragment API.
package purchase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Darkildo/fragment-api-go/internal/helpers"
	"github.com/Darkildo/fragment-api-go/internal/httpcore"
	"github.com/Darkildo/fragment-api-go/internal/tonwallet"
	"github.com/Darkildo/fragment-api-go/internal/types"
)

// Params groups the parameters that differ between BuyStars,
// GiftPremium, and TopupTON.
type Params struct {
	InitMethod string            // e.g. "initBuyStarsRequest"
	LinkMethod string            // e.g. "getBuyStarsLink"
	Extra      map[string]string // method-specific fields (quantity, months, amount)
	ShowSender bool
}

// Execute runs the common purchase flow against the Fragment API:
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
func Execute(ctx context.Context, core *httpcore.Core, wm *tonwallet.Manager, log *slog.Logger, user *types.UserInfo, p Params) (*types.PurchaseResult, error) {
	log.Debug("initiating purchase",
		"method", p.InitMethod,
		"recipient", user.Recipient,
	)

	// Step 1: initiate purchase.
	initData := map[string]string{
		"method":    p.InitMethod,
		"recipient": user.Recipient,
	}
	for k, v := range p.Extra {
		initData[k] = v
	}

	initResp, err := core.MakeRequest(ctx, initData)
	if err != nil {
		return &types.PurchaseResult{User: user}, fmt.Errorf("init %s: %w", p.InitMethod, err)
	}

	// Check for API error.
	if apiErr := ExtractAPIError(initResp); apiErr != nil {
		return &types.PurchaseResult{User: user},
			types.NewPaymentInitiationError(fmt.Sprintf("%s: %s", p.InitMethod, apiErr.Error()), apiErr)
	}

	// req_id is at top-level in Fragment response.
	reqID, _ := initResp["req_id"].(string)
	if reqID == "" {
		return &types.PurchaseResult{User: user},
			types.NewPaymentInitiationError(fmt.Sprintf("no req_id from %s", p.InitMethod), nil)
	}

	// Step 2: get transaction data.
	showSenderStr := "0"
	if p.ShowSender {
		showSenderStr = "1"
	}
	linkResp, err := core.MakeRequest(ctx, map[string]string{
		"method":      p.LinkMethod,
		"id":          reqID,
		"show_sender": showSenderStr,
		"transaction": "1",
	})
	if err != nil {
		return &types.PurchaseResult{User: user}, fmt.Errorf("get link %s: %w", p.LinkMethod, err)
	}

	if apiErr := ExtractAPIError(linkResp); apiErr != nil {
		return &types.PurchaseResult{User: user},
			types.NewTransactionError(fmt.Sprintf("%s: %s", p.LinkMethod, apiErr.Error()), apiErr)
	}

	txMsg, err := helpers.ExtractTransactionMsg(linkResp)
	if err != nil {
		return &types.PurchaseResult{User: user}, fmt.Errorf("extract tx message: %w", err)
	}

	// Step 3: send transaction via wallet.
	log.Info("sending transaction",
		"destination", txMsg.Address,
		"amount_nano", txMsg.Amount,
	)

	txHash, err := wm.SendTransaction(ctx, txMsg.Address, txMsg.Amount, txMsg.Payload)
	if err != nil {
		return &types.PurchaseResult{User: user, BalanceChecked: true},
			types.NewTransactionError("send fragment transaction", err)
	}

	cost, _ := helpers.NanoToTON(txMsg.Amount)

	log.Info("purchase completed",
		"tx_hash", txHash,
		"cost_ton", cost,
	)

	return &types.PurchaseResult{
		Success:         true,
		TransactionHash: txHash,
		User:            user,
		BalanceChecked:  true,
		RequiredAmount:  cost,
	}, nil
}

// ExtractAPIError checks if the Fragment API response contains an "error" field.
// Returns nil if no error is present.
func ExtractAPIError(resp map[string]interface{}) error {
	errVal, ok := resp["error"]
	if !ok {
		return nil
	}
	switch v := errVal.(type) {
	case string:
		if v == "Session expired" || v == "AUTH_SESSION_EXPIRED" {
			return types.NewAuthenticationError(v, nil)
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
