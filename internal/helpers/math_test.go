package helpers

import (
	"testing"
)

func TestMin(t *testing.T) {
	var x, y int = 5, 3
	min := Min(x, y)
	if min != y {
		t.Errorf("Min returned unexpected value: got %v want %v",
			min, y)
	}
}
