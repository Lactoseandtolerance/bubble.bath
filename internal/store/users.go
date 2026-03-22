package store

import (
	"context"
	"fmt"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HSVEncrypted struct {
	Hue []byte
	Sat []byte
	Val []byte
}

type UserRow struct {
	models.User
	HueEncrypted []byte
	SatEncrypted []byte
	ValEncrypted []byte
}

type UserStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) *UserStore {
	return &UserStore{pool: pool}
}

func (us *UserStore) Insert(ctx context.Context, user *models.User, hsv HSVEncrypted) error {
	_, err := us.pool.Exec(ctx, `
		INSERT INTO users (id, digit_code, hue_encrypted, sat_encrypted, val_encrypted, color_hash, display_name, avatar_shape, recovery_secret, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, user.ID, user.DigitCode, hsv.Hue, hsv.Sat, hsv.Val, user.ColorHash, user.DisplayName, user.AvatarShape, user.RecoveryValidatorSecret, user.CreatedAt)
	if err != nil {
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (us *UserStore) FindByDigitCode(ctx context.Context, digitCode int) ([]UserRow, error) {
	rows, err := us.pool.Query(ctx, `
		SELECT id, digit_code, hue_encrypted, sat_encrypted, val_encrypted, color_hash, display_name, avatar_shape, created_at
		FROM users
		WHERE digit_code = $1
	`, digitCode)
	if err != nil {
		return nil, fmt.Errorf("querying users by digit_code: %w", err)
	}
	defer rows.Close()

	var result []UserRow
	for rows.Next() {
		var row UserRow
		err := rows.Scan(
			&row.ID, &row.DigitCode,
			&row.HueEncrypted, &row.SatEncrypted, &row.ValEncrypted,
			&row.ColorHash, &row.DisplayName, &row.AvatarShape, &row.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning user row: %w", err)
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func (us *UserStore) FindByID(ctx context.Context, id uuid.UUID) (*UserRow, error) {
	var row UserRow
	err := us.pool.QueryRow(ctx, `
		SELECT id, digit_code, hue_encrypted, sat_encrypted, val_encrypted, color_hash, display_name, avatar_shape, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(
		&row.ID, &row.DigitCode,
		&row.HueEncrypted, &row.SatEncrypted, &row.ValEncrypted,
		&row.ColorHash, &row.DisplayName, &row.AvatarShape, &row.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return &row, nil
}

func (us *UserStore) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := us.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
