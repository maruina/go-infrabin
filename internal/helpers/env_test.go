package helpers

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	expected := "120"
	os.Setenv("MAX_DELAY", expected)
	value := GetEnv("MAX_DELAY", "12")
	if value != expected {
		t.Errorf("GetEnv returned unexpected value: got %v want %v",
			value, expected)
	}

	defaultValue := "42"
	value = GetEnv("MISSING", defaultValue)
	if value != defaultValue {
		t.Errorf("GetEnv returned unexpected value: got %v want %v",
			value, defaultValue)
	}
}
