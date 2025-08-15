package xapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// TweetOption configures tweet fetching
type TweetOption func(*tweetOptions)

type tweetOptions struct {
	count        int
	cursor       string
	returnCursor bool
}

// WithCount sets the number of tweets to fetch
func WithCount(count int) TweetOption {
	return func(opts *tweetOptions) {
		opts.count = count
	}
}

// WithCursor sets the pagination cursor
func WithCursor(cursor string) TweetOption {
	return func(opts *tweetOptions) {
		opts.cursor = cursor
	}
}

// WithPagination enables cursor return for pagination
func WithPagination() TweetOption {
	return func(opts *tweetOptions) {
		opts.returnCursor = true
	}
}

// Tweets fetches a user's recent tweets (default: 20 tweets)
func (c *Client) Tweets(ctx context.Context, username string, options ...TweetOption) ([]*Tweet, error) {
	page, err := c.TweetsPage(ctx, username, options...)
	if err != nil {
		return nil, err
	}
	return page.Tweets, nil
}

// TweetsPage fetches tweets with pagination info (default: 20 tweets)
func (c *Client) TweetsPage(ctx context.Context, username string, options ...TweetOption) (*TweetPage, error) {
	return c.tweetsPage(ctx, username, append(options, WithPagination())...)
}

func (c *Client) tweetsPage(ctx context.Context, username string, options ...TweetOption) (*TweetPage, error) {
	// Apply options with defaults
	opts := &tweetOptions{
		count: 20, // Default count
	}
	for _, opt := range options {
		opt(opts)
	}

	// Get user first to get their ID
	user, err := c.User(ctx, username)
	if err != nil {
		return nil, err
	}

	// Build variables with optional cursor
	variables := fmt.Sprintf(`{"userId":"%s","count":%d,"includePromotedContent":false,"withQuickPromoteEligibilityTweetFields":false,"withVoice":false`, user.ID, opts.count)
	if opts.cursor != "" {
		variables += fmt.Sprintf(`,"cursor":"%s"`, opts.cursor)
	}
	variables += "}"

	features := `{"profile_label_improvements_pcf_label_in_post_enabled":false,"hidden_profile_subscriptions_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true,"subscriptions_verification_info_is_identity_verified_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":false,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"verified_phone_label_enabled":false,"payments_enabled":false,"subscriptions_feature_can_gift_premium":false,"rweb_xchat_enabled":false,"rweb_tipjar_consumption_enabled":true,"creator_subscriptions_tweet_preview_api_enabled":true,"freedom_of_speech_not_reach_fetch_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":false,"articles_preview_enabled":false,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"communities_web_enable_tweet_community_results_fetch":true,"responsive_web_grok_analyze_post_followups_enabled":false,"responsive_web_grok_share_attachment_enabled":false,"c9s_tweet_anatomy_moderator_badge_enabled":true,"longform_notetweets_consumption_enabled":true,"rweb_video_screen_enabled":false,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"longform_notetweets_rich_text_read_enabled":true,"responsive_web_jetfuel_frame":false,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"responsive_web_grok_analysis_button_from_backend":false,"view_counts_everywhere_api_enabled":true,"responsive_web_grok_image_annotation_enabled":false,"responsive_web_grok_imagine_annotation_enabled":false,"tweet_awards_web_tipping_enabled":false,"premium_content_api_read_enabled":false,"standardized_nudges_misinfo":true,"responsive_web_grok_community_note_auto_translation_is_enabled":false}`

	resp, err := c.request(ctx, "GET", "E8Wq-_jFSaU7hxVcuOPR9g/UserTweets", map[string]string{
		"variables": variables,
		"features":  features,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			User struct {
				Result struct {
					Timeline Timeline `json:"timeline"`
				} `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract tweets
	tweets := c.extractTweets(result.Data.User.Result.Timeline)

	// Extract cursors if requested
	var nextCursor, prevCursor *Cursor
	if opts.returnCursor {
		nextCursor, prevCursor = c.extractCursors(result.Data.User.Result.Timeline)
	}

	return &TweetPage{
		Tweets:     tweets,
		NextCursor: nextCursor,
		PrevCursor: prevCursor,
		HasMore:    nextCursor != nil,
	}, nil
}

// Tweet fetches a single tweet by ID
func (c *Client) Tweet(ctx context.Context, tweetID string) (*Tweet, error) {
	resp, err := c.request(ctx, "GET", "qxWQxcMLiTPcavz9Qy5hwQ/TweetResultByRestId", map[string]string{
		"variables": fmt.Sprintf(`{"tweetId":"%s","withCommunity":false,"includePromotedContent":false,"withVoice":false}`, tweetID),
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			TweetResult struct {
				Result *TweetData `json:"result"`
			} `json:"tweetResult"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Data.TweetResult.Result == nil || result.Data.TweetResult.Result.Legacy == nil {
		return nil, fmt.Errorf("tweet not found")
	}

	tweet := result.Data.TweetResult.Result.Legacy
	tweet.ID = result.Data.TweetResult.Result.RestID
	return tweet, nil
}

// Broadcast fetches live broadcast information
func (c *Client) Broadcast(ctx context.Context, broadcastID string) (*Broadcast, error) {
	resp, err := c.request(ctx, "GET", "BGhq0o90P-tPie4pyhqlVA/BroadcastQuery", map[string]string{
		"variables": fmt.Sprintf(`{"id":"%s"}`, broadcastID),
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Broadcast *Broadcast `json:"broadcast"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Data.Broadcast == nil {
		return nil, fmt.Errorf("broadcast not found")
	}

	return result.Data.Broadcast, nil
}

// Highlights fetches a user's highlighted tweets
func (c *Client) Highlights(ctx context.Context, userID string, count int) ([]*Tweet, error) {
	if count == 0 {
		count = 20
	}

	// Complete features parameter from working HAR file
	features := `{"rweb_video_screen_enabled":false,"payments_enabled":false,"rweb_xchat_enabled":false,"profile_label_improvements_pcf_label_in_post_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"premium_content_api_read_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"responsive_web_grok_analyze_post_followups_enabled":false,"responsive_web_jetfuel_frame":true,"responsive_web_grok_share_attachment_enabled":true,"articles_preview_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"responsive_web_grok_analysis_button_from_backend":true,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_grok_image_annotation_enabled":true,"responsive_web_grok_imagine_annotation_enabled":true,"responsive_web_grok_community_note_auto_translation_is_enabled":false,"responsive_web_enhance_cards_enabled":false}`

	resp, err := c.request(ctx, "GET", "gmHw9geMTncZ7jeLLUUNOw/UserHighlightsTweets", map[string]string{
		"variables": fmt.Sprintf(`{"userId":"%s","count":%d,"includePromotedContent":true,"withVoice":true}`, userID, count),
		"features":  features,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			User struct {
				Result struct {
					Timeline Timeline `json:"timeline"`
				} `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.extractTweets(result.Data.User.Result.Timeline), nil
}

// Following fetches users that a user follows
func (c *Client) Following(ctx context.Context, userID string, count int) ([]*User, error) {
	if count == 0 {
		count = 20
	}

	// Features parameter from working HAR file
	features := `{"rweb_video_screen_enabled":false,"payments_enabled":false,"rweb_xchat_enabled":false,"profile_label_improvements_pcf_label_in_post_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_timeline_navigation_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"premium_content_api_read_enabled":false,"communities_web_enable_tweet_community_results_fetch":true,"c9s_tweet_anatomy_moderator_badge_enabled":true,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"responsive_web_grok_analyze_post_followups_enabled":true,"responsive_web_jetfuel_frame":true,"responsive_web_grok_share_attachment_enabled":true,"articles_preview_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"view_counts_everywhere_api_enabled":true,"longform_notetweets_consumption_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":true,"tweet_awards_web_tipping_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"responsive_web_grok_analysis_button_from_backend":true,"creator_subscriptions_quote_tweet_preview_enabled":false,"freedom_of_speech_not_reach_fetch_enabled":true,"standardized_nudges_misinfo":true,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"longform_notetweets_rich_text_read_enabled":true,"longform_notetweets_inline_media_enabled":true,"responsive_web_grok_image_annotation_enabled":true,"responsive_web_grok_imagine_annotation_enabled":true,"responsive_web_grok_community_note_auto_translation_is_enabled":false,"responsive_web_enhance_cards_enabled":false}`

	resp, err := c.request(ctx, "GET", "SaWqzw0TFAWMx1nXWjXoaQ/Following", map[string]string{
		"variables": fmt.Sprintf(`{"userId":"%s","count":%d,"includePromotedContent":false,"withGrokTranslatedBio":false}`, userID, count),
		"features":  features,
	})
	if err != nil {
		return nil, err
	}

	return c.extractUsers(resp)
}

// Followers fetches a user's followers
func (c *Client) Followers(ctx context.Context, userID string, count int) ([]*User, error) {
	if count == 0 {
		count = 20
	}

	resp, err := c.request(ctx, "GET", "i6PPdIMm1MO7CpAqjau7sw/Followers", map[string]string{
		"variables": fmt.Sprintf(`{"userId":"%s","count":%d,"includePromotedContent":false,"withGrokTranslatedBio":false}`, userID, count),
	})
	if err != nil {
		return nil, err
	}

	return c.extractUsers(resp)
}

// BlueVerified fetches blue verified followers
func (c *Client) BlueVerified(ctx context.Context, userID string, count int) ([]*User, error) {
	if count == 0 {
		count = 20
	}

	resp, err := c.request(ctx, "GET", "fxEl9kp1Tgolqkq8_Lo3sg/BlueVerifiedFollowers", map[string]string{
		"variables": fmt.Sprintf(`{"userId":"%s","count":%d,"includePromotedContent":false,"withGrokTranslatedBio":false}`, userID, count),
	})
	if err != nil {
		return nil, err
	}

	return c.extractUsers(resp)
}

// UserBusiness fetches business profile team timeline
func (c *Client) UserBusiness(ctx context.Context, userID string, teamName string, count int) ([]*Tweet, error) {
	if count == 0 {
		count = 20
	}
	if teamName == "" {
		teamName = "NotAssigned"
	}

	resp, err := c.request(ctx, "GET", "zUBrgfL8uXdM3VR9TqHzNQ/UserBusinessProfileTeamTimeline", map[string]string{
		"variables": fmt.Sprintf(`{"userId":"%s","count":%d,"teamName":"%s","includePromotedContent":false,"withClientEventToken":false,"withVoice":true}`, userID, count, teamName),
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			User struct {
				Result struct {
					Timeline Timeline `json:"timeline"`
				} `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return c.extractTweets(result.Data.User.Result.Timeline), nil
}

// UsersByIDs fetches multiple users by their IDs in one call
func (c *Client) UsersByIDs(ctx context.Context, userIDs []string) ([]*User, error) {
	if len(userIDs) == 0 {
		return nil, fmt.Errorf("no user IDs provided")
	}

	// Build JSON array of user IDs
	idsJSON := "["
	for i, id := range userIDs {
		if i > 0 {
			idsJSON += ","
		}
		idsJSON += fmt.Sprintf(`"%s"`, id)
	}
	idsJSON += "]"

	// Features parameter from working HAR file (simpler version for UsersByRestIds)
	features := `{"payments_enabled":false,"rweb_xchat_enabled":false,"profile_label_improvements_pcf_label_in_post_enabled":true,"rweb_tipjar_consumption_enabled":true,"verified_phone_label_enabled":false,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`

	resp, err := c.request(ctx, "GET", "1hjT2eXW1Zcw-2xk8EbvoA/UsersByRestIds", map[string]string{
		"variables": fmt.Sprintf(`{"userIds":%s}`, idsJSON),
		"features":  features,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Users []*struct {
				Result *UserData `json:"result"`
			} `json:"users"`
		} `json:"data"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var users []*User
	for _, userWrapper := range result.Data.Users {
		if userWrapper.Result != nil && userWrapper.Result.Legacy != nil {
			user := userWrapper.Result.Legacy
			user.ID = userWrapper.Result.RestID
			users = append(users, user)
		}
	}

	return users, nil
}

// Profile fetches a user and their recent tweets in one call
func (c *Client) Profile(ctx context.Context, username string, tweetCount int) (*Profile, error) {
	if tweetCount == 0 {
		tweetCount = 20
	}

	user, err := c.User(ctx, username)
	if err != nil {
		return nil, err
	}

	tweets, err := c.Tweets(ctx, username, WithCount(tweetCount))
	if err != nil {
		return nil, err
	}

	return &Profile{
		User:   user,
		Tweets: tweets,
		Stats:  calculateStats(user, tweets),
	}, nil
}

// Helper methods for data extraction

// extractTweets extracts tweet data from timeline response
func (c *Client) extractTweets(timeline Timeline) []*Tweet {
	var tweets []*Tweet

	for _, instruction := range timeline.Instructions {
		if instruction.Type == "TimelineAddEntries" {
			for _, entry := range instruction.Entries {
				if entry.Content.ItemContent != nil &&
					entry.Content.ItemContent.TweetResults != nil &&
					entry.Content.ItemContent.TweetResults.Result != nil &&
					entry.Content.ItemContent.TweetResults.Result.Legacy != nil {

					tweet := entry.Content.ItemContent.TweetResults.Result.Legacy
					tweet.ID = entry.Content.ItemContent.TweetResults.Result.RestID
					tweets = append(tweets, tweet)
				}
			}
		}
	}

	return tweets
}

// extractUsers extracts user data from following/followers response
func (c *Client) extractUsers(respData []byte) ([]*User, error) {
	var result struct {
		Data struct {
			User struct {
				Result struct {
					Timeline Timeline `json:"timeline"`
				} `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var users []*User
	for _, instruction := range result.Data.User.Result.Timeline.Instructions {
		if instruction.Type == "TimelineAddEntries" {
			for _, entry := range instruction.Entries {
				if entry.Content.ItemContent != nil &&
					entry.Content.ItemContent.UserResults != nil &&
					entry.Content.ItemContent.UserResults.Result != nil &&
					entry.Content.ItemContent.UserResults.Result.Legacy != nil {

					user := entry.Content.ItemContent.UserResults.Result.Legacy
					user.ID = entry.Content.ItemContent.UserResults.Result.RestID
					users = append(users, user)
				}
			}
		}
	}

	return users, nil
}

// extractCursors extracts pagination cursors from timeline
func (c *Client) extractCursors(timeline Timeline) (*Cursor, *Cursor) {
	var nextCursor, prevCursor *Cursor

	for _, instruction := range timeline.Instructions {
		if instruction.Type == "TimelineAddEntries" {
			for _, entry := range instruction.Entries {
				if strings.HasPrefix(entry.EntryID, "cursor-bottom-") {
					if entry.Content.CursorType == "Bottom" && entry.Content.Value != "" {
						nextCursor = &Cursor{
							Value:      entry.Content.Value,
							CursorType: "Bottom",
						}
					}
				} else if strings.HasPrefix(entry.EntryID, "cursor-top-") {
					if entry.Content.CursorType == "Top" && entry.Content.Value != "" {
						prevCursor = &Cursor{
							Value:      entry.Content.Value,
							CursorType: "Top",
						}
					}
				}
			}
		}
	}

	return nextCursor, prevCursor
}

// calculateStats computes profile statistics
func calculateStats(user *User, tweets []*Tweet) *Stats {
	if len(tweets) == 0 {
		return &Stats{}
	}

	totalEngagement := 0
	maxEngagement := 0
	var topTweet *Tweet

	for _, tweet := range tweets {
		engagement := tweet.FavoriteCount + tweet.RetweetCount + tweet.ReplyCount
		totalEngagement += engagement
		if engagement > maxEngagement {
			maxEngagement = engagement
			topTweet = tweet
		}
	}

	return &Stats{
		TotalEngagement: totalEngagement,
		AvgEngagement:   float64(totalEngagement) / float64(len(tweets)),
		TopTweet:        topTweet,
	}
}