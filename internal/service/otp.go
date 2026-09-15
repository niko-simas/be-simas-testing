package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"simas-backend/internal/auth"
)

var (
	ErrOTPExpired     = errors.New("otp expired")
	ErrOTPInvalid     = errors.New("otp invalid")
	ErrOTPMaxAttempts = errors.New("otp max attempts reached")
	ErrOTPResendWait  = errors.New("otp resend cooldown active")
	ErrOTPNotFound    = errors.New("otp not found")
)

const (
	otpTTL         = 3 * time.Minute
	otpMaxAttempts = 5
	otpResendCooldown = 60 * time.Second
)

type otpEntry struct {
	Hash      string    `json:"hash"`
	Attempts  int       `json:"attempts"`
	CreatedAt time.Time `json:"created_at"`
}

type OTPService struct {
	redis *redis.Client
}

func NewOTPService(redisClient *redis.Client) *OTPService {
	return &OTPService{redis: redisClient}
}

func otpKey(email string) string {
	return "otp:mobile-register:" + email
}

func otpCooldownKey(email string) string {
	return "otp:cooldown:" + email
}

// Generate creates a new OTP, stores its hash in Redis, returns plain OTP
func (s *OTPService) Generate(ctx context.Context, email string) (string, error) {
	if s.redis == nil {
		return "", errors.New("redis unavailable")
	}

	// Check resend cooldown
	exists, _ := s.redis.Exists(ctx, otpCooldownKey(email)).Result()
	if exists > 0 {
		return "", ErrOTPResendWait
	}

	otp, err := auth.GenerateOTP()
	if err != nil {
		return "", err
	}

	hash := hashOTP(otp)
	entry := otpEntry{
		Hash:      hash,
		Attempts:  0,
		CreatedAt: time.Now(),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}

	pipe := s.redis.Pipeline()
	pipe.Set(ctx, otpKey(email), data, otpTTL)
	pipe.Set(ctx, otpCooldownKey(email), "1", otpResendCooldown)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}

	return otp, nil
}

// Verify checks OTP code, deletes on success, increments attempts on failure
func (s *OTPService) Verify(ctx context.Context, email, code string) error {
	if s.redis == nil {
		return errors.New("redis unavailable")
	}

	data, err := s.redis.Get(ctx, otpKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return ErrOTPExpired
	}
	if err != nil {
		return err
	}

	var entry otpEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		return ErrOTPNotFound
	}

	if entry.Attempts >= otpMaxAttempts {
		s.redis.Del(ctx, otpKey(email))
		return ErrOTPMaxAttempts
	}

	if !auth.ConstantTimeCompare(hashOTP(code), entry.Hash) {
		entry.Attempts++
		updated, _ := json.Marshal(entry)
		// Preserve remaining TTL
		ttl, _ := s.redis.TTL(ctx, otpKey(email)).Result()
		if ttl <= 0 {
			ttl = otpTTL
		}
		s.redis.Set(ctx, otpKey(email), updated, ttl)
		return ErrOTPInvalid
	}

	// Success — delete OTP and cooldown
	s.redis.Del(ctx, otpKey(email))
	s.redis.Del(ctx, otpCooldownKey(email))
	return nil
}

func hashOTP(otp string) string {
	h := sha256.Sum256([]byte(otp))
	return hex.EncodeToString(h[:])
}

// OAuth session keys
func oauthSessionKey(id string) string {
	return "oauth:mobile:pending:" + id
}

const oauthSessionTTL = 10 * time.Minute

// SavePendingOAuth stores pending OAuth session in Redis
func SavePendingOAuth(ctx context.Context, r *redis.Client, id string, data interface{}) error {
	if r == nil {
		return errors.New("redis unavailable")
	}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal pending oauth: %w", err)
	}
	return r.Set(ctx, oauthSessionKey(id), b, oauthSessionTTL).Err()
}

// GetPendingOAuth retrieves and deletes pending OAuth session (single-use)
func GetPendingOAuth(ctx context.Context, r *redis.Client, id string, dest interface{}) error {
	if r == nil {
		return errors.New("redis unavailable")
	}
	key := oauthSessionKey(id)
	data, err := r.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return ErrOTPNotFound
	}
	if err != nil {
		return err
	}
	// Delete immediately — single use
	r.Del(ctx, key)
	return json.Unmarshal([]byte(data), dest)
}
