package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ApprovalChallenge represents a human-approval gate for high-risk actions.
type ApprovalChallenge struct {
	ID         string     `json:"id"`
	Action     string     `json:"action"`
	Prompt     string     `json:"prompt"`
	RiskScore  int        `json:"risk_score"`
	Status     string     `json:"status"` // pending, approved, rejected, expired
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
	DecisionBy string    `json:"decision_by,omitempty"`
}

// ApprovalService manages human-in-the-loop approval challenges for the Sentinel system.
type ApprovalService struct {
	challenges map[string]*ApprovalChallenge
	timeout    time.Duration
	mu         sync.RWMutex
	onExpire   func(challenge *ApprovalChallenge)
}

// NewApprovalService creates an ApprovalService with the given challenge timeout.
// If timeout is zero or negative, defaults to 5 minutes.
func NewApprovalService(timeout time.Duration) *ApprovalService {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &ApprovalService{
		challenges: make(map[string]*ApprovalChallenge),
		timeout:    timeout,
	}
}

// generateChallengeID produces an ID like "challenge-1708012345678-a3xf".
func generateChallengeID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("challenge-%d-%s", time.Now().UnixMilli(), string(suffix))
}

// StartChallenge creates a new pending approval challenge for the given action.
func (as *ApprovalService) StartChallenge(action, prompt string, score int) *ApprovalChallenge {
	now := time.Now().UTC()
	ch := &ApprovalChallenge{
		ID:        generateChallengeID(),
		Action:    action,
		Prompt:    prompt,
		RiskScore: score,
		Status:    "pending",
		CreatedAt: now,
		ExpiresAt: now.Add(as.timeout),
	}

	as.mu.Lock()
	as.challenges[ch.ID] = ch
	as.mu.Unlock()

	return ch
}

// Confirm approves or rejects a pending challenge. Returns an error if the
// challenge does not exist or is no longer in the pending state.
func (as *ApprovalService) Confirm(challengeID string, approved bool, decidedBy string) (*ApprovalChallenge, error) {
	as.mu.Lock()
	defer as.mu.Unlock()

	ch, ok := as.challenges[challengeID]
	if !ok {
		return nil, fmt.Errorf("challenge not found: %s", challengeID)
	}

	if ch.Status != "pending" {
		return nil, fmt.Errorf("challenge %s is already %s", challengeID, ch.Status)
	}

	// Check expiry before confirming.
	if time.Now().UTC().After(ch.ExpiresAt) {
		ch.Status = "expired"
		return nil, fmt.Errorf("challenge %s has expired", challengeID)
	}

	now := time.Now().UTC()
	ch.DecidedAt = &now
	ch.DecisionBy = decidedBy

	if approved {
		ch.Status = "approved"
	} else {
		ch.Status = "rejected"
	}

	return ch, nil
}

// GetChallenge returns the challenge with the given ID, or nil if not found.
func (as *ApprovalService) GetChallenge(challengeID string) *ApprovalChallenge {
	as.mu.RLock()
	defer as.mu.RUnlock()
	return as.challenges[challengeID]
}

// ListPending returns all challenges that are still in the pending state.
func (as *ApprovalService) ListPending() []*ApprovalChallenge {
	as.mu.RLock()
	defer as.mu.RUnlock()

	pending := make([]*ApprovalChallenge, 0)
	for _, ch := range as.challenges {
		if ch.Status == "pending" {
			pending = append(pending, ch)
		}
	}
	return pending
}

// CleanExpired transitions any pending challenges past their expiry time to the
// expired state. If an onExpire callback is set, it is invoked for each newly
// expired challenge (outside the write lock).
func (as *ApprovalService) CleanExpired() {
	now := time.Now().UTC()
	var expired []*ApprovalChallenge

	as.mu.Lock()
	for _, ch := range as.challenges {
		if ch.Status == "pending" && now.After(ch.ExpiresAt) {
			ch.Status = "expired"
			expired = append(expired, ch)
		}
	}
	as.mu.Unlock()

	if as.onExpire != nil {
		for _, ch := range expired {
			as.onExpire(ch)
		}
	}
}

// StartExpiryWatcher launches a background goroutine that periodically calls
// CleanExpired at the given interval. The goroutine runs until the process exits.
func (as *ApprovalService) StartExpiryWatcher(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			as.CleanExpired()
		}
	}()
}
