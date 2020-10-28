package helpers

import "time"

// Min returns the smaller of x or y.
func MinDuration(x, y time.Duration) time.Duration {
	if x > y {
		return y
	}
	return x
}
