package helpers

import (
	"testing"
	"time"
)

func TestMin(t *testing.T) {
	var x, y time.Duration = 5 * time.Second, 3 * time.Second
	min := MinDuration(x, y)
	if min != y {
		t.Errorf("Min returned unexpected value: got %v want %v", min, y)
	}
}

func TestMinSwapped(t *testing.T) {
	var x, y time.Duration = 3 * time.Second, 5 * time.Second
	min := MinDuration(x, y)
	if min != x {
		t.Errorf("Min returned unexpected value: got %v want %v", min, x)
	}
}
