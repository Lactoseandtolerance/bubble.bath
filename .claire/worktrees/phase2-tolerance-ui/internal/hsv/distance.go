package hsv

import "math"

// Distance computes Euclidean distance in normalized HSV space.
// Hue is circular (0–360), Saturation and Value are 0–100.
// All axes are normalized to 0–100 before distance calculation.
// Returns a value in [0, ~173] (sqrt(100² + 100² + 100²)).
func Distance(h1, s1, v1, h2, s2, v2 int) float64 {
	// Circular hue difference, normalized to 0–100
	hd := math.Abs(float64(h1 - h2))
	if hd > 180 {
		hd = 360 - hd
	}
	hd *= 100.0 / 180.0

	sd := float64(s1 - s2)
	vd := float64(v1 - v2)

	return math.Sqrt(hd*hd + sd*sd + vd*vd)
}
