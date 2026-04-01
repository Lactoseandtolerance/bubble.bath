package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Lactoseandtolerance/bubble-bath/internal/hsv"
	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
)

// LoginPickerRequest is the input for tolerance-based color picker login.
type LoginPickerRequest struct {
	DigitCode  int `json:"digit_code"`
	Hue        int `json:"hue"`
	Saturation int `json:"saturation"`
	Value      int `json:"value"`
}

// LoginPicker authenticates via nearest-neighbor matching in HSV space.
func (s *Service) LoginPicker(ctx context.Context, req LoginPickerRequest) (*AuthResponse, error) {
	if err := validateCredentials(req.DigitCode, req.Hue, req.Saturation, req.Value); err != nil {
		return nil, err
	}

	rows, err := s.users.FindByDigitCode(ctx, req.DigitCode)
	if err != nil {
		return nil, fmt.Errorf("finding users: %w", err)
	}
	if len(rows) == 0 {
		return nil, ErrInvalidCredentials
	}

	// Decrypt stored HSV to build candidate list
	candidates := make([]hsv.Candidate, 0, len(rows))
	for i, row := range rows {
		h, err := s.colEnc.DecryptInt(row.HueEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting hue: %w", err)
		}
		sat, err := s.colEnc.DecryptInt(row.SatEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting saturation: %w", err)
		}
		v, err := s.colEnc.DecryptInt(row.ValEncrypted)
		if err != nil {
			return nil, fmt.Errorf("decrypting value: %w", err)
		}
		candidates = append(candidates, hsv.Candidate{
			Index:      i,
			Hue:        h,
			Saturation: sat,
			Value:      v,
		})
	}

	tol := hsv.ClampTolerance(s.baseTolerance, s.toleranceFloor, s.toleranceCeiling)
	match, err := hsv.FindNearest(candidates, req.Hue, req.Saturation, req.Value, tol)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	matched := rows[match.Index]
	user := &models.User{
		ID:         matched.ID,
		DigitCode:  matched.DigitCode,
		Hue:        candidates[match.Index].Hue,
		Saturation: candidates[match.Index].Saturation,
		Value:      candidates[match.Index].Value,
	}
	return s.issueTokens(user, time.Now())
}
