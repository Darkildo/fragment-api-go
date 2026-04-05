package fragment

import (
	"context"
	"errors"
	"fmt"
)

// GetRecipientStars looks up a Telegram user for a Stars transfer.
// The username may include a leading '@'.
func (c *Client) GetRecipientStars(ctx context.Context, username string) (*UserInfo, error) {
	return c.checkUser(ctx, username, "searchStarsRecipient", map[string]string{"quantity": ""})
}

// GetRecipientPremium looks up a Telegram user for a Premium gift.
// Returns an error if the user already has Premium or cannot receive a gift.
func (c *Client) GetRecipientPremium(ctx context.Context, username string) (*UserInfo, error) {
	return c.checkUser(ctx, username, "searchPremiumGiftRecipient", nil)
}

// GetRecipientTON looks up a Telegram user or channel for a TON Ads top-up.
func (c *Client) GetRecipientTON(ctx context.Context, username string) (*UserInfo, error) {
	return c.checkUser(ctx, username, "searchAdsTopupRecipient", nil)
}

// checkUser validates a username and queries the Fragment API.
//
// Fragment API response format:
//
//	Success: {"found": {"name": "...", "recipient": "...", "photo": "<img ...>"}}
//	Error:   {"error": "No Telegram users found."}
func (c *Client) checkUser(ctx context.Context, username, method string, extra map[string]string) (*UserInfo, error) {
	clean, err := validateUsername(username)
	if err != nil {
		return nil, newUserNotFoundError(username, err)
	}

	data := map[string]string{
		"method": method,
		"query":  clean,
	}
	for k, v := range extra {
		data[k] = v
	}

	resp, err := c.core.makeRequest(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("check user %q: %w", clean, err)
	}

	// Check for error response.
	if errVal, ok := resp["error"]; ok {
		var msg string
		switch v := errVal.(type) {
		case string:
			msg = v
		case map[string]interface{}:
			msg, _ = v["error"].(string)
		}
		if msg == "" {
			msg = "user search failed"
		}

		if msg == "Session expired" || msg == "AUTH_SESSION_EXPIRED" {
			return nil, newAuthenticationError(msg, nil)
		}

		return nil, newUserNotFoundError(clean, errors.New(msg))
	}

	// Extract from "found" top-level key.
	found, ok := resp["found"].(map[string]interface{})
	if !ok || found == nil {
		return nil, newUserNotFoundError(clean, errors.New("no 'found' data in response"))
	}

	user := &UserInfo{Found: true}

	if v, ok := found["name"].(string); ok {
		user.Name = v
	} else {
		user.Name = clean
	}

	if v, ok := found["recipient"].(string); ok {
		user.Recipient = v
	}

	if v, ok := found["photo"].(string); ok {
		user.Avatar = extractAvatarURL(v)
	}

	if user.Recipient == "" {
		return nil, newUserNotFoundError(clean, errors.New("recipient field is empty in response"))
	}

	c.log.Debug("user found",
		"username", clean,
		"name", user.Name,
		"recipient", user.Recipient,
	)

	return user, nil
}
