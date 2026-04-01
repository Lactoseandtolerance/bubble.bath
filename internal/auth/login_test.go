package auth

import (
	"context"
	"testing"

	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
)

func TestLoginDirectSuccess(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	req := SignupRequest{
		DigitCode:   55,
		Hue:         120,
		Saturation:  90,
		Value:       80,
		DisplayName: "test_login_1",
	}
	_, err := svc.Signup(context.Background(), req)
	if err != nil {
		t.Fatalf("Signup failed: %v", err)
	}

	loginReq := LoginDirectRequest{
		DigitCode:  55,
		Hue:        120,
		Saturation: 90,
		Value:      80,
	}
	resp, err := svc.LoginDirect(context.Background(), loginReq)
	if err != nil {
		t.Fatalf("LoginDirect failed: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginDirectWrongColor(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	req := SignupRequest{
		DigitCode:   56,
		Hue:         100,
		Saturation:  50,
		Value:       50,
		DisplayName: "test_login_2",
	}
	svc.Signup(context.Background(), req)

	loginReq := LoginDirectRequest{
		DigitCode:  56,
		Hue:        101,
		Saturation: 50,
		Value:      50,
	}
	_, err := svc.LoginDirect(context.Background(), loginReq)
	if err == nil {
		t.Error("expected error for wrong color")
	}
}

func TestLoginDirectWrongDigitCode(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	req := SignupRequest{
		DigitCode:   57,
		Hue:         200,
		Saturation:  60,
		Value:       70,
		DisplayName: "test_login_3",
	}
	svc.Signup(context.Background(), req)

	loginReq := LoginDirectRequest{
		DigitCode:  58,
		Hue:        200,
		Saturation: 60,
		Value:      70,
	}
	_, err := svc.LoginDirect(context.Background(), loginReq)
	if err == nil {
		t.Error("expected error for wrong digit code")
	}
}
