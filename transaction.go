package xapi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Constants for transaction ID generation
const (
	DefaultKeyword         = "obfiowerehiring"
	AdditionalRandomNumber = 3
	TwitterEpoch           = 1682924400 // Unix timestamp base
)

// TransactionGenerator is the unified, production-ready transaction ID generator
// with intelligent caching, automatic refresh, and robust error handling
type TransactionGenerator struct {
	config *ProductionConfig
	
	// Cache layers with different lifetimes
	htmlDataCache    *CacheEntry   // HTML data with 6-hour lifetime
	animationCache   *CacheEntry   // Animation keys with 3-hour lifetime
	verificationCache *CacheEntry  // Verification keys with 6-hour lifetime
	
	// Core data (real algorithm implementation)
	homePageHTML     string        // Cached HTML from Twitter homepage
	onDemandFileHTML string        // Cached ondemand.s file content
	keyBytes         []int         // Decoded verification key bytes
	animationKey     string        // Generated animation key
	rowIndex         int           // Dynamic row index from ondemand file
	keyBytesIndices  []int         // Dynamic indices from ondemand file
	key              string        // Raw verification key
	
	// Thread safety
	mu           sync.RWMutex
	refreshMutex sync.Mutex  // Separate mutex for refresh operations
	
	// Production features
	metrics      *GeneratorMetrics
	initialized  bool
	lastRefresh  time.Time
	httpClient   *http.Client
}

// CacheEntry represents a cached piece of data with expiration
type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
}

// GeneratorMetrics tracks transaction generator performance
type GeneratorMetrics struct {
	TotalGenerations    int64     `json:"total_generations"`
	CacheHits          int64     `json:"cache_hits"`
	CacheMisses        int64     `json:"cache_misses"`
	HTMLDataFetches    int64     `json:"html_data_fetches"`
	RefreshAttempts    int64     `json:"refresh_attempts"`
	RefreshFailures    int64     `json:"refresh_failures"`
	AverageGenTime     float64   `json:"average_generation_time_ms"`
	LastRefreshTime    time.Time `json:"last_refresh_time"`
}

// NewTransactionGenerator creates a unified transaction generator
func NewTransactionGenerator() (*TransactionGenerator, error) {
	return NewTransactionGeneratorWithConfig(DefaultProductionConfig())
}

// NewTransactionGeneratorWithConfig creates a transaction generator with custom config
func NewTransactionGeneratorWithConfig(config *ProductionConfig) (*TransactionGenerator, error) {
	if config == nil {
		config = DefaultProductionConfig()
	}
	
	generator := &TransactionGenerator{
		config:  config,
		metrics: &GeneratorMetrics{},
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	
	// Initialize with fresh data
	if err := generator.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize transaction generator: %w", err)
	}
	
	return generator, nil
}

// initialize fetches initial data and sets up caches
func (tg *TransactionGenerator) initialize() error {
	tg.refreshMutex.Lock()
	defer tg.refreshMutex.Unlock()
	
	start := time.Now()
	defer func() {
		tg.updateGenerationTime(time.Since(start))
	}()
	
	// Fetch real data from Twitter
	if err := tg.fetchTwitterData(); err != nil {
		return fmt.Errorf("failed to fetch Twitter data: %w", err)
	}
	
	// Extract real algorithm data
	if err := tg.extractAlgorithmData(); err != nil {
		return fmt.Errorf("failed to extract algorithm data: %w", err)
	}
	
	// Initialize cache entries with real data
	now := time.Now()
	tg.htmlDataCache = &CacheEntry{
		Data:      map[string]string{"homepage": tg.homePageHTML, "ondemand": tg.onDemandFileHTML},
		CreatedAt: now,
		ExpiresAt: now.Add(tg.config.HTMLDataCacheLifetime),
	}
	
	tg.animationCache = &CacheEntry{
		Data:      tg.animationKey,
		CreatedAt: now,
		ExpiresAt: now.Add(tg.config.AnimationKeyLifetime),
	}
	
	tg.verificationCache = &CacheEntry{
		Data:      tg.key,
		CreatedAt: now,
		ExpiresAt: now.Add(tg.config.HTMLDataCacheLifetime),
	}
	
	tg.initialized = true
	tg.lastRefresh = now
	tg.metrics.LastRefreshTime = now
	
	return nil
}

// Generate creates a new transaction ID with intelligent caching
func (tg *TransactionGenerator) Generate(method, path string) (string, error) {
	start := time.Now()
	defer func() {
		tg.updateGenerationTime(time.Since(start))
		tg.incrementTotalGenerations()
	}()
	
	// Check if we need to refresh any cached data
	if tg.needsRefresh() {
		if err := tg.Refresh(); err != nil {
			// Log error but continue with potentially stale data
			// In production, this would be logged to monitoring system
		}
	}
	
	// Generate unique transaction ID
	return tg.generateUniqueTransactionID(method, path)
}

// needsRefresh checks if any cached data needs refreshing based on production config
func (tg *TransactionGenerator) needsRefresh() bool {
	tg.mu.RLock()
	defer tg.mu.RUnlock()
	
	if !tg.initialized {
		return true
	}
	
	now := time.Now()
	
	// Check each cache layer
	if tg.htmlDataCache != nil && now.After(tg.htmlDataCache.ExpiresAt) {
		return true
	}
	
	if tg.animationCache != nil && now.After(tg.animationCache.ExpiresAt) {
		return true
	}
	
	if tg.verificationCache != nil && now.After(tg.verificationCache.ExpiresAt) {
		return true
	}
	
	return false
}

// Refresh refreshes all cached data
func (tg *TransactionGenerator) Refresh() error {
	tg.refreshMutex.Lock()
	defer tg.refreshMutex.Unlock()
	
	tg.metrics.RefreshAttempts++
	
	// Fetch fresh data from Twitter using real algorithm
	if err := tg.fetchTwitterData(); err != nil {
		tg.metrics.RefreshFailures++
		return fmt.Errorf("failed to fetch Twitter data: %w", err)
	}
	
	// Re-extract algorithm data from fresh HTML
	if err := tg.extractAlgorithmData(); err != nil {
		tg.metrics.RefreshFailures++
		return fmt.Errorf("failed to extract algorithm data: %w", err)
	}
	
	// Update caches with fresh data
	if err := tg.updateCachesWithFreshData(); err != nil {
		tg.metrics.RefreshFailures++
		return fmt.Errorf("failed to update caches: %w", err)
	}
	
	tg.mu.Lock()
	tg.lastRefresh = time.Now()
	tg.metrics.LastRefreshTime = tg.lastRefresh
	tg.mu.Unlock()
	
	return nil
}

// ForceRefresh immediately refreshes all data regardless of cache expiration
func (tg *TransactionGenerator) ForceRefresh() error {
	// Invalidate all caches
	tg.mu.Lock()
	if tg.htmlDataCache != nil {
		tg.htmlDataCache.ExpiresAt = time.Now().Add(-1 * time.Hour)
	}
	if tg.animationCache != nil {
		tg.animationCache.ExpiresAt = time.Now().Add(-1 * time.Hour)
	}
	if tg.verificationCache != nil {
		tg.verificationCache.ExpiresAt = time.Now().Add(-1 * time.Hour)
	}
	tg.mu.Unlock()
	
	return tg.Refresh()
}

// ForceRefreshTransactionID is an alias for backward compatibility
func (tg *TransactionGenerator) ForceRefreshTransactionID() error {
	return tg.ForceRefresh()
}

// updateCachesWithFreshData updates all caches with freshly fetched data
func (tg *TransactionGenerator) updateCachesWithFreshData() error {
	now := time.Now()
	
	// Update HTML data cache
	tg.htmlDataCache = &CacheEntry{
		Data:      map[string]string{"homepage": tg.homePageHTML, "ondemand": tg.onDemandFileHTML},
		CreatedAt: now,
		ExpiresAt: now.Add(tg.config.HTMLDataCacheLifetime),
	}
	
	// Update animation cache
	tg.animationCache = &CacheEntry{
		Data:      tg.animationKey,
		CreatedAt: now,
		ExpiresAt: now.Add(tg.config.AnimationKeyLifetime),
	}
	
	// Update verification cache
	tg.verificationCache = &CacheEntry{
		Data:      tg.key,
		CreatedAt: now,
		ExpiresAt: now.Add(tg.config.HTMLDataCacheLifetime),
	}
	
	return nil
}

// generateUniqueTransactionID generates a unique transaction ID using crypto/rand
func (tg *TransactionGenerator) generateUniqueTransactionID(method, path string) (string, error) {
	// Calculate precise timestamp
	nowMicro := time.Now().UnixMicro()
	epochMicro := int64(TwitterEpoch) * 1000000
	timeNow := int64((nowMicro - epochMicro) / 1000000)
	
	// Convert time to bytes (little endian)
	timeNowBytes := make([]byte, 4)
	for i := 0; i < 4; i++ {
		timeNowBytes[i] = byte((timeNow >> (i * 8)) & 0xFF)
	}
	
	// Create hash string with current animation key
	tg.mu.RLock()
	currentAnimationKey := tg.animationKey
	currentKeyBytes := tg.keyBytes
	tg.mu.RUnlock()
	
	hashString := fmt.Sprintf("%s!%s!%d%s%s", method, path, timeNow, DefaultKeyword, currentAnimationKey)
	hash := sha256.Sum256([]byte(hashString))
	hashBytes := hash[:16]
	
	// Generate random XOR byte using crypto/rand
	var randomBytes [1]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return "", fmt.Errorf("failed to generate random number: %w", err)
	}
	
	// Build final byte array
	var bytesArr []byte
	
	// Add key bytes (convert from []int to []byte)
	for _, kb := range currentKeyBytes {
		bytesArr = append(bytesArr, byte(kb))
	}
	
	bytesArr = append(bytesArr, timeNowBytes...)
	bytesArr = append(bytesArr, hashBytes...)
	bytesArr = append(bytesArr, byte(AdditionalRandomNumber))
	
	// XOR encrypt
	result := []byte{randomBytes[0]}
	for _, b := range bytesArr {
		result = append(result, b^randomBytes[0])
	}
	
	// Base64 encode and remove padding
	encoded := base64.StdEncoding.EncodeToString(result)
	return trimBase64Padding(encoded), nil
}

// Real algorithm implementation methods

// fetchTwitterData fetches the required HTML data from Twitter
func (tg *TransactionGenerator) fetchTwitterData() error {
	// Fetch home page
	req, err := http.NewRequest("GET", "https://x.com", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := tg.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch home page: %w", err)
	}
	defer resp.Body.Close()

	homeBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read home page: %w", err)
	}
	tg.homePageHTML = string(homeBytes)
	tg.metrics.HTMLDataFetches++

	// Extract ondemand file URL
	onDemandFileRegex := regexp.MustCompile(`['\"]{1}ondemand\.s['\"]{1}:\s*['\"]{1}([\w]*)['\"]{1}`)
	matches := onDemandFileRegex.FindStringSubmatch(tg.homePageHTML)
	if len(matches) < 2 {
		return fmt.Errorf("ondemand file URL not found in home page")
	}

	onDemandURL := fmt.Sprintf("https://abs.twimg.com/responsive-web/client-web/ondemand.s.%sa.js", matches[1])

	// Fetch ondemand file
	req, err = http.NewRequest("GET", onDemandURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create ondemand request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")

	resp, err = tg.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch ondemand file: %w", err)
	}
	defer resp.Body.Close()

	onDemandBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read ondemand file: %w", err)
	}
	tg.onDemandFileHTML = string(onDemandBytes)

	return nil
}

// extractAlgorithmData extracts keys and generates animation data using the corrected algorithm
func (tg *TransactionGenerator) extractAlgorithmData() error {
	// Extract indices from ondemand file
	if err := tg.extractIndices(); err != nil {
		return fmt.Errorf("failed to extract indices: %w", err)
	}

	// Extract key from home page
	if err := tg.extractKey(); err != nil {
		return fmt.Errorf("failed to extract key: %w", err)
	}

	// Generate animation key using corrected matrix algorithm
	if err := tg.generateAnimationKey(); err != nil {
		return fmt.Errorf("failed to generate animation key: %w", err)
	}

	return nil
}

// extractIndices extracts the dynamic indices from the ondemand.s file
func (tg *TransactionGenerator) extractIndices() error {
	indicesRegex := regexp.MustCompile(`\(\w{1}\[(\d{1,2})\],\s*16\)`)
	matches := indicesRegex.FindAllStringSubmatch(tg.onDemandFileHTML, -1)
	if len(matches) == 0 {
		return fmt.Errorf("no indices found in ondemand file")
	}

	var indices []int
	for _, match := range matches {
		if len(match) > 1 {
			index, err := strconv.Atoi(match[1])
			if err != nil {
				return fmt.Errorf("failed to parse index %s: %w", match[1], err)
			}
			indices = append(indices, index)
		}
	}

	if len(indices) == 0 {
		return fmt.Errorf("no valid indices extracted")
	}

	tg.rowIndex = indices[0]
	tg.keyBytesIndices = indices[1:]
	return nil
}

// extractKey extracts the twitter-site-verification key from the home page
func (tg *TransactionGenerator) extractKey() error {
	doc, err := html.Parse(strings.NewReader(tg.homePageHTML))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}

	key := tg.findTwitterSiteVerification(doc)
	if key == "" {
		return fmt.Errorf("twitter-site-verification meta tag not found")
	}

	tg.key = key

	// Decode key bytes
	decoded, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return fmt.Errorf("failed to decode key: %w", err)
	}

	tg.keyBytes = make([]int, len(decoded))
	for i, b := range decoded {
		tg.keyBytes[i] = int(b)
	}

	return nil
}

// findTwitterSiteVerification recursively searches for the twitter-site-verification meta tag
func (tg *TransactionGenerator) findTwitterSiteVerification(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var name, content string
		for _, attr := range n.Attr {
			switch attr.Key {
			case "name":
				name = attr.Val
			case "content":
				content = attr.Val
			}
		}
		if name == "twitter-site-verification" {
			return content
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if result := tg.findTwitterSiteVerification(c); result != "" {
			return result
		}
	}

	return ""
}

// generateAnimationKey generates the animation key using frame data and timing (WITH CORRECTED MATRIX)
func (tg *TransactionGenerator) generateAnimationKey() error {
	// Extract animation frames from the HTML (2D array)
	frames, err := tg.extractAnimationFrames()
	if err != nil {
		return fmt.Errorf("failed to extract animation frames: %w", err)
	}

	// Calculate frame timing - match Python exactly
	actualRowIndex := tg.rowIndex
	if actualRowIndex >= len(tg.keyBytes) {
		actualRowIndex = len(tg.keyBytes) - 1
	}
	rowIndex := tg.keyBytes[actualRowIndex] % 16

	frameTime := 1
	for _, index := range tg.keyBytesIndices {
		if index < len(tg.keyBytes) {
			frameTime *= tg.keyBytes[index] % 16
		}
	}
	frameTime = int(jsRound(float64(frameTime)/10)) * 10

	// Select the frame row - Python: frame_row = arr[row_index]
	var frameRow []int
	if len(frames) == 0 {
		return fmt.Errorf("no frame data available")
	}
	
	if rowIndex < len(frames) {
		frameRow = frames[rowIndex]
	} else {
		frameRow = frames[0] // Fallback to first row
	}

	// Calculate target time
	targetTime := float64(frameTime) / 4096.0

	// Generate animation key using corrected algorithm
	tg.animationKey = tg.animate(frameRow, targetTime)
	return nil
}

// extractAnimationFrames extracts row data from a single selected SVG animation frame
func (tg *TransactionGenerator) extractAnimationFrames() ([][]int, error) {
	// First, determine which frame to select using key_bytes[5] % 4 (Python approach)
	if len(tg.keyBytes) <= 5 {
		return nil, fmt.Errorf("key_bytes too short for frame selection")
	}
	
	frameIndex := tg.keyBytes[5] % 4
	frameID := fmt.Sprintf("loading-x-anim-%d", frameIndex)
	
	// Find the specific frame element using simple string search
	framePattern := regexp.MustCompile(fmt.Sprintf(`id=['"]%s['"][^>]*>(.*?)</g>`, frameID))
	frameMatch := framePattern.FindStringSubmatch(tg.homePageHTML)
	
	if len(frameMatch) < 2 {
		return nil, fmt.Errorf("❌ EXTRACTION FAILED: frame %s not found in HTML - no fallback", frameID)
	}
	
	frameContent := frameMatch[1]
	
	// Look for path elements within the frame content
	pathPattern := regexp.MustCompile(`<path[^>]*\sd=['"]([^'"]*?)['"][^>]*>`)
	pathMatches := pathPattern.FindAllStringSubmatch(frameContent, -1)
	
	var selectedPathData string
	for _, pathMatch := range pathMatches {
		if len(pathMatch) >= 2 && len(pathMatch[1]) > 9 && strings.Contains(pathMatch[1], "C") {
			selectedPathData = pathMatch[1]
			break
		}
	}
	
	if selectedPathData == "" {
		return nil, fmt.Errorf("❌ EXTRACTION FAILED: no valid path data found in frame %s - no fallback", frameID)
	}
	
	// Extract data exactly as Python does: pathData[9:].split("C")
	curveData := selectedPathData[9:]
	curveParts := strings.Split(curveData, "C")
	
	var rows [][]int
	// For each "C" segment, extract all numbers to create a row
	for _, part := range curveParts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		
		// Replace all non-digits with spaces, then split
		numberRegex := regexp.MustCompile(`[^\d]+`)
		cleanedPart := numberRegex.ReplaceAllString(part, " ")
		cleanedPart = strings.TrimSpace(cleanedPart)
		
		if cleanedPart == "" {
			continue
		}
		
		var row []int
		for _, numStr := range strings.Fields(cleanedPart) {
			if num, err := strconv.Atoi(numStr); err == nil {
				row = append(row, num)
			}
		}
		
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	
	if len(rows) == 0 {
		return nil, fmt.Errorf("❌ EXTRACTION FAILED: no valid rows extracted from frame %s - no fallback", frameID)
	}
	
	return rows, nil
}

// animate performs the cubic bezier animation calculation WITH CORRECTED MATRIX ORDERING
func (tg *TransactionGenerator) animate(frameRow []int, targetTime float64) string {
	if len(frameRow) < 15 {
		// Pad with zeros if needed
		for len(frameRow) < 15 {
			frameRow = append(frameRow, 0)
		}
	}

	// Extract color values
	fromColor := []float64{float64(frameRow[0]), float64(frameRow[1]), float64(frameRow[2]), 1.0}
	toColor := []float64{float64(frameRow[3]), float64(frameRow[4]), float64(frameRow[5]), 1.0}

	// Extract rotation values
	fromRotation := []float64{0.0}
	toRotation := []float64{solve(float64(frameRow[6]), 60.0, 360.0, true)}

	// Create cubic bezier curves from remaining frame data
	curveValues := make([]float64, 0)
	for i, val := range frameRow[7:] {
		solved := solve(float64(val), isOdd(i), 1.0, false)
		curveValues = append(curveValues, solved)
	}

	// Evaluate cubic bezier curve properly
	val := evaluateCubicBezier(curveValues, targetTime)

	// Interpolate colors and rotation
	color := interpolate(fromColor, toColor, val)
	rotation := interpolate(fromRotation, toRotation, val)

	// Generate matrix from rotation WITH CORRECTED ORDERING
	matrix := convertRotationToMatrix(rotation[0])

	// Build result string
	var strArr []string

	// Add color components as hex
	for i := 0; i < 3; i++ {
		rounded := math.Round(color[i])
		strArr = append(strArr, fmt.Sprintf("%x", int(rounded)))
	}

	// Add matrix components
	for _, value := range matrix {
		rounded := math.Round(value*100) / 100
		if rounded < 0 {
			rounded = -rounded
		}
		hexValue := floatToHex(rounded)
		if strings.HasPrefix(hexValue, ".") {
			strArr = append(strArr, "0"+hexValue)
		} else if hexValue == "" {
			strArr = append(strArr, "0")
		} else {
			strArr = append(strArr, hexValue)
		}
	}

	// Add trailing zeros
	strArr = append(strArr, "0", "0")

	// Remove dots and dashes
	result := strings.Join(strArr, "")
	result = strings.ReplaceAll(result, ".", "")
	result = strings.ReplaceAll(result, "-", "")

	return strings.ToLower(result)
}

// Metrics and monitoring methods
func (tg *TransactionGenerator) incrementTotalGenerations() {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	tg.metrics.TotalGenerations++
}

func (tg *TransactionGenerator) updateGenerationTime(duration time.Duration) {
	tg.mu.Lock()
	defer tg.mu.Unlock()
	
	genTimeMs := float64(duration.Nanoseconds()) / 1e6
	
	if tg.metrics.TotalGenerations == 1 {
		tg.metrics.AverageGenTime = genTimeMs
	} else {
		// Exponential moving average
		alpha := 0.1
		tg.metrics.AverageGenTime = (1-alpha)*tg.metrics.AverageGenTime + alpha*genTimeMs
	}
}

// GetMetrics returns current generator metrics
func (tg *TransactionGenerator) GetMetrics() *GeneratorMetrics {
	tg.mu.RLock()
	defer tg.mu.RUnlock()
	
	metrics := *tg.metrics
	return &metrics
}

// IsStale returns true if the generator data needs refreshing
func (tg *TransactionGenerator) IsStale() bool {
	return tg.needsRefresh()
}

// GetStats returns information about the generator state
func (tg *TransactionGenerator) GetStats() *TransactionGeneratorStats {
	tg.mu.RLock()
	defer tg.mu.RUnlock()
	
	return &TransactionGeneratorStats{
		KeyLength:      len(tg.keyBytes),
		IndicesCount:   len(tg.keyBytesIndices),
		AnimationKey:   tg.animationKey,
		LastFetchTime:  tg.lastRefresh,
		IsStale:        tg.needsRefresh(),
		HomePageLength: len(tg.homePageHTML),
		OnDemandLength: len(tg.onDemandFileHTML),
	}
}

// TransactionGeneratorStats provides information about the generator state
type TransactionGeneratorStats struct {
	KeyLength      int       `json:"key_length"`
	IndicesCount   int       `json:"indices_count"`
	AnimationKey   string    `json:"animation_key"`
	LastFetchTime  time.Time `json:"last_fetch_time"`
	IsStale        bool      `json:"is_stale"`
	HomePageLength int       `json:"home_page_length"`
	OnDemandLength int       `json:"ondemand_length"`
}

// Utility functions
func trimBase64Padding(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}

// Helper functions for animation calculation

// solve processes a value with min/max scaling and optional rounding
func solve(value, minVal, maxVal float64, rounding bool) float64 {
	result := value*(maxVal-minVal)/255 + minVal
	if rounding {
		return math.Floor(result)
	}
	return math.Round(result*100) / 100
}

// isOdd checks if a number is odd and returns the appropriate value for bezier calculations
func isOdd(num int) float64 {
	if num%2 == 1 { // Match Rust: check specifically for 1, not just non-zero
		return -1.0
	}
	return 0.0
}

// interpolate performs linear interpolation between two arrays of values
func interpolate(from, to []float64, t float64) []float64 {
	result := make([]float64, len(from))
	for i := range from {
		result[i] = from[i] + (to[i]-from[i])*t
		result[i] = math.Max(0, math.Min(255, result[i]))
	}
	return result
}

// convertRotationToMatrix converts a rotation angle to a 2D transformation matrix
// FIXED: Correct browser matrix ordering: [cos, sin, -sin, cos]
func convertRotationToMatrix(angle float64) []float64 {
	radians := angle * math.Pi / 180
	cos := math.Cos(radians)
	sin := math.Sin(radians)

	// FIXED: Correct browser matrix ordering: [cos, sin, -sin, cos]
	// This matches the expected output from JavaScript getComputedStyle()
	return []float64{cos, sin, -sin, cos}
}

// floatToHex uses correct mathematical logic like the Rust implementation
// This uses proper modulo and division instead of replicating Python bugs
func floatToHex(x float64) string {
	if x == 0.0 {
		return "0"
	}

	var result []string
	quotient := int(x)
	fraction := x - float64(quotient)

	parseDigit := func(value int) string {
		if value > 9 {
			return string(rune(value + 55))
		}
		return strconv.Itoa(value)
	}

	// Handle integer part with correct mathematical logic
	if quotient == 0 {
		result = append(result, "0")
	} else {
		var digits []string
		for quotient > 0 {
			remainder := quotient % 16  // CORRECT: proper modulo
			quotient = quotient / 16    // CORRECT: proper division
			digits = append([]string{parseDigit(remainder)}, digits...)
		}
		result = append(result, digits...)
	}

	// Handle fractional part
	if fraction > 0 {
		result = append(result, ".")
		
		for fraction > 0 && len(result) < 20 { // Avoid infinite loops
			fraction *= 16
			integer := int(fraction)
			fraction -= float64(integer)
			result = append(result, parseDigit(integer))
		}
	}

	return strings.Join(result, "")
}

// jsRound matches Rust js_round implementation exactly
func jsRound(num float64) float64 {
	decimalPart := num - math.Trunc(num)
	if decimalPart == -0.5 {
		return math.Ceil(num)
	}
	return math.Round(num)
}

// evaluateCubicBezier implements Rust-style binary search cubic curve algorithm
// This matches the Rust implementation exactly instead of standard bezier formula
func evaluateCubicBezier(controlPoints []float64, t float64) float64 {
	if len(controlPoints) < 4 {
		return t
	}
	
	// Handle edge cases like Rust implementation
	if t <= 0.0 {
		startGradient := 0.0
		if controlPoints[0] > 0.0 {
			startGradient = controlPoints[1] / controlPoints[0]
		} else if controlPoints[1] == 0.0 && controlPoints[2] > 0.0 {
			startGradient = controlPoints[3] / controlPoints[2]
		}
		return startGradient * t
	}
	
	if t >= 1.0 {
		endGradient := 0.0
		if controlPoints[2] < 1.0 {
			endGradient = (controlPoints[3] - 1.0) / (controlPoints[2] - 1.0)
		} else if controlPoints[2] == 1.0 && controlPoints[0] < 1.0 {
			endGradient = (controlPoints[1] - 1.0) / (controlPoints[0] - 1.0)
		}
		return 1.0 + endGradient*(t-1.0)
	}
	
	// Binary search approach from Rust implementation
	startValue := 0.0
	endValue := 1.0
	mid := 0.0
	
	for startValue < endValue {
		mid = (startValue + endValue) / 2.0
		xEst := cubicCalculate(controlPoints[0], controlPoints[2], mid)
		if math.Abs(t-xEst) < 0.00001 {
			return cubicCalculate(controlPoints[1], controlPoints[3], mid)
		}
		if xEst < t {
			startValue = mid
		} else {
			endValue = mid
		}
	}
	
	return cubicCalculate(controlPoints[1], controlPoints[3], mid)
}

// cubicCalculate helper function matching Rust implementation
func cubicCalculate(a, b, m float64) float64 {
	return 3.0*a*(1.0-m)*(1.0-m)*m + 3.0*b*(1.0-m)*m*m + m*m*m
}