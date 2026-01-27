package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// TokenStore handles bootstrap token storage and consumption
type TokenStore struct {
	rdb *redis.Client
}

// NewTokenStore creates a new token store
func NewTokenStore(rdb *redis.Client) *TokenStore {
	return &TokenStore{rdb: rdb}
}

// TokenData represents the data stored in Redis for a bootstrap token
type TokenData struct {
	NodeID int `json:"nodeId"`
}

// GenerateToken generates a random token string
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateToken creates a new bootstrap token in Redis
func (ts *TokenStore) CreateToken(ctx context.Context, nodeID int, ttlSec int) (string, error) {
	token, err := GenerateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	key := fmt.Sprintf("bootstrap:token:%s", token)
	data := TokenData{NodeID: nodeID}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token data: %w", err)
	}

	err = ts.rdb.Set(ctx, key, jsonData, time.Duration(ttlSec)*time.Second).Err()
	if err != nil {
		return "", fmt.Errorf("failed to store token in Redis: %w", err)
	}

	return token, nil
}

// ValidateToken checks if a token exists without consuming it
func (ts *TokenStore) ValidateToken(ctx context.Context, token string) (bool, error) {
	key := fmt.Sprintf("bootstrap:token:%s", token)
	exists, err := ts.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check token existence: %w", err)
	}
	return exists > 0, nil
}

// GetTokenData retrieves token data without consuming it
func (ts *TokenStore) GetTokenData(ctx context.Context, token string) (*TokenData, error) {
	key := fmt.Sprintf("bootstrap:token:%s", token)
	jsonData, err := ts.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // Token not found or expired
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get token data: %w", err)
	}

	var data TokenData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token data: %w", err)
	}

	return &data, nil
}

// ConsumeToken atomically consumes a token and returns the node ID
// This is the ONLY place where a token should be consumed
// Uses Lua script to ensure atomicity: check existence, read data, delete key
func (ts *TokenStore) ConsumeToken(ctx context.Context, token string) (int, error) {
	key := fmt.Sprintf("bootstrap:token:%s", token)
	
	// Lua script for atomic consume operation
	script := `
		local key = KEYS[1]
		local data = redis.call('GET', key)
		if not data then
			return nil
		end
		redis.call('DEL', key)
		return data
	`
	
	result, err := ts.rdb.Eval(ctx, script, []string{key}).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to execute consume script: %w", err)
	}
	
	if result == nil {
		return 0, fmt.Errorf("token not found or already consumed")
	}
	
	jsonData, ok := result.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected result type from Redis")
	}
	
	var data TokenData
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return 0, fmt.Errorf("failed to unmarshal token data: %w", err)
	}
	
	return data.NodeID, nil
}
