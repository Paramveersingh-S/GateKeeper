package admin

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/argon2"
)

var (
	// In production, this should be an environment variable
	jwtSecret = []byte("super-secret-gatekeeper-jwt-key")
)

// GenerateJWT creates a new JWT token for a tenant
func GenerateJWT(tenantID string) (string, error) {
	claims := jwt.MapClaims{
		"tenant_id": tenantID,
		"exp":       time.Now().Add(24 * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT parses and validates a JWT token, returning the tenant ID
func ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		tenantID, ok := claims["tenant_id"].(string)
		if !ok {
			return "", errors.New("invalid tenant_id claim")
		}
		return tenantID, nil
	}

	return "", errors.New("invalid token")
}

// Argon2 parameters
const (
	timeCost    = 1
	memoryCost  = 64 * 1024
	threads     = 4
	keyLen      = 32
	saltLen     = 16
)

// HashAPIKey creates an Argon2 hash of a plaintext API key
func HashAPIKey(key string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(key), salt, timeCost, memoryCost, threads, keyLen)

	// Format: $argon2id$v=19$m=65536,t=1,p=4$salt$hash
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memoryCost, timeCost, threads, b64Salt, b64Hash), nil
}

// CompareAPIKeyHash checks if a plaintext key matches the stored Argon2 hash
func CompareAPIKeyHash(key, storedHash string) (bool, error) {
	parts := strings.Split(storedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid hash format")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	actualHash := argon2.IDKey([]byte(key), salt, timeCost, memoryCost, threads, keyLen)
	
	if subtle.ConstantTimeCompare(expectedHash, actualHash) == 1 {
		return true, nil
	}
	return false, nil
}
