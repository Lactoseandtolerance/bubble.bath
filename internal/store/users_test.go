package store

import (
	"context"
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
		Hue:         360,
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
