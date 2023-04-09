package helpers

import "os"

// GetEnv retrieves list of possible environment variables with a default value
// if the environment variable is missing
func GetEnv(keys_and_fallback ...string) string {
	last := len(keys_and_fallback) - 1
	for _, key := range keys_and_fallback[:last] {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
	}
	return keys_and_fallback[last]
}
