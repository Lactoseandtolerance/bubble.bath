package store

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/Lactoseandtolerance/bubble-bath/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://bubblebath@localhost:5432/bubblebath?sslmode=disable"
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("skipping DB test: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM users WHERE display_name LIKE 'test_%'")
		pool.Close()
	})
	return pool
}

func TestInsertAndFindUser(t *testing.T) {
	pool := testPool(t)
	us := NewUserStore(pool)
	ctx := context.Background()

	user := &models.User{
		ID:          uuid.New(),
		DigitCode:   42,
		Hue:         180,
		Saturation:  75,
		Value:       50,
		ColorHash:   []byte("fakehash"),
		DisplayName: "test_user_1",
	}
	hsvEncrypted := HSVEncrypted{
		Hue: []byte("enchue"),
		Sat: []byte("encsat"),
		Val: []byte("encval"),
	}

	err := us.Insert(ctx, user, hsvEncrypted)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	found, err := us.FindByDigitCode(ctx, 42)
	if err != nil {
		t.Fatalf("FindByDigitCode failed: %v", err)
	}

	if len(found) == 0 {
		t.Fatal("FindByDigitCode returned no results")
	}

	match := false
	for _, row := range found {
		if row.ID == user.ID {
			match = true
			if row.DisplayName != "test_user_1" {
				t.Errorf("DisplayName = %q, want %q", row.DisplayName, "test_user_1")
			}
		}
	}
	if !match {
		t.Error("inserted user not found in results")
	}
}

func TestFindByID(t *testing.T) {
	pool := testPool(t)
	us := NewUserStore(pool)
	ctx := context.Background()

	id := uuid.New()
	user := &models.User{
		ID:          id,
		DigitCode:   99,
		Hue:         359,
		Saturation:  100,
		Value:       100,
		ColorHash:   []byte("fakehash2"),
		DisplayName: "test_user_2",
	}
	hsvEncrypted := HSVEncrypted{
		Hue: []byte("enchue2"),
		Sat: []byte("encsat2"),
		Val: []byte("encval2"),
	}

	us.Insert(ctx, user, hsvEncrypted)

	found, err := us.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if found == nil {
		t.Fatal("FindByID returned nil")
	}
	if found.DisplayName != "test_user_2" {
		t.Errorf("DisplayName = %q, want %q", found.DisplayName, "test_user_2")
	}
}

func TestUpdateDisplayName(t *testing.T) {
	pool := testPool(t)
	us := NewUserStore(pool)
	ctx := context.Background()

	// Insert two test users using the store's Insert method
	user1 := &models.User{
		ID:          uuid.New(),
		DigitCode:   10,
		Hue:         100,
		Saturation:  50,
		Value:       50,
		ColorHash:   []byte("fakehash_tag1"),
		DisplayName: "test_tag_setup1",
	}
	user2 := &models.User{
		ID:          uuid.New(),
		DigitCode:   11,
		Hue:         200,
		Saturation:  60,
		Value:       60,
		ColorHash:   []byte("fakehash_tag2"),
		DisplayName: "test_tag_setup2",
	}
	us.Insert(ctx, user1, HSVEncrypted{Hue: []byte("h1"), Sat: []byte("s1"), Val: []byte("v1")})
	us.Insert(ctx, user2, HSVEncrypted{Hue: []byte("h2"), Sat: []byte("s2"), Val: []byte("v2")})

	// Test: update display name
	err := us.UpdateDisplayName(ctx, user1.ID, "test_tag_alpha")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify it was updated
	row, err := us.FindByID(ctx, user1.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if row.DisplayName != "test_tag_alpha" {
		t.Errorf("expected 'test_tag_alpha', got %q", row.DisplayName)
	}

	// Test: duplicate display name returns ErrDuplicateDisplayName
	err = us.UpdateDisplayName(ctx, user2.ID, "test_tag_alpha")
	if !errors.Is(err, ErrDuplicateDisplayName) {
		t.Errorf("expected ErrDuplicateDisplayName, got %v", err)
	}

	// Test: clearing display name (empty string) works
	err = us.UpdateDisplayName(ctx, user1.ID, "")
	if err != nil {
		t.Fatalf("expected no error clearing name, got %v", err)
	}

	// Test: two users can both have empty display names
	err = us.UpdateDisplayName(ctx, user2.ID, "")
	if err != nil {
		t.Fatalf("expected no error for second empty name, got %v", err)
	}
}
