package xapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

// XPFFGenerator handles x-xp-forwarded-for header generation
type XPFFGenerator struct {
	baseKey string
}

// NavigatorProperties represents browser navigator properties
type NavigatorProperties struct {
	HasBeenActive string `json:"hasBeenActive"`
	UserAgent     string `json:"userAgent"`
	Webdriver     string `json:"webdriver"`
}

// XPFFPayload represents the payload structure for XPFF header
type XPFFPayload struct {
	NavigatorProperties NavigatorProperties `json:"navigator_properties"`
	CreatedAt           int64               `json:"created_at"`
}

// NewXPFFGenerator creates a new XPFF generator with the hardcoded base key
func NewXPFFGenerator() *XPFFGenerator {
	// Hardcoded base key from Twitter's implementation
	baseKey := "0e6be1f1e21ffc33590b888fd4dc81b19713e570e805d4e5df80a493c9571a05"
	return &XPFFGenerator{
		baseKey: baseKey,
	}
}

// GenerateXPFF generates the encrypted x-xp-forwarded-for header value
func (x *XPFFGenerator) GenerateXPFF(guestID, userAgent string) (string, error) {
	// Create the payload
	payload := XPFFPayload{
		NavigatorProperties: NavigatorProperties{
			HasBeenActive: "true",
			UserAgent:     userAgent,
			Webdriver:     "false",
		},
		CreatedAt: time.Now().UnixMilli(),
	}

	// Convert payload to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Generate encryption key
	encryptionKey, err := x.generateEncryptionKey(guestID)
	if err != nil {
		return "", fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Encrypt the payload
	encrypted, err := x.encryptAESGCM(payloadJSON, encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt payload: %w", err)
	}

	return hex.EncodeToString(encrypted), nil
}

// generateEncryptionKey derives the encryption key from base key and guest ID
func (x *XPFFGenerator) generateEncryptionKey(guestID string) ([]byte, error) {
	// Combine base key with URL-encoded guest ID
	combined := x.baseKey + guestID
	
	// Generate SHA-256 hash
	hash := sha256.Sum256([]byte(combined))
	
	return hash[:], nil
}

// encryptAESGCM encrypts data using AES-GCM
func (x *XPFFGenerator) encryptAESGCM(plaintext, key []byte) ([]byte, error) {
	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce (12 bytes for GCM)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt the plaintext
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	// Concatenate: nonce + ciphertext (which includes the auth tag)
	result := append(nonce, ciphertext...)
	
	return result, nil
}

// IsXPFFValid checks if the XPFF header is still valid (within 5 minutes)
func (x *XPFFGenerator) IsXPFFValid(createdAt int64) bool {
	now := time.Now().UnixMilli()
	// Valid for 5 minutes (300,000 milliseconds)
	return (now - createdAt) < 300000
}