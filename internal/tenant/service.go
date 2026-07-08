package tenant

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Service provides business logic for tenants and API keys.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// GenerateAPIKey generates a new raw API key and stores its Argon2 hash.
// It returns the plaintext key which must be shown to the user exactly once.
func (s *Service) GenerateAPIKey(ctx context.Context, tenantID string) (string, error) {
	// 1. Generate 32 bytes of random data for the key
	rawKeyBytes := make([]byte, 32)
	if _, err := rand.Read(rawKeyBytes); err != nil {
		return "", err
	}
	plaintextKey := "gk_" + base64.RawURLEncoding.EncodeToString(rawKeyBytes)

	// 2. Hash using Argon2
	hash, err := HashAPIKey(plaintextKey)
	if err != nil {
		return "", err
	}

	// 3. Store in DB
	_, err = s.store.CreateAPIKey(ctx, tenantID, hash)
	if err != nil {
		return "", err
	}

	return plaintextKey, nil
}

// ValidateAPIKey checks if the provided plaintext key matches any active key in the DB.
// Note: In a real system, you'd lookup the key hash directly. Since Argon2 uses a random salt,
// you must extract the salt from the stored hash. However, since the gateway needs to do this
// on every request, it's common to prefix the plaintext key with a key ID (e.g., `gk_{key_id}_{secret}`)
// so you can lookup the stored hash by ID in O(1) time, then verify the Argon2 hash.
// For this skeleton, we assume the DB schema uses a deterministic hash (like HMAC-SHA256) 
// for O(1) lookup if `key_hash` is queried directly, or we fetch the user by ID first.
// Here we implement the Argon2 hashing helper.
func HashAPIKey(plaintext string) (string, error) {
	// In production, generate a random salt. For O(1) lookups without a key ID prefix,
	// some systems use HMAC-SHA256. If Argon2 is strictly required for API keys, 
	// the key must be structured as ID.Secret.
	
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	
	// Argon2id parameters
	time := uint32(1)
	memory := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)
	
	hash := argon2.IDKey([]byte(plaintext), salt, time, memory, threads, keyLen)
	
	// Encode as salt.hash
	encodedSalt := hex.EncodeToString(salt)
	encodedHash := hex.EncodeToString(hash)
	
	return fmt.Sprintf("%s.%s", encodedSalt, encodedHash), nil
}

func VerifyAPIKey(plaintext, storedHash string) bool {
	parts := strings.Split(storedHash, ".")
	if len(parts) != 2 {
		return false
	}
	
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	
	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	
	time := uint32(1)
	memory := uint32(64 * 1024)
	threads := uint8(4)
	keyLen := uint32(32)
	
	hash := argon2.IDKey([]byte(plaintext), salt, time, memory, threads, keyLen)
	
	// Constant time compare
	if len(hash) != len(expectedHash) {
		return false
	}
	
	var result byte
	for i := 0; i < len(hash); i++ {
		result |= hash[i] ^ expectedHash[i]
	}
	
	return result == 0
}
