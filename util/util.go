package util

import (
	"math"

	"golang.org/x/exp/constraints"
)

// Clamp returns a value that does not exceed the specified range [min, max]
func Clamp[T1, T2, T3 constraints.Float | constraints.Integer](v T1, min T2, max T3) T1 {
	v = T1(math.Max(float64(v), float64(min)))
	v = T1(math.Min(float64(v), float64(max)))
	return v
}

// Between returns if a value is between the specified range [min, max]
func Between[T1, T2, T3 constraints.Float | constraints.Integer](v T1, min T2, max T3) bool {
	return float64(min) <= float64(v) && float64(v) <= float64(max)
}
