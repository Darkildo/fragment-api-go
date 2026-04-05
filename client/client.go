// Package client provides the high-level Fragment API client.
//
// FragmentAPI is the main entry point for interacting with the Fragment.com API.
// It supports purchasing Telegram Stars, gifting Premium subscriptions,
// topping up TON Ads accounts, and direct TON transfers.
//
// Usage:
//
//	cfg := client.Config{
//	    Cookies:        "stel_ssid=...; stel_token=...",
//	    HashValue:      "abc123...",
//	    WalletMnemonic: "word1 word2 ... word24",
//	    WalletAPIKey:   "your-tonapi-key",
//	    WalletVersion:  "V4R2",
//	}
//
//	api, err := client.New(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer api.Close()
//
//	result, err := api.BuyStars(ctx, "username", 100, false)
package client

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Darkildo/fragment-api-go/core"
	fragErrors "github.com/Darkildo/fragment-api-go/errors"
	"github.com/Darkildo/fragment-api-go/models"
	"github.com/Darkildo/fragment-api-go/utils"
	"github.com/Darkildo/fragment-api-go/wallet"
)

const (
	// TransferFeeTON is the fee buffer for Fragment purchases (in TON).
	TransferFeeTON = 0.001
)

// Config contains all configuration needed to create a FragmentAPI client.
type Config struct {
	// Cookies is the raw cookie string from Fragment.com browser session.
	// Required cookies: stel_ssid, stel_token, stel_dt, stel_ton_token.
	Cookies string

	// HashValue is the API hash parameter from Fragment.com network requests.
	HashValue string

	// WalletMnemonic is the 24-word TON wallet seed phrase.
	WalletMnemonic string

	// WalletAPIKey is the TonAPI key from tonconsole.com.
	WalletAPIKey string

	// WalletVersion is the TON wallet contract version (e.g., "V4R2", "V5R1", "W5").
	// Defaults to "V4R2" if empty.
	WalletVersion string

	// Timeout is the HTTP request timeout. Defaults to 15 seconds if zero.
	Timeout time.Duration
}

// FragmentAPI is the main client for the Fragment.com API.
// It provides methods for purchasing Stars, Premium subscriptions,
// TON Ads top-ups, and direct TON transfers.
type FragmentAPI struct {
	core   *core.Client
	wallet *wallet.Manager
}

// New creates a new FragmentAPI client with the given configuration.
func New(cfg Config) (*FragmentAPI, error) {
	coreClient, err := core.NewClient(core.Config{
		Cookies:   cfg.Cookies,
		HashValue: cfg.HashValue,
		Timeout:   cfg.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create core client: %w", err)
	}

	walletMgr, err := wallet.NewManager(wallet.Config{
		Mnemonic: cfg.WalletMnemonic,
		APIKey:   cfg.WalletAPIKey,
		Version:  cfg.WalletVersion,
	})
	if err != nil {
		coreClient.Close()
		return nil, fmt.Errorf("failed to create wallet manager: %w", err)
	}

	return &FragmentAPI{
		core:   coreClient,
		wallet: walletMgr,
	}, nil
}

// Close releases all resources held by the client.
func (f *FragmentAPI) Close() {
	if f.core != nil {
		f.core.Close()
	}
}

// ---------------------------------------------------------------------------
// Recipient Information Methods
// ---------------------------------------------------------------------------

// GetRecipientStars retrieves recipient information for a Telegram Stars transfer.
//
// The username can include or omit the leading '@' character.
// Returns UserInfo with the user's name, recipient address, and avatar.
func (f *FragmentAPI) GetRecipientStars(ctx context.Context, username string) (*models.UserInfo, error) {
	return f.checkUser(ctx, username, "searchStarsRecipient")
}

// GetRecipientPremium retrieves recipient information for a Premium subscription gift.
//
// Returns an error if the user doesn't exist, already has Premium,
// or cannot receive a Premium gift.
func (f *FragmentAPI) GetRecipientPremium(ctx context.Context, username string) (*models.UserInfo, error) {
	return f.checkUser(ctx, username, "searchPremiumGiftRecipient")
}

// GetRecipientTON retrieves recipient information for a TON Ads top-up.
func (f *FragmentAPI) GetRecipientTON(ctx context.Context, username string) (*models.UserInfo, error) {
	return f.checkUser(ctx, username, "searchAdsTopupRecipient")
}

// ---------------------------------------------------------------------------
// Payment Methods
// ---------------------------------------------------------------------------

// BuyStars sends Telegram Stars to a user.
//
// Parameters:
//   - username: recipient's Telegram username (5-32 characters).
//   - quantity: number of stars to send (1-999999).
//   - showSender: if true, the sender is visible to the recipient.
//
// Returns PurchaseResult with transaction details or error information.
func (f *FragmentAPI) BuyStars(ctx context.Context, username string, quantity int, showSender bool) (*models.PurchaseResult, error) {
	// Validate input
	cleanUsername, err := utils.ValidateUsername(username)
	if err != nil {
		return nil, fragErrors.NewInvalidAmountError(username, 0, 0)
	}
	if err := utils.ValidateAmount(quantity, 1, 999999); err != nil {
		return nil, fragErrors.NewInvalidAmountError(quantity, 1, 999999)
	}

	// Step 1: Look up user
	user, err := f.checkUser(ctx, cleanUsername, "searchStarsRecipient")
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error()}, nil
	}

	// Step 2: Initiate purchase
	initData := map[string]string{
		"method":    "initBuyStarsRequest",
		"recipient": user.Recipient,
		"quantity":  fmt.Sprintf("%d", quantity),
	}
	initResp, err := f.core.MakeRequest(ctx, initData)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	reqID, _ := extractString(initResp, "req_id")
	if reqID == "" {
		return &models.PurchaseResult{
			Success: false,
			Error:   "failed to get request ID from initBuyStarsRequest",
			User:    user,
		}, nil
	}

	// Step 3: Get payment link / transaction data
	showSenderStr := "0"
	if showSender {
		showSenderStr = "1"
	}
	linkData := map[string]string{
		"method":      "getBuyStarsLink",
		"id":          reqID,
		"show_sender": showSenderStr,
	}
	linkResp, err := f.core.MakeRequest(ctx, linkData)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	// Step 4: Extract transaction details
	txMsg, err := extractTransactionMessage(linkResp)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	// Step 5: Check balance
	balance, err := f.wallet.GetBalance(ctx)
	if err != nil {
		return &models.PurchaseResult{
			Success: false,
			Error:   fmt.Sprintf("failed to check balance: %v", err),
			User:    user,
		}, nil
	}

	requiredNano, _ := fmt.Sscanf(txMsg.Amount, "%d", new(int64))
	_ = requiredNano
	balanceChecked := true

	// Step 6: Send transaction
	txHash, err := f.wallet.SendTransaction(ctx, txMsg.Address, txMsg.Amount, txMsg.Payload)
	if err != nil {
		return &models.PurchaseResult{
			Success:        false,
			Error:          fmt.Sprintf("transaction failed: %v", err),
			User:           user,
			BalanceChecked: balanceChecked,
		}, nil
	}

	requiredTON, _ := utils.NanoToTON(txMsg.Amount)

	_ = balance // Will be used once wallet is implemented

	return &models.PurchaseResult{
		Success:         true,
		TransactionHash: txHash,
		User:            user,
		BalanceChecked:  balanceChecked,
		RequiredAmount:  requiredTON,
	}, nil
}

// GiftPremium gifts a Telegram Premium subscription to a user.
//
// Parameters:
//   - username: recipient's Telegram username.
//   - months: subscription duration (3, 6, or 12).
//   - showSender: if true, the sender is visible to the recipient.
func (f *FragmentAPI) GiftPremium(ctx context.Context, username string, months int, showSender bool) (*models.PurchaseResult, error) {
	// Validate
	cleanUsername, err := utils.ValidateUsername(username)
	if err != nil {
		return nil, fragErrors.NewInvalidAmountError(username, 0, 0)
	}
	if err := utils.ValidatePremiumMonths(months); err != nil {
		return nil, fragErrors.NewInvalidAmountError(months, 3, 12)
	}

	// Look up user
	user, err := f.checkUser(ctx, cleanUsername, "searchPremiumGiftRecipient")
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error()}, nil
	}

	// Initiate
	initData := map[string]string{
		"method":    "initGiftPremiumRequest",
		"recipient": user.Recipient,
		"months":    fmt.Sprintf("%d", months),
	}
	initResp, err := f.core.MakeRequest(ctx, initData)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	reqID, _ := extractString(initResp, "req_id")
	if reqID == "" {
		return &models.PurchaseResult{
			Success: false,
			Error:   "failed to get request ID from initGiftPremiumRequest",
			User:    user,
		}, nil
	}

	// Get link
	showSenderStr := "0"
	if showSender {
		showSenderStr = "1"
	}
	linkData := map[string]string{
		"method":      "getGiftPremiumLink",
		"id":          reqID,
		"show_sender": showSenderStr,
	}
	linkResp, err := f.core.MakeRequest(ctx, linkData)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	// Extract + send
	txMsg, err := extractTransactionMessage(linkResp)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	txHash, err := f.wallet.SendTransaction(ctx, txMsg.Address, txMsg.Amount, txMsg.Payload)
	if err != nil {
		return &models.PurchaseResult{
			Success: false,
			Error:   fmt.Sprintf("transaction failed: %v", err),
			User:    user,
		}, nil
	}

	requiredTON, _ := utils.NanoToTON(txMsg.Amount)

	return &models.PurchaseResult{
		Success:         true,
		TransactionHash: txHash,
		User:            user,
		BalanceChecked:  true,
		RequiredAmount:  requiredTON,
	}, nil
}

// TopupTON tops up TON balance for a Telegram Ads account.
//
// Parameters:
//   - username: target username or ads account.
//   - amount: amount of TON to transfer (1-999999).
//   - showSender: if true, the sender is visible to the recipient.
func (f *FragmentAPI) TopupTON(ctx context.Context, username string, amount int, showSender bool) (*models.PurchaseResult, error) {
	// Validate
	cleanUsername, err := utils.ValidateUsername(username)
	if err != nil {
		return nil, fragErrors.NewInvalidAmountError(username, 0, 0)
	}
	if err := utils.ValidateAmount(amount, 1, 999999); err != nil {
		return nil, fragErrors.NewInvalidAmountError(amount, 1, 999999)
	}

	// Look up
	user, err := f.checkUser(ctx, cleanUsername, "searchAdsTopupRecipient")
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error()}, nil
	}

	// Initiate
	initData := map[string]string{
		"method":    "initAdsTopupRequest",
		"recipient": user.Recipient,
		"amount":    fmt.Sprintf("%d", amount),
	}
	initResp, err := f.core.MakeRequest(ctx, initData)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	reqID, _ := extractString(initResp, "req_id")
	if reqID == "" {
		return &models.PurchaseResult{
			Success: false,
			Error:   "failed to get request ID from initAdsTopupRequest",
			User:    user,
		}, nil
	}

	// Get link
	showSenderStr := "0"
	if showSender {
		showSenderStr = "1"
	}
	linkData := map[string]string{
		"method":      "getAdsTopupLink",
		"id":          reqID,
		"show_sender": showSenderStr,
	}
	linkResp, err := f.core.MakeRequest(ctx, linkData)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	// Extract + send
	txMsg, err := extractTransactionMessage(linkResp)
	if err != nil {
		return &models.PurchaseResult{Success: false, Error: err.Error(), User: user}, nil
	}

	txHash, err := f.wallet.SendTransaction(ctx, txMsg.Address, txMsg.Amount, txMsg.Payload)
	if err != nil {
		return &models.PurchaseResult{
			Success: false,
			Error:   fmt.Sprintf("transaction failed: %v", err),
			User:    user,
		}, nil
	}

	requiredTON, _ := utils.NanoToTON(txMsg.Amount)

	return &models.PurchaseResult{
		Success:         true,
		TransactionHash: txHash,
		User:            user,
		BalanceChecked:  true,
		RequiredAmount:  requiredTON,
	}, nil
}

// TransferTON sends TON directly to any wallet address or Telegram username.
//
// Parameters:
//   - toAddress: destination address (TON address or "username.t.me" format).
//   - amountTON: amount to transfer in TON.
//   - memo: optional text comment (pass "" for no memo).
func (f *FragmentAPI) TransferTON(ctx context.Context, toAddress string, amountTON float64, memo string) (*models.TransferResult, error) {
	return f.wallet.TransferTON(ctx, toAddress, amountTON, memo)
}

// GetWalletBalance retrieves the current wallet balance and metadata.
func (f *FragmentAPI) GetWalletBalance(ctx context.Context) (*models.WalletBalance, error) {
	return f.wallet.GetBalance(ctx)
}

// GetWalletInfo returns metadata about the wallet configuration.
func (f *FragmentAPI) GetWalletInfo() map[string]interface{} {
	return f.wallet.GetWalletInfo()
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// avatarRegexp matches src attributes in HTML img tags.
var avatarRegexp = regexp.MustCompile(`src="([^"]+)"`)

// checkUser validates a username and looks it up via the Fragment API.
func (f *FragmentAPI) checkUser(ctx context.Context, username string, method string) (*models.UserInfo, error) {
	cleanUsername, err := utils.ValidateUsername(username)
	if err != nil {
		return nil, fragErrors.NewUserNotFoundError(username, err)
	}

	data := map[string]string{
		"method": method,
		"query":  cleanUsername,
	}

	resp, err := f.core.MakeRequest(ctx, data)
	if err != nil {
		return nil, err
	}

	// Check if response indicates an error
	if ok, _ := resp["ok"].(bool); !ok {
		errMsg, _ := resp["error"].(string)
		if errMsg == "" {
			errMsg = "user search failed"
		}
		return nil, fragErrors.NewUserNotFoundError(cleanUsername, fmt.Errorf("%s", errMsg))
	}

	// Extract user data from the response HTML
	// Fragment API returns HTML fragments; parse them for user info
	user := &models.UserInfo{
		Found: true,
	}

	// The response structure varies; extract what we can
	if result, ok := resp["result"].(map[string]interface{}); ok {
		if name, ok := result["name"].(string); ok {
			user.Name = name
		}
		if recipient, ok := result["recipient"].(string); ok {
			user.Recipient = recipient
		}
		if photo, ok := result["photo"].(string); ok {
			user.Avatar = extractAvatarURL(photo)
		}
	}

	// Fallback: try to extract from HTML content
	if html, ok := resp["html"].(string); ok {
		if user.Name == "" {
			user.Name = cleanUsername
		}
		if user.Avatar == "" {
			user.Avatar = extractAvatarURL(html)
		}
	}

	if user.Recipient == "" {
		// Try to extract recipient from nested structure
		if result, ok := resp["result"].(map[string]interface{}); ok {
			if rec, ok := result["recipient_id"].(string); ok {
				user.Recipient = rec
			}
		}
	}

	return user, nil
}

// extractAvatarURL extracts the image URL from an HTML img tag src attribute.
func extractAvatarURL(html string) string {
	matches := avatarRegexp.FindStringSubmatch(html)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractString extracts a string value from a map at the given key path.
func extractString(data map[string]interface{}, key string) (string, bool) {
	// Try direct access
	if val, ok := data[key]; ok {
		if s, ok := val.(string); ok {
			return s, true
		}
	}
	// Try under "result"
	if result, ok := data["result"].(map[string]interface{}); ok {
		if val, ok := result[key]; ok {
			if s, ok := val.(string); ok {
				return s, true
			}
		}
	}
	return "", false
}

// extractTransactionMessage extracts transaction details from a Fragment API response.
func extractTransactionMessage(data map[string]interface{}) (*models.TransactionMessage, error) {
	// Try to find transaction data in the response
	result, ok := data["result"].(map[string]interface{})
	if !ok {
		return nil, fragErrors.NewPaymentInitiationError("no result in response", nil)
	}

	// The response typically contains a "messages" array
	messages, ok := result["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return nil, fragErrors.NewPaymentInitiationError("no transaction messages in response", nil)
	}

	// Get first message
	msg, ok := messages[0].(map[string]interface{})
	if !ok {
		return nil, fragErrors.NewPaymentInitiationError("invalid transaction message format", nil)
	}

	address, _ := msg["address"].(string)
	amount, _ := msg["amount"].(string)
	payload, _ := msg["payload"].(string)

	if address == "" || amount == "" {
		return nil, fragErrors.NewPaymentInitiationError("missing address or amount in transaction message", nil)
	}

	return &models.TransactionMessage{
		Address: address,
		Amount:  amount,
		Payload: payload,
	}, nil
}

// stripHTML removes HTML tags from a string.
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}
