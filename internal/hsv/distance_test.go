package hsv

import (
	"math"
	"testing"
)

func TestDistanceIdentical(t *testing.T) {
	d := Distance(180, 50, 50, 180, 50, 50)
	if d != 0 {
		t.Errorf("identical colors: got %f, want 0", d)
	}
}

func TestDistanceCircularHue(t *testing.T) {
	// 5→355 wraps to 10 degrees, 5→15 is also 10 degrees — should be equal
	d1 := Distance(5, 50, 50, 355, 50, 50)
	d2 := Distance(5, 50, 50, 15, 50, 50)
	if math.Abs(d1-d2) > 0.001 {
		t.Errorf("circular hue: d(5,355)=%f != d(5,15)=%f", d1, d2)
	}
}

func TestDistanceMaximum(t *testing.T) {
	// Opposite hue (180 apart) + opposite S and V → max distance ≈ 173
	d := Distance(0, 0, 0, 180, 100, 100)
	expected := math.Sqrt(100*100 + 100*100 + 100*100)
	if math.Abs(d-expected) > 0.001 {
		t.Errorf("max distance: got %f, want %f", d, expected)
	}
}

func TestDistanceSaturationOnly(t *testing.T) {
	d := Distance(0, 0, 50, 0, 100, 50)
	if math.Abs(d-100) > 0.001 {
		t.Errorf("saturation only: got %f, want 100", d)
	}
}

func TestDistanceValueOnly(t *testing.T) {
	d := Distance(0, 50, 0, 0, 50, 100)
	if math.Abs(d-100) > 0.001 {
		t.Errorf("value only: got %f, want 100", d)
	}
}

func TestDistanceHueOnly(t *testing.T) {
	// 90 degrees apart → 90*(100/180) = 50 normalized
	d := Distance(0, 50, 50, 90, 50, 50)
	expected := 90.0 * 100.0 / 180.0
	if math.Abs(d-expected) > 0.001 {
		t.Errorf("hue only: got %f, want %f", d, expected)
	}
}
