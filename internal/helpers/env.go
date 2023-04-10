package helpers

import "os"

// GetEnv retrieves list of possible environment variables with a default value
// if the environment variable is missing
func GetEnv(keysAndFallback ...string) string {
	last := len(keysAndFallback) - 1
	for _, key := range keysAndFallback[:last] {
		if value, ok := os.LookupEnv(key); ok {
			return value
		}
	}
	return keysAndFallback[last]
}
