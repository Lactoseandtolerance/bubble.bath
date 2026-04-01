package hsv

import "errors"

// ErrNoMatch is returned when no candidate is within the tolerance radius.
var ErrNoMatch = errors.New("no match within tolerance")

// Candidate represents a stored user's decrypted HSV values.
type Candidate struct {
	Index      int
	Hue        int
	Saturation int
	Value      int
}

// MatchResult holds the index of the matched candidate and the distance.
type MatchResult struct {
	Index    int
	Distance float64
}

// ClampTolerance clamps a base tolerance to [floor, ceiling].
func ClampTolerance(base, floor, ceiling float64) float64 {
	if base < floor {
		return floor
	}
	if base > ceiling {
		return ceiling
	}
	return base
}

// FindNearest finds the closest candidate to the submitted HSV.
// Returns the nearest candidate if within tolerance, else ErrNoMatch.
func FindNearest(candidates []Candidate, h, s, v int, tolerance float64) (*MatchResult, error) {
	if len(candidates) == 0 {
		return nil, ErrNoMatch
	}

	var best *MatchResult
	for _, c := range candidates {
		d := Distance(c.Hue, c.Saturation, c.Value, h, s, v)
		if best == nil || d < best.Distance {
			best = &MatchResult{Index: c.Index, Distance: d}
		}
	}

	if best.Distance > tolerance {
		return nil, ErrNoMatch
	}
	return best, nil
}
