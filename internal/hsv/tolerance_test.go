package hsv

import "testing"

func TestClampTolerance(t *testing.T) {
	tests := []struct {
		name                     string
		base, floor, ceiling     float64
		want                     float64
	}{
		{"within bounds", 15, 5, 25, 15},
		{"below floor", 2, 5, 25, 5},
		{"above ceiling", 30, 5, 25, 25},
		{"at floor", 5, 5, 25, 5},
		{"at ceiling", 25, 5, 25, 25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClampTolerance(tt.base, tt.floor, tt.ceiling)
			if got != tt.want {
				t.Errorf("ClampTolerance(%v,%v,%v) = %v, want %v",
					tt.base, tt.floor, tt.ceiling, got, tt.want)
			}
		})
	}
}

func TestFindNearestExactMatch(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 120, Saturation: 80, Value: 60},
	}
	m, err := FindNearest(candidates, 120, 80, 60, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 0 || m.Distance != 0 {
		t.Errorf("got index=%d dist=%f, want index=0 dist=0", m.Index, m.Distance)
	}
}

func TestFindNearestWithinTolerance(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 120, Saturation: 80, Value: 60},
	}
	m, err := FindNearest(candidates, 123, 82, 58, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 0 {
		t.Errorf("got index=%d, want 0", m.Index)
	}
	if m.Distance == 0 {
		t.Error("distance should be non-zero for imprecise match")
	}
}

func TestFindNearestOutsideTolerance(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 120, Saturation: 80, Value: 60},
	}
	_, err := FindNearest(candidates, 200, 20, 20, 15)
	if err != ErrNoMatch {
		t.Errorf("got err=%v, want ErrNoMatch", err)
	}
}

func TestFindNearestPicksClosest(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 100, Saturation: 50, Value: 50},
		{Index: 1, Hue: 120, Saturation: 50, Value: 50},
		{Index: 2, Hue: 140, Saturation: 50, Value: 50},
	}
	// H=118 is closest to H=120 (index 1)
	m, err := FindNearest(candidates, 118, 50, 50, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 1 {
		t.Errorf("got index=%d, want 1 (closest)", m.Index)
	}
}

func TestFindNearestEmpty(t *testing.T) {
	_, err := FindNearest(nil, 120, 80, 60, 15)
	if err != ErrNoMatch {
		t.Errorf("got err=%v, want ErrNoMatch", err)
	}
}

func TestFindNearestCircularHue(t *testing.T) {
	candidates := []Candidate{
		{Index: 0, Hue: 350, Saturation: 50, Value: 50},
		{Index: 1, Hue: 180, Saturation: 50, Value: 50},
	}
	// H=5 wraps around — should be closest to H=350 (15 degrees away)
	m, err := FindNearest(candidates, 5, 50, 50, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Index != 0 {
		t.Errorf("got index=%d, want 0 (hue 350 is closest to 5 via wrap)", m.Index)
	}
}
