package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                      uuid.UUID `json:"-"`
	DigitCode               int       `json:"-"`
	Hue                     int       `json:"-"`
	Saturation              int       `json:"-"`
	Value                   int       `json:"-"`
	ColorHash               []byte    `json:"-"`
	DisplayName             string    `json:"display_name"`
	AvatarShape             string    `json:"avatar_shape"`
	RecoveryValidatorSecret []byte    `json:"-"`
	CreatedAt               time.Time `json:"created_at"`
}
