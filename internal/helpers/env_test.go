package helpers

import (
	"testing"
)

func TestGetEnv(t *testing.T) {
	expected := "120"
	t.Setenv("MAX_DELAY", expected)
	value := GetEnv("MAX_DELAY", "12")
	if value != expected {
		t.Errorf("GetEnv returned unexpected value: got %v want %v", value, expected)
	}
}

func TestGetEnvDefault(t *testing.T) {
	defaultValue := "42"
	value := GetEnv("MISSING", defaultValue)
	if value != defaultValue {
		t.Errorf("GetEnv returned unexpected value: got %v want %v",
			value, defaultValue)
	}
}

func TestGetEnvMany(t *testing.T) {
	expected := "35"
	t.Setenv("SECOND", expected)
	value := GetEnv("MISSING", "SECOND", "")
	if value != expected {
		t.Errorf("GetEnv returned unexpected value: got %v want %v", value, expected)
	}
}

func TestGetEnvManyDefault(t *testing.T) {
	defaultValue := "42"
	value := GetEnv("MISSING", "SECOND", defaultValue)
	if value != defaultValue {
		t.Errorf("GetEnv returned unexpected value: got %v want %v", value, defaultValue)
	}
}
