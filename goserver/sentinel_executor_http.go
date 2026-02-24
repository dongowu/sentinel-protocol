package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// ExecuteToken is a single-use pass issued after a successful gate evaluation.
type ExecuteToken struct {
	ID        string    `json:"id"`
	Action    string    `json:"action"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Redeemed  bool      `json:"redeemed"`
}

// ExecuteGuard issues and validates one-time execution tokens.
// Each token can only be redeemed once, preventing replay attacks.
type ExecuteGuard struct {
	mu     sync.Mutex
	tokens map[string]*ExecuteToken
	ttl    time.Duration
}

// NewExecuteGuard creates an ExecuteGuard with the given token time-to-live.
// If ttl is zero or negative, defaults to 30 seconds.
func NewExecuteGuard(ttl time.Duration) *ExecuteGuard {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &ExecuteGuard{
		tokens: make(map[string]*ExecuteToken),
		ttl:    ttl,
	}
}

// Issue creates a new one-time token for the given action.
func (eg *ExecuteGuard) Issue(action string) *ExecuteToken {
	eg.mu.Lock()
	defer eg.mu.Unlock()

	id := generateTokenID()
	now := time.Now().UTC()
	tok := &ExecuteToken{
		ID:        id,
		Action:    action,
		IssuedAt:  now,
		ExpiresAt: now.Add(eg.ttl),
	}
	eg.tokens[id] = tok
	return tok
}

// Redeem consumes a one-time token. Returns an error if the token is missing,
// already used, or expired.
func (eg *ExecuteGuard) Redeem(tokenID string) (*ExecuteToken, error) {
	eg.mu.Lock()
	defer eg.mu.Unlock()

	tok, ok := eg.tokens[tokenID]
	if !ok {
		return nil, fmt.Errorf("token not found: %s", tokenID)
	}
	if tok.Redeemed {
		return nil, fmt.Errorf("token already redeemed: %s", tokenID)
	}
	if time.Now().UTC().After(tok.ExpiresAt) {
		return nil, fmt.Errorf("token expired: %s", tokenID)
	}

	tok.Redeemed = true
	return tok, nil
}

// PendingCount returns the number of unredeemed, unexpired tokens.
func (eg *ExecuteGuard) PendingCount() int {
	eg.mu.Lock()
	defer eg.mu.Unlock()

	now := time.Now().UTC()
	count := 0
	for _, tok := range eg.tokens {
		if !tok.Redeemed && now.Before(tok.ExpiresAt) {
			count++
		}
	}
	return count
}

func generateTokenID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return "tok-" + hex.EncodeToString(b)
}
