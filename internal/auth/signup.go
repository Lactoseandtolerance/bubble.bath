package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	bbcrypto "github.com/Lactoseandtolerance/bubble-bath/internal/crypto"
	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
	"github.com/google/uuid"
)

var ErrDuplicateCredentials = errors.New("a user with this digit code and color already exists")

type SignupRequest struct {
	DigitCode   int    `json:"digit_code"`
	Hue         int    `json:"hue"`
	Saturation  int    `json:"saturation"`
	Value       int    `json:"value"`
	DisplayName string `json:"display_name"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Service struct {
	users            *store.UserStore
	tokenEnc         *bbcrypto.TokenEncryptor
	colEnc           *bbcrypto.ColumnEncryptor
	accessTTL        time.Duration
	refreshTTL       time.Duration
	baseTolerance    float64
	toleranceFloor   float64
	toleranceCeiling float64
}

func NewService(
	users *store.UserStore,
	tokenEnc *bbcrypto.TokenEncryptor,
	colEnc *bbcrypto.ColumnEncryptor,
	accessTTLMinutes int,
	refreshTTLDays int,
	baseTolerance float64,
	toleranceFloor float64,
	toleranceCeiling float64,
) *Service {
	return &Service{
		users:            users,
		tokenEnc:         tokenEnc,
		colEnc:           colEnc,
		accessTTL:        time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL:       time.Duration(refreshTTLDays) * 24 * time.Hour,
		baseTolerance:    baseTolerance,
		toleranceFloor:   toleranceFloor,
		toleranceCeiling: toleranceCeiling,
	}
}

func (s *Service) Signup(ctx context.Context, req SignupRequest) (*AuthResponse, error) {
	if err := validateCredentials(req.DigitCode, req.Hue, req.Saturation, req.Value); err != nil {
		return nil, err
	}

	existing, err := s.users.FindByDigitCode(ctx, req.DigitCode)
	if err != nil {
		return nil, fmt.Errorf("checking existing users: %w", err)
	}
	for _, row := range existing {
		h, _ := s.colEnc.DecryptInt(row.HueEncrypted)
		sat, _ := s.colEnc.DecryptInt(row.SatEncrypted)
		v, _ := s.colEnc.DecryptInt(row.ValEncrypted)
		if h == req.Hue && sat == req.Saturation && v == req.Value {
			return nil, ErrDuplicateCredentials
		}
	}

	colorHash, err := bbcrypto.HashColor(req.DigitCode, req.Hue, req.Saturation, req.Value)
	if err != nil {
		return nil, fmt.Errorf("hashing color: %w", err)
	}

	hueEnc, err := s.colEnc.EncryptInt(req.Hue)
	if err != nil {
		return nil, fmt.Errorf("encrypting hue: %w", err)
	}
	satEnc, err := s.colEnc.EncryptInt(req.Saturation)
	if err != nil {
		return nil, fmt.Errorf("encrypting saturation: %w", err)
	}
	valEnc, err := s.colEnc.EncryptInt(req.Value)
	if err != nil {
		return nil, fmt.Errorf("encrypting value: %w", err)
	}

	now := time.Now()
	user := &models.User{
		ID:          uuid.New(),
		DigitCode:   req.DigitCode,
		Hue:         req.Hue,
		Saturation:  req.Saturation,
		Value:       req.Value,
		ColorHash:   colorHash,
		DisplayName: req.DisplayName,
		CreatedAt:   now,
	}

	err = s.users.Insert(ctx, user, store.HSVEncrypted{
		Hue: hueEnc,
		Sat: satEnc,
		Val: valEnc,
	})
	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return s.issueTokens(user, now)
}

func (s *Service) issueTokens(user *models.User, now time.Time) (*AuthResponse, error) {
	accessPayload := models.TokenPayload{
		UserID:     user.ID,
		DigitCode:  user.DigitCode,
		Hue:        user.Hue,
		Saturation: user.Saturation,
		Value:      user.Value,
		IssuedAt:   now,
		ExpiresAt:  now.Add(s.accessTTL),
	}
	accessToken, err := s.tokenEnc.Encrypt(accessPayload)
	if err != nil {
		return nil, fmt.Errorf("encrypting access token: %w", err)
	}

	refreshPayload := models.TokenPayload{
		UserID:     user.ID,
		DigitCode:  user.DigitCode,
		Hue:        user.Hue,
		Saturation: user.Saturation,
		Value:      user.Value,
		IssuedAt:   now,
		ExpiresAt:  now.Add(s.refreshTTL),
	}
	refreshToken, err := s.tokenEnc.Encrypt(refreshPayload)
	if err != nil {
		return nil, fmt.Errorf("encrypting refresh token: %w", err)
	}

	return &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func validateCredentials(digitCode, h, s, v int) error {
	if digitCode < 0 || digitCode > 99 {
		return fmt.Errorf("digit_code must be 0-99, got %d", digitCode)
	}
	if h < 0 || h > 359 {
		return fmt.Errorf("hue must be 0-359, got %d", h)
	}
	if s < 0 || s > 100 {
		return fmt.Errorf("saturation must be 0-100, got %d", s)
	}
	if v < 0 || v > 100 {
		return fmt.Errorf("value must be 0-100, got %d", v)
	}
	return nil
}
