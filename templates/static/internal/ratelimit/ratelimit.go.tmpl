package ratelimit

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	emailCooldownDuration = 2 * time.Minute
	ipRateLimitWindow     = 15 * time.Minute
	ipRateLimitMax        = 10
)

// Limiter handles rate limiting for authentication endpoints
type Limiter struct {
	client *redis.Client
}

// NewLimiter creates a new rate limiter instance
func NewLimiter(client *redis.Client) *Limiter {
	return &Limiter{
		client: client,
	}
}

// CheckEmailCooldown returns true if the email is on cooldown (should reject request)
func (l *Limiter) CheckEmailCooldown(ctx context.Context, email string) (bool, error) {
	key := emailCooldownKey(email)
	exists, err := l.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check email cooldown: %w", err)
	}
	return exists > 0, nil
}

// SetEmailCooldown sets a 2-minute cooldown for the given email
func (l *Limiter) SetEmailCooldown(ctx context.Context, email string) error {
	key := emailCooldownKey(email)
	err := l.client.Set(ctx, key, "1", emailCooldownDuration).Err()
	if err != nil {
		return fmt.Errorf("failed to set email cooldown: %w", err)
	}
	return nil
}

// CheckIPRateLimit returns true if the IP has exceeded rate limit (10 req/15 min)
func (l *Limiter) CheckIPRateLimit(ctx context.Context, ip string) (bool, error) {
	return l.CheckIPRateLimitWithPurpose(ctx, ip, "auth")
}

// CheckIPRateLimitWithPurpose returns true if the IP has exceeded rate limit for a specific purpose
func (l *Limiter) CheckIPRateLimitWithPurpose(ctx context.Context, ip string, purpose string) (bool, error) {
	key := ipRateLimitKeyWithPurpose(ip, purpose)
	now := time.Now().Unix()
	windowStart := now - int64(ipRateLimitWindow.Seconds())

	// Remove expired entries
	err := l.client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart)).Err()
	if err != nil {
		return false, fmt.Errorf("failed to clean up expired entries: %w", err)
	}

	// Count requests in current window
	count, err := l.client.ZCard(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to count requests: %w", err)
	}

	return count >= ipRateLimitMax, nil
}

// RecordIPRequest records a request for the given IP address
func (l *Limiter) RecordIPRequest(ctx context.Context, ip string) error {
	return l.RecordIPRequestWithPurpose(ctx, ip, "auth")
}

// RecordIPRequestWithPurpose records a request for the given IP address with a specific purpose
func (l *Limiter) RecordIPRequestWithPurpose(ctx context.Context, ip string, purpose string) error {
	key := ipRateLimitKeyWithPurpose(ip, purpose)
	now := time.Now().Unix()

	// Add current request with timestamp as score and value
	member := fmt.Sprintf("%d", now)
	err := l.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: member,
	}).Err()
	if err != nil {
		return fmt.Errorf("failed to record IP request: %w", err)
	}

	// Set expiry on the key to clean up old data
	err = l.client.Expire(ctx, key, ipRateLimitWindow).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiry on rate limit key: %w", err)
	}

	return nil
}

// emailCooldownKey generates a Redis key for email cooldown
func emailCooldownKey(email string) string {
	hash := sha256.Sum256([]byte(email))
	return fmt.Sprintf("ratelimit:email:%x", hash)
}

// ipRateLimitKeyWithPurpose generates a Redis key for IP rate limiting with a specific purpose
func ipRateLimitKeyWithPurpose(ip string, purpose string) string {
	return fmt.Sprintf("ratelimit:ip:%s:%s", ip, purpose)
}
