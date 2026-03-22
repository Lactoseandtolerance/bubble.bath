package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	bbcrypto "github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type LoginDirectRequest struct {
	DigitCode  int `json:"digit_code"`
	Hue        int `json:"hue"`
	Saturation int `json:"saturation"`
	Value      int `json:"value"`
}

func (s *Service) LoginDirect(ctx context.Context, req LoginDirectRequest) (*AuthResponse, error) {
	if err := validateCredentials(req.DigitCode, req.Hue, req.Saturation, req.Value); err != nil {
		return nil, err
	}

	candidates, err := s.users.FindByDigitCode(ctx, req.DigitCode)
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}

	for _, row := range candidates {
		ok, err := bbcrypto.VerifyColor(req.DigitCode, req.Hue, req.Saturation, req.Value, row.ColorHash)
		if err != nil {
			continue
		}
		if ok {
			user := &models.User{
				ID:         row.ID,
				DigitCode:  row.DigitCode,
				Hue:        req.Hue,
				Saturation: req.Saturation,
				Value:      req.Value,
			}
			return s.issueTokens(user, time.Now())
		}
	}

	return nil, ErrInvalidCredentials
}
