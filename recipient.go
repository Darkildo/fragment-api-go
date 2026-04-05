package fragment

import (
	"context"
	"fmt"
)

// GetRecipientStars looks up a Telegram user for a Stars transfer.
// The username may include a leading '@'.
func (c *Client) GetRecipientStars(ctx context.Context, username string) (*UserInfo, error) {
	return c.checkUser(ctx, username, "searchStarsRecipient")
}

// GetRecipientPremium looks up a Telegram user for a Premium gift.
// Returns an error if the user already has Premium or cannot receive a gift.
func (c *Client) GetRecipientPremium(ctx context.Context, username string) (*UserInfo, error) {
	return c.checkUser(ctx, username, "searchPremiumGiftRecipient")
}

// GetRecipientTON looks up a Telegram user or channel for a TON Ads top-up.
func (c *Client) GetRecipientTON(ctx context.Context, username string) (*UserInfo, error) {
	return c.checkUser(ctx, username, "searchAdsTopupRecipient")
}

// checkUser validates a username and queries the Fragment API.
func (c *Client) checkUser(ctx context.Context, username, method string) (*UserInfo, error) {
	clean, err := validateUsername(username)
	if err != nil {
		return nil, newUserNotFoundError(username, err)
	}

	resp, err := c.core.makeRequest(ctx, map[string]string{
		"method": method,
		"query":  clean,
	})
	if err != nil {
		return nil, err
	}

	if ok, _ := resp["ok"].(bool); !ok {
		msg, _ := resp["error"].(string)
		if msg == "" {
			msg = "user search failed"
		}
		return nil, newUserNotFoundError(clean, fmt.Errorf("%s", msg))
	}

	user := &UserInfo{Found: true}

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if v, ok := result["name"].(string); ok {
			user.Name = v
		}
		if v, ok := result["recipient"].(string); ok {
			user.Recipient = v
		}
		if v, ok := result["recipient_id"].(string); ok && user.Recipient == "" {
			user.Recipient = v
		}
		if v, ok := result["photo"].(string); ok {
			user.Avatar = extractAvatarURL(v)
		}
	}

	if html, ok := resp["html"].(string); ok {
		if user.Name == "" {
			user.Name = clean
		}
		if user.Avatar == "" {
			user.Avatar = extractAvatarURL(html)
		}
	}

	return user, nil
}
