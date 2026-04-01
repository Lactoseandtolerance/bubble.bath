package auth

import (
	"context"
	"testing"

	"github.com/Lactoseandtolerance/bubble-bath/internal/store"
)

func TestLoginPickerExactMatch(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 30, Hue: 180, Saturation: 50, Value: 80, DisplayName: "test_picker_1",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	resp, err := svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 30, Hue: 180, Saturation: 50, Value: 80,
	})
	if err != nil {
		t.Fatalf("LoginPicker exact: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginPickerWithinTolerance(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 31, Hue: 200, Saturation: 60, Value: 70, DisplayName: "test_picker_2",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	resp, err := svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 31, Hue: 203, Saturation: 62, Value: 68,
	})
	if err != nil {
		t.Fatalf("LoginPicker within tolerance: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginPickerOutsideTolerance(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 32, Hue: 100, Saturation: 50, Value: 50, DisplayName: "test_picker_3",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	_, err = svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 32, Hue: 300, Saturation: 10, Value: 90,
	})
	if err == nil {
		t.Error("expected error for color outside tolerance")
	}
}

func TestLoginPickerNearestNeighbor(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 25.0, 5.0, 30.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 33, Hue: 100, Saturation: 50, Value: 50, DisplayName: "test_picker_nn1",
	})
	if err != nil {
		t.Fatalf("Signup user1: %v", err)
	}
	_, err = svc.Signup(context.Background(), SignupRequest{
		DigitCode: 33, Hue: 200, Saturation: 50, Value: 50, DisplayName: "test_picker_nn2",
	})
	if err != nil {
		t.Fatalf("Signup user2: %v", err)
	}

	resp, err := svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 33, Hue: 103, Saturation: 50, Value: 50,
	})
	if err != nil {
		t.Fatalf("LoginPicker nearest: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
}

func TestLoginPickerWrongDigitCode(t *testing.T) {
	pool, tokenEnc, colEnc := testDeps(t)
	us := store.NewUserStore(pool)
	svc := NewService(us, tokenEnc, colEnc, 60, 30, 15.0, 5.0, 25.0)

	_, err := svc.Signup(context.Background(), SignupRequest{
		DigitCode: 34, Hue: 180, Saturation: 50, Value: 80, DisplayName: "test_picker_dc",
	})
	if err != nil {
		t.Fatalf("Signup: %v", err)
	}

	_, err = svc.LoginPicker(context.Background(), LoginPickerRequest{
		DigitCode: 35, Hue: 180, Saturation: 50, Value: 80,
	})
	if err == nil {
		t.Error("expected error for wrong digit code")
	}
}
