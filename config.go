package xapi

import "time"

// ProductionConfig holds production-optimized configuration
type ProductionConfig struct {
	// Caching strategy - longer cache times for production reliability
	HTMLDataCacheLifetime    time.Duration // How long to cache HTML data
	TransactionIDLifetime    time.Duration // How long to reuse transaction IDs
	AnimationKeyLifetime     time.Duration // How long to cache animation keys
	
	// Retry strategy - automatic recovery from failures
	EnableAutoRetry          bool          // Enable automatic retry on failures
	MaxRetryAttempts         int           // Maximum retry attempts per request
	RetryBackoffBase         time.Duration // Base backoff duration between retries
	RetryBackoffMultiplier   float64       // Backoff multiplier for exponential backoff
	
	// Error threshold for cache invalidation
	ErrorThresholdForRefresh int           // Number of errors before forcing refresh
	
	// Request timing and rate limiting
	RequestTimeout           time.Duration // Timeout for individual requests
	RateLimitRequests        float64       // Requests per second
	
	// Debug and monitoring
	EnableDebugLogging       bool          // Enable detailed debug logs
	EnableMetrics           bool          // Enable performance metrics
}

// DefaultProductionConfig returns optimized settings for production use
func DefaultProductionConfig() *ProductionConfig {
	return &ProductionConfig{
		// Aggressive caching for production stability
		HTMLDataCacheLifetime:    6 * time.Hour,    // Cache HTML data for 6 hours
		TransactionIDLifetime:    1 * time.Hour,     // Reuse transaction IDs for 1 hour
		AnimationKeyLifetime:     3 * time.Hour,     // Cache animation keys for 3 hours
		
		// Robust retry strategy
		EnableAutoRetry:          true,              // Always retry on failures
		MaxRetryAttempts:         3,                 // Up to 3 retries per request
		RetryBackoffBase:         500 * time.Millisecond, // Start with 500ms backoff
		RetryBackoffMultiplier:   2.0,               // Exponential backoff: 500ms, 1s, 2s
		
		// Conservative error handling
		ErrorThresholdForRefresh: 2,                 // Refresh after 2 consecutive errors
		
		// Reasonable timeouts
		RequestTimeout:           30 * time.Second,   // 30s timeout per request
		RateLimitRequests:        50.0 / 60.0,       // 50 requests per minute
		
		// Production logging
		EnableDebugLogging:       false,             // Disable debug in production
		EnableMetrics:           true,               // Enable metrics collection
	}
}

// DevelopmentConfig returns settings optimized for development/testing
func DevelopmentConfig() *ProductionConfig {
	return &ProductionConfig{
		// Shorter cache times for faster iteration
		HTMLDataCacheLifetime:    10 * time.Minute,  // 10 minutes for development
		TransactionIDLifetime:    5 * time.Minute,   // 5 minutes for development
		AnimationKeyLifetime:     15 * time.Minute,  // 15 minutes for development
		
		// Aggressive retry for debugging
		EnableAutoRetry:          true,              
		MaxRetryAttempts:         2,                 // Fewer retries for faster feedback
		RetryBackoffBase:         200 * time.Millisecond,
		RetryBackoffMultiplier:   1.5,
		
		// Sensitive error handling
		ErrorThresholdForRefresh: 1,                 // Refresh after 1 error in dev
		
		// Shorter timeouts for faster iteration
		RequestTimeout:           15 * time.Second,
		RateLimitRequests:        100.0 / 60.0,      // Higher rate limit for testing
		
		// Full logging in development
		EnableDebugLogging:       true,
		EnableMetrics:           true,
	}
}

// UltraFreshConfig returns settings for maximum freshness (testing only)
func UltraFreshConfig() *ProductionConfig {
	return &ProductionConfig{
		// No caching - always fresh
		HTMLDataCacheLifetime:    0,                 // No caching
		TransactionIDLifetime:    0,                 // Generate new ID every time
		AnimationKeyLifetime:     0,                 // No caching
		
		// No retries to see raw performance
		EnableAutoRetry:          false,
		MaxRetryAttempts:         0,
		RetryBackoffBase:         0,
		RetryBackoffMultiplier:   1.0,
		
		// Immediate refresh on any error
		ErrorThresholdForRefresh: 1,
		
		// Standard timeouts
		RequestTimeout:           10 * time.Second,
		RateLimitRequests:        30.0 / 60.0,
		
		// Full debugging for ultra-fresh testing
		EnableDebugLogging:       true,
		EnableMetrics:           true,
	}
}