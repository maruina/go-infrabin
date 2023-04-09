package helpers

import (
	"os"
	"testing"
)

func TestGetEnv(t *testing.T) {
	expected := "120"
	os.Setenv("MAX_DELAY", expected)
	defer os.Unsetenv("MAX_DELAY")
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
	os.Setenv("SECOND", expected)
	defer os.Unsetenv("SECOND")
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
