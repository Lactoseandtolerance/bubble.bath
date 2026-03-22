package models

import (
	"time"

	"github.com/google/uuid"
)

type TokenPayload struct {
	UserID     uuid.UUID `json:"user_id"`
	DigitCode  int       `json:"digit_code"`
	Hue        int       `json:"hue"`
	Saturation int       `json:"saturation"`
	Value      int       `json:"value"`
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
