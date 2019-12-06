package main

import (
	"os"
)

// GetEnv retrieves an environment variable with a default value
// if the environment variable is missing
func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

// Min returns the smaller of x or y.
func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}
