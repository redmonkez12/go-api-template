package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                      uuid.UUID  `json:"id"`
	Email                   string     `json:"email"`
	PasswordHash            string     `json:"-"` // Never expose password hash in JSON
	EmailVerified           bool       `json:"email_verified"`
	EmailVerificationToken  *string    `json:"-"`
	EmailVerificationSentAt *time.Time `json:"-"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}
