package xapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Client provides access to Twitter's API with automatic transaction ID generation
// This is the unified, production-ready client implementation
type Client struct {
	config *ProductionConfig
	
	// Core components
	http        *http.Client
	rateLimiter *rate.Limiter
	txnGen      *TransactionGenerator
	
	// Authentication
	guestToken  string
	guestID     string
	
	// XPFF header generation
	xpffGen *XPFFGenerator
	
	// Request state management
	mu           sync.RWMutex
	metrics      *ClientMetrics
	lastSuccess  time.Time
	errorStreak  int
	totalRequests int64
	
	// Production features
	retryEnabled bool
	debugEnabled bool
}

// ClientMetrics tracks performance metrics for monitoring
type ClientMetrics struct {
	TotalRequests     int64     `json:"total_requests"`
	SuccessfulRequests int64    `json:"successful_requests"`
	FailedRequests    int64     `json:"failed_requests"`
	RetryAttempts     int64     `json:"retry_attempts"`
	CacheHits         int64     `json:"cache_hits"`
	CacheMisses       int64     `json:"cache_misses"`
	AverageLatency    float64   `json:"average_latency_ms"`
	LastSuccessTime   time.Time `json:"last_success_time"`
	UptimeStart       time.Time `json:"uptime_start"`
}

// New creates a Twitter API client with production settings
func New() (*Client, error) {
	return NewClient(nil)
}

// NewClient creates a production-ready Twitter API client
func NewClient(config *ProductionConfig) (*Client, error) {
	if config == nil {
		config = DefaultProductionConfig()
	}
	
	// Initialize transaction generator with production config
	txnGen, err := NewTransactionGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize transaction generator: %w", err)
	}
	
	// Generate authentication tokens
	guestToken := generateGuestToken()
	guestID := generateGuestID()
	
	// Initialize XPFF generator
	xpffGen := NewXPFFGenerator()
	
	client := &Client{
		config: config,
		http: &http.Client{
			Timeout: config.RequestTimeout,
		},
		rateLimiter: rate.NewLimiter(rate.Limit(config.RateLimitRequests), 1),
		txnGen:      txnGen,
		guestToken:  guestToken,
		guestID:     guestID,
		xpffGen:     xpffGen,
		metrics: &ClientMetrics{
			UptimeStart: time.Now(),
		},
		retryEnabled: config.EnableAutoRetry,
		debugEnabled: config.EnableDebugLogging,
	}
	
	return client, nil
}

// NewDevelopmentClient creates a client optimized for development
func NewDevelopmentClient() (*Client, error) {
	return NewClient(DevelopmentConfig())
}

// Debug enables or disables debug logging at runtime
func (c *Client) Debug(enabled bool) {
	c.debugEnabled = enabled
}

// SetDebugMode enables or disables debug logging at runtime
func (c *Client) SetDebugMode(enabled bool) {
	c.debugEnabled = enabled
}

// User fetches a user's profile with automatic retry and intelligent caching
func (c *Client) User(ctx context.Context, username string) (*User, error) {
	return c.executeWithRetry(ctx, func(ctx context.Context) (*User, error) {
		return c.fetchUser(ctx, username)
	})
}

// executeWithRetry implements the retry logic with exponential backoff
func (c *Client) executeWithRetry(ctx context.Context, operation func(context.Context) (*User, error)) (*User, error) {
	c.mu.Lock()
	c.totalRequests++
	requestID := c.totalRequests
	c.mu.Unlock()
	
	start := time.Now()
	defer func() {
		c.updateLatencyMetrics(time.Since(start))
	}()
	
	var lastError error
	maxAttempts := 1
	if c.retryEnabled {
		maxAttempts = c.config.MaxRetryAttempts + 1 // +1 for initial attempt
	}
	
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if c.debugEnabled {
			fmt.Printf("🔄 Request #%d, attempt %d/%d\n", requestID, attempt, maxAttempts)
		}
		
		// Execute the operation
		result, err := operation(ctx)
		
		if err == nil {
			// Success - update metrics and reset error streak
			c.recordSuccess()
			if c.debugEnabled && attempt > 1 {
				fmt.Printf("✅ Request #%d succeeded after %d attempts\n", requestID, attempt)
			}
			return result, nil
		}
		
		// Record the error
		lastError = err
		c.recordError()
		
		// Don't retry if this is the last attempt or if context is cancelled
		if attempt >= maxAttempts || ctx.Err() != nil {
			break
		}
		
		// Calculate backoff duration with exponential backoff
		backoffDuration := time.Duration(float64(c.config.RetryBackoffBase) * 
			math.Pow(c.config.RetryBackoffMultiplier, float64(attempt-1)))
		
		if c.debugEnabled {
			fmt.Printf("⚠️ Request #%d failed (attempt %d): %v, retrying in %v...\n", 
				requestID, attempt, err, backoffDuration)
		}
		
		// Wait before retry (with context cancellation support)
		select {
		case <-time.After(backoffDuration):
			// Check if we should refresh data due to error streak
			if c.shouldRefreshDueToErrors() {
				if c.debugEnabled {
					fmt.Printf("🔄 Refreshing data due to error streak (%d errors)\n", c.getErrorStreak())
				}
				c.txnGen.ForceRefreshTransactionID()
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	// All attempts failed
	if c.debugEnabled {
		fmt.Printf("❌ Request #%d failed after %d attempts: %v\n", requestID, maxAttempts, lastError)
	}
	
	return nil, fmt.Errorf("request failed after %d attempts: %w", maxAttempts, lastError)
}

// fetchUser performs the actual user fetch operation
func (c *Client) fetchUser(ctx context.Context, username string) (*User, error) {
	username = strings.TrimPrefix(username, "@")

	features := `{"profile_label_improvements_pcf_label_in_post_enabled":false,"hidden_profile_subscriptions_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true,"subscriptions_verification_info_is_identity_verified_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":false,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"verified_phone_label_enabled":false,"payments_enabled":false,"subscriptions_feature_can_gift_premium":false,"rweb_xchat_enabled":false,"rweb_tipjar_consumption_enabled":true,"creator_subscriptions_tweet_preview_api_enabled":true,"freedom_of_speech_not_reach_fetch_enabled":true,"responsive_web_twitter_article_tweet_consumption_enabled":false,"articles_preview_enabled":false,"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled":true,"responsive_web_edit_tweet_api_enabled":true,"graphql_is_translatable_rweb_tweet_is_translatable_enabled":true,"communities_web_enable_tweet_community_results_fetch":true,"responsive_web_grok_analyze_post_followups_enabled":false,"responsive_web_grok_share_attachment_enabled":false,"c9s_tweet_anatomy_moderator_badge_enabled":true,"longform_notetweets_consumption_enabled":true,"rweb_video_screen_enabled":false,"longform_notetweets_inline_media_enabled":true,"responsive_web_enhance_cards_enabled":false,"responsive_web_grok_show_grok_translated_post":false,"longform_notetweets_rich_text_read_enabled":true,"responsive_web_jetfuel_frame":false,"responsive_web_grok_analyze_button_fetch_trends_enabled":false,"creator_subscriptions_quote_tweet_preview_enabled":false,"responsive_web_grok_analysis_button_from_backend":false,"view_counts_everywhere_api_enabled":true,"responsive_web_grok_image_annotation_enabled":false,"responsive_web_grok_imagine_annotation_enabled":false,"tweet_awards_web_tipping_enabled":false,"premium_content_api_read_enabled":false,"standardized_nudges_misinfo":true,"responsive_web_grok_community_note_auto_translation_is_enabled":false}`

	resp, err := c.request(ctx, "GET", "ck5KkZ8t5cOmoLssopN99Q/UserByScreenName", map[string]string{
		"variables": fmt.Sprintf(`{"screen_name":"%s","withGrokTranslatedBio":false}`, username),
		"features":  features,
	})
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			User struct {
				Result *UserData `json:"result"`
			} `json:"user"`
		} `json:"data"`
	}

	if c.debugEnabled {
		fmt.Printf("🔍 Raw User API response: %s\n", string(resp))
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Data.User.Result == nil {
		return nil, fmt.Errorf("user not found")
	}

	userResult := result.Data.User.Result

	// Create user from legacy data (which has most fields)
	var user *User
	if userResult.Legacy != nil {
		user = userResult.Legacy
	} else {
		user = &User{}
	}

	// Set IDs
	user.ID = userResult.RestID
	user.RestID = userResult.RestID

	// Override with core data if available (name, screen_name are here)
	if userResult.Core != nil {
		user.Name = userResult.Core.Name
		user.ScreenName = userResult.Core.ScreenName
		if userResult.Core.CreatedAt != "" {
			// Parse created_at if needed
			if createdAt, err := time.Parse("Mon Jan 02 15:04:05 -0700 2006", userResult.Core.CreatedAt); err == nil {
				user.CreatedAt = createdAt
			}
		}
	}

	if c.debugEnabled {
		fmt.Printf("🔍 Final user data: Name=%s, ScreenName=%s, RestID=%s, Followers=%d\n",
			user.Name, user.ScreenName, user.RestID, user.FollowersCount)
	}

	return user, nil
}

// request makes an authenticated API request with smart transaction ID management
func (c *Client) request(ctx context.Context, method, endpoint string, params map[string]string) ([]byte, error) {
	// Rate limiting
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}

	// Build URL
	u, err := url.Parse("https://api.x.com/graphql/" + endpoint)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set headers with smart transaction ID
	if err := c.setHeaders(req, method, u.Path); err != nil {
		return nil, fmt.Errorf("failed to set headers: %w", err)
	}

	if c.debugEnabled {
		fmt.Printf("→ %s %s\n", method, u.String())
	}

	// Make request
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if c.debugEnabled {
		fmt.Printf("← %d %s\n", resp.StatusCode, string(body)[:min(200, len(body))])
	}

	// Handle different response codes
	switch resp.StatusCode {
	case 200:
		// Success
		return body, nil

	case 401, 403:
		return nil, fmt.Errorf("authentication error: %d %s", resp.StatusCode, string(body))

	case 429:
		return nil, fmt.Errorf("rate limited: %d %s", resp.StatusCode, string(body))

	default:
		return nil, fmt.Errorf("API error: %d %s", resp.StatusCode, string(body))
	}
}

// setHeaders sets required headers for Twitter API
func (c *Client) setHeaders(req *http.Request, method, path string) error {
	// Generate transaction ID
	txnID, err := c.txnGen.Generate(method, path)
	if err != nil {
		return fmt.Errorf("failed to generate transaction ID: %w", err)
	}

	// Generate XPFF header
	userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:141.0) Gecko/20100101 Firefox/141.0"
	xpffHeader, err := c.xpffGen.GenerateXPFF(c.guestID, userAgent)
	if err != nil {
		if c.debugEnabled {
			fmt.Printf("⚠️ Failed to generate XPFF header: %v\n", err)
		}
		// Continue without XPFF header
	}

	if c.debugEnabled {
		fmt.Printf("Using transaction ID: %s...\n", txnID[:min(20, len(txnID))])
		if xpffHeader != "" {
			fmt.Printf("Generated XPFF header: %s...\n", xpffHeader[:min(50, len(xpffHeader))])
		}
	}

	req.Header.Set("Authorization", "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA")
	req.Header.Set("X-Client-Transaction-Id", txnID)
	if xpffHeader != "" {
		req.Header.Set("X-Xp-Forwarded-For", xpffHeader)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://x.com")
	req.Header.Set("Referer", "https://x.com/")
	req.Header.Set("X-Twitter-Active-User", "yes")
	req.Header.Set("X-Twitter-Client-Language", "en")

	return nil
}

// generateGuestToken creates a guest token similar to Twitter's format
func generateGuestToken() string {
	// Twitter guest tokens are typically 19-digit numbers starting with 19
	now := time.Now().Unix()
	randomPart := rand.Int63n(1000000000) // 9 digits

	// Combine timestamp and random part to create a 19-digit token
	token := strconv.FormatInt(now*1000000000+randomPart, 10)

	// Ensure it's 19 digits by padding or truncating if needed
	if len(token) > 19 {
		token = token[:19]
	} else if len(token) < 19 {
		token = "195" + token // Prefix with "195" to make it 19 digits
	}

	return token
}

// generateGuestID creates a guest ID in Twitter's format (v1%3A + timestamp)
func generateGuestID() string {
	now := time.Now().UnixMilli()
	return fmt.Sprintf("v1%%3A%d", now)
}

// recordSuccess updates success metrics and resets error streak
func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.metrics.SuccessfulRequests++
	c.metrics.LastSuccessTime = time.Now()
	c.lastSuccess = time.Now()
	c.errorStreak = 0
}

// recordError updates error metrics and increments error streak
func (c *Client) recordError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.metrics.FailedRequests++
	c.errorStreak++
}

// shouldRefreshDueToErrors checks if we should force refresh due to consecutive errors
func (c *Client) shouldRefreshDueToErrors() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return c.errorStreak >= c.config.ErrorThresholdForRefresh
}

// getErrorStreak returns current error streak (thread-safe)
func (c *Client) getErrorStreak() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.errorStreak
}

// updateLatencyMetrics updates average latency calculation
func (c *Client) updateLatencyMetrics(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	latencyMs := float64(duration.Nanoseconds()) / 1e6
	totalRequests := float64(c.metrics.SuccessfulRequests + c.metrics.FailedRequests)
	
	// Calculate rolling average
	if totalRequests == 1 {
		c.metrics.AverageLatency = latencyMs
	} else {
		// Exponential moving average with alpha = 0.1
		alpha := 0.1
		c.metrics.AverageLatency = (1-alpha)*c.metrics.AverageLatency + alpha*latencyMs
	}
}

// GetMetrics returns current client metrics for monitoring
func (c *Client) GetMetrics() *ClientMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	// Return a copy to prevent race conditions
	metrics := *c.metrics
	return &metrics
}

// GetSuccessRate returns the current success rate as a percentage
func (c *Client) GetSuccessRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	total := c.metrics.SuccessfulRequests + c.metrics.FailedRequests
	if total == 0 {
		return 0.0
	}
	
	return float64(c.metrics.SuccessfulRequests) / float64(total) * 100.0
}

// GetUptime returns how long the client has been running
func (c *Client) GetUptime() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	return time.Since(c.metrics.UptimeStart)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}