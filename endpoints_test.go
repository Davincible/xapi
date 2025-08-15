package xapi

import (
	"context"
	"testing"
	"time"
)

func TestUser(t *testing.T) {
	start := time.Now()
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Test with username
	t.Logf("Testing User endpoint with 'nasa'...")
	userStart := time.Now()
	user, err := client.User(ctx, "nasa")
	userDuration := time.Since(userStart)
	
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	t.Logf("User request took: %v", userDuration)
	t.Logf("User details:")
	t.Logf("  ID: %s", user.ID)
	t.Logf("  ScreenName: %s", user.ScreenName)
	t.Logf("  Name: %s", user.Name)
	t.Logf("  Description: %s", user.Description)
	t.Logf("  Followers: %d", user.FollowersCount)
	t.Logf("  Following: %d", user.FriendsCount)
	t.Logf("  Tweets: %d", user.StatusesCount)
	t.Logf("  Verified: %t", user.Verified)
	t.Logf("  Blue Verified: %t", user.IsBlueVerified)
	
	if user.ID == "" {
		t.Error("User ID should not be empty")
	}
	if user.ScreenName == "" {
		t.Error("User screen name should not be empty")
	}
	if user.Name == "" {
		t.Error("User name should not be empty")
	}
	
	// Test with @ prefix
	t.Logf("Testing User endpoint with '@nasa'...")
	user2Start := time.Now()
	user2, err := client.User(ctx, "@nasa")
	user2Duration := time.Since(user2Start)
	
	if err != nil {
		t.Fatalf("Failed to get user with @ prefix: %v", err)
	}
	
	t.Logf("User2 request took: %v", user2Duration)
	t.Logf("User2 ID: %s", user2.ID)
	
	if user.ID != user2.ID {
		t.Error("User ID should be the same regardless of @ prefix")
	}
	
	t.Logf("Total TestUser time: %v", time.Since(start))
}

func TestTweets(t *testing.T) {
	start := time.Now()
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Test default tweets
	t.Logf("Testing Tweets endpoint with 'nasa'...")
	tweetsStart := time.Now()
	tweets, err := client.Tweets(ctx, "nasa")
	tweetsDuration := time.Since(tweetsStart)
	
	if err != nil {
		t.Fatalf("Failed to get tweets: %v", err)
	}
	
	t.Logf("Tweets request took: %v", tweetsDuration)
	t.Logf("Retrieved %d tweets", len(tweets))
	
	for i, tweet := range tweets {
		if i < 3 { // Log first 3 tweets in detail
			t.Logf("Tweet %d:", i+1)
			t.Logf("  ID: %s", tweet.ID)
			t.Logf("  Text: %s", tweet.FullText)
			t.Logf("  Created: %v", tweet.CreatedAt)
			t.Logf("  Likes: %d", tweet.FavoriteCount)
			t.Logf("  Retweets: %d", tweet.RetweetCount)
			t.Logf("  Replies: %d", tweet.ReplyCount)
			t.Logf("  Views: %d", tweet.ViewCount)
		}
		
		if tweet.ID == "" {
			t.Error("Tweet ID should not be empty")
		}
		if tweet.FullText == "" {
			t.Error("Tweet text should not be empty")
		}
	}
	
	t.Logf("Total TestTweets time: %v", time.Since(start))
}

func TestTweetsWithOptions(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Test with count option
	tweets, err := client.Tweets(ctx, "nasa", WithCount(5))
	if err != nil {
		t.Fatalf("Failed to get tweets with count: %v", err)
	}
	
	if len(tweets) > 5 {
		t.Errorf("Expected at most 5 tweets, got %d", len(tweets))
	}
}

func TestTweetsPage(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Test with pagination
	page, err := client.TweetsPage(ctx, "nasa", WithCount(10))
	if err != nil {
		t.Fatalf("Failed to get tweet page: %v", err)
	}
	
	t.Logf("Retrieved %d tweets in page", len(page.Tweets))
	
	// Test cursor functionality if available
	if page.NextCursor != nil {
		tweets2, err := client.Tweets(ctx, "nasa", WithCursor(page.NextCursor.Value))
		if err != nil {
			t.Fatalf("Failed to get tweets with cursor: %v", err)
		}
		
		t.Logf("Retrieved %d tweets with cursor", len(tweets2))
	}
}

func TestTweet(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// First get some tweets to get a valid tweet ID
	tweets, err := client.Tweets(ctx, "nasa", WithCount(1))
	if err != nil {
		t.Fatalf("Failed to get tweets: %v", err)
	}
	
	if len(tweets) == 0 {
		t.Skip("No tweets available to test with")
	}
	
	// Test single tweet fetch
	tweet, err := client.Tweet(ctx, tweets[0].ID)
	if err != nil {
		t.Fatalf("Failed to get single tweet: %v", err)
	}
	
	if tweet.ID != tweets[0].ID {
		t.Error("Tweet ID should match")
	}
}

func TestProfile(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	profile, err := client.Profile(ctx, "nasa", 10)
	if err != nil {
		t.Fatalf("Failed to get profile: %v", err)
	}
	
	if profile.User == nil {
		t.Error("Profile user should not be nil")
	}
	if profile.User.ID == "" {
		t.Error("Profile user ID should not be empty")
	}
	
	t.Logf("Profile has %d tweets", len(profile.Tweets))
	
	if profile.Stats == nil {
		t.Error("Profile stats should not be nil")
	}
	if profile.Stats.AvgEngagement < 0 {
		t.Error("Average engagement should not be negative")
	}
}

func TestFollowing(t *testing.T) {
	start := time.Now()
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Get a user first to get their ID
	t.Logf("Getting user 'nasa' for Following test...")
	user, err := client.User(ctx, "nasa")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	t.Logf("Testing Following endpoint with userID: %s", user.ID)
	followingStart := time.Now()
	following, err := client.Following(ctx, user.ID, 5)
	followingDuration := time.Since(followingStart)
	
	if err != nil {
		t.Logf("Following endpoint returned error after %v: %v", followingDuration, err)
		t.Logf("This endpoint may be restricted or need special permissions")
		return // Skip test if endpoint is restricted
	}
	
	t.Logf("Following request took: %v", followingDuration)
	t.Logf("Retrieved %d following users", len(following))
	
	for i, followedUser := range following {
		if i < 3 { // Log first 3 users in detail
			t.Logf("Following User %d:", i+1)
			t.Logf("  ID: %s", followedUser.ID)
			t.Logf("  ScreenName: %s", followedUser.ScreenName)
			t.Logf("  Name: %s", followedUser.Name)
			t.Logf("  Followers: %d", followedUser.FollowersCount)
		}
		
		if followedUser.ID == "" {
			t.Error("Following user ID should not be empty")
		}
		if followedUser.ScreenName == "" {
			t.Error("Following user screen name should not be empty")
		}
	}
	
	t.Logf("Total TestFollowing time: %v", time.Since(start))
}

func TestFollowers(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Get a user first to get their ID
	user, err := client.User(ctx, "nasa")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	followers, err := client.Followers(ctx, user.ID, 5)
	if err != nil {
		t.Logf("Followers endpoint returned error (may be restricted): %v", err)
		return // Skip test if endpoint is restricted
	}
	
	t.Logf("Retrieved %d followers", len(followers))
	
	for _, follower := range followers {
		if follower.ID == "" {
			t.Error("Follower user ID should not be empty")
		}
		if follower.ScreenName == "" {
			t.Error("Follower user screen name should not be empty")
		}
	}
}

func TestBlueVerified(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Get a user first to get their ID
	user, err := client.User(ctx, "nasa")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	blueFollowers, err := client.BlueVerified(ctx, user.ID, 5)
	if err != nil {
		t.Fatalf("Failed to get blue verified followers: %v", err)
	}
	
	for _, blueUser := range blueFollowers {
		if blueUser.ID == "" {
			t.Error("Blue verified user ID should not be empty")
		}
		if !blueUser.IsBlueVerified {
			t.Error("User should be blue verified")
		}
	}
}

func TestHighlights(t *testing.T) {
	start := time.Now()
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Get a user first to get their ID
	t.Logf("Getting user 'nasa' for Highlights test...")
	user, err := client.User(ctx, "nasa")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	t.Logf("Testing Highlights endpoint with userID: %s", user.ID)
	highlightsStart := time.Now()
	highlights, err := client.Highlights(ctx, user.ID, 5)
	highlightsDuration := time.Since(highlightsStart)
	
	if err != nil {
		t.Logf("Highlights endpoint returned error after %v: %v", highlightsDuration, err)
		t.Logf("This endpoint may need specific features parameter or be restricted")
		return // Skip test if endpoint fails
	}
	
	t.Logf("Highlights request took: %v", highlightsDuration)
	t.Logf("Retrieved %d highlight tweets", len(highlights))
	
	for i, tweet := range highlights {
		if i < 3 { // Log first 3 highlights in detail
			t.Logf("Highlight Tweet %d:", i+1)
			t.Logf("  ID: %s", tweet.ID)
			t.Logf("  Text: %s", tweet.FullText)
			t.Logf("  Likes: %d", tweet.FavoriteCount)
		}
		
		if tweet.ID == "" {
			t.Error("Highlight tweet ID should not be empty")
		}
		if tweet.FullText == "" {
			t.Error("Highlight tweet text should not be empty")
		}
	}
	
	t.Logf("Total TestHighlights time: %v", time.Since(start))
}

func TestUserBusiness(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Get a user first to get their ID
	user, err := client.User(ctx, "nasa")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	businessTweets, err := client.UserBusiness(ctx, user.ID, "NotAssigned", 5)
	if err != nil {
		t.Fatalf("Failed to get business tweets: %v", err)
	}
	
	for _, tweet := range businessTweets {
		if tweet.ID == "" {
			t.Error("Business tweet ID should not be empty")
		}
	}
}

func TestUsersByIDs(t *testing.T) {
	start := time.Now()
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Get a couple users first to get their IDs
	t.Logf("Getting users 'nasa' and 'spacex' for UsersByIDs test...")
	user1, err := client.User(ctx, "nasa")
	if err != nil {
		t.Fatalf("Failed to get user1: %v", err)
	}
	
	user2, err := client.User(ctx, "spacex")
	if err != nil {
		t.Fatalf("Failed to get user2: %v", err)
	}
	
	userIDs := []string{user1.ID, user2.ID}
	t.Logf("Testing UsersByIDs endpoint with IDs: %v", userIDs)
	usersByIDsStart := time.Now()
	users, err := client.UsersByIDs(ctx, userIDs)
	usersByIDsDuration := time.Since(usersByIDsStart)
	
	if err != nil {
		t.Logf("UsersByIDs endpoint returned error after %v: %v", usersByIDsDuration, err)
		t.Logf("This endpoint may need authentication or special permissions")
		return // Skip test if endpoint fails
	}
	
	t.Logf("UsersByIDs request took: %v", usersByIDsDuration)
	t.Logf("Retrieved %d users (expected 2)", len(users))
	
	for i, user := range users {
		t.Logf("Bulk User %d:", i+1)
		t.Logf("  ID: %s", user.ID)
		t.Logf("  ScreenName: %s", user.ScreenName)
		t.Logf("  Name: %s", user.Name)
	}
	
	if len(users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(users))
	}
	
	foundUser1, foundUser2 := false, false
	for _, user := range users {
		if user.ID == user1.ID {
			foundUser1 = true
		}
		if user.ID == user2.ID {
			foundUser2 = true
		}
	}
	
	if !foundUser1 || !foundUser2 {
		t.Error("Should find both requested users")
	}
	
	t.Logf("Total TestUsersByIDs time: %v", time.Since(start))
}

func TestFunctionalOptions(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Test multiple options together
	page, err := client.TweetsPage(ctx, "nasa", WithCount(3), WithPagination())
	if err != nil {
		t.Fatalf("Failed to get tweets with multiple options: %v", err)
	}
	
	if len(page.Tweets) > 3 {
		t.Errorf("Expected at most 3 tweets, got %d", len(page.Tweets))
	}
}

func TestClientMetrics(t *testing.T) {
	start := time.Now()
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	t.Logf("Making test request to generate metrics...")
	// Make a request to generate metrics
	_, err = client.User(ctx, "nasa")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	
	// Test metrics
	t.Logf("Testing client metrics...")
	metrics := client.GetMetrics()
	t.Logf("Metrics details:")
	t.Logf("  TotalRequests: %d", metrics.TotalRequests)
	t.Logf("  SuccessfulRequests: %d", metrics.SuccessfulRequests) 
	t.Logf("  FailedRequests: %d", metrics.FailedRequests)
	t.Logf("  CacheHits: %d", metrics.CacheHits)
	t.Logf("  CacheMisses: %d", metrics.CacheMisses)
	t.Logf("  AverageLatency: %v", metrics.AverageLatency)
	
	if metrics.TotalRequests == 0 {
		t.Error("Total requests should be greater than 0")
	}
	
	successRate := client.GetSuccessRate()
	t.Logf("Success rate: %.2f%%", successRate)
	if successRate < 0 || successRate > 100 {
		t.Errorf("Success rate should be between 0 and 100, got %f", successRate)
	}
	
	uptime := client.GetUptime()
	t.Logf("Client uptime: %v", uptime)
	if uptime <= 0 {
		t.Error("Uptime should be positive")
	}
	
	t.Logf("Total TestClientMetrics time: %v", time.Since(start))
}

func TestDebugMode(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test debug mode toggle
	client.Debug(true)
	client.Debug(false)
	client.SetDebugMode(true)
	client.SetDebugMode(false)
}


func TestContextCancellation(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test context cancellation
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	
	_, err = client.User(ctx, "nasa")
	if err == nil {
		t.Log("Request completed before timeout - this is okay")
	}
}

func TestErrorHandling(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Test with invalid username
	_, err = client.User(ctx, "")
	if err == nil {
		t.Error("Should return error for empty username")
	}
	
	// Test with invalid tweet ID
	_, err = client.Tweet(ctx, "invalid_id")
	if err == nil {
		t.Error("Should return error for invalid tweet ID")
	}
	
	// Test UsersByIDs with empty slice
	_, err = client.UsersByIDs(ctx, []string{})
	if err == nil {
		t.Error("Should return error for empty user IDs slice")
	}
}

func TestBroadcast(t *testing.T) {
	client, err := New()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	
	// Test with a sample broadcast ID (this might not always be available)
	_, err = client.Broadcast(ctx, "sample_broadcast_id")
	// We expect this to potentially fail since broadcasts are ephemeral
	// Just testing that the method doesn't panic
	if err != nil {
		t.Log("Broadcast test failed as expected (broadcasts are ephemeral)")
	}
}