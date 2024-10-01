package main

import (
	"fmt"
	"github.com/coreos/go-oidc"
	"log"
	"os"
	"strconv"
	"strings"
)

// getEnv retrieves the value of the environment variable named by the key.
// If the variable is present in the environment, the value (which may be empty) is returned.
// Otherwise, the defaultValue is returned. The function supports int and string types.
// For any other type, the function will log an info message and return the defaultValue.
func getEnv[T any](key string, defaultValue T) T {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	var result T
	switch any(result).(type) {
	case int:
		v, err := strconv.Atoi(value)
		if err != nil {
			log.Fatalf("Failed to convert %s to int: %v", key, err)
		}
		return any(v).(T)
	case string:
		return any(value).(T)
	default:
		log.Printf("Unsupported type for environment variable %s, using default", key)
		return defaultValue
	}
	return result
}

// formatUserInfo formats the user information from an oidc.UserInfo object into a string.
// It retrieves the claims from the UserInfo object, aligns the keys, and formats the values.
// The formatted string is returned.
func formatUserInfo(i *oidc.UserInfo) string {
	var m map[string]any
	err := i.Claims(&m)
	if err != nil {
		return err.Error()
	}
	var sb strings.Builder

	// Find the maximum key length
	maxKeyLen := 0
	for k := range m {
		if len(k) > maxKeyLen {
			maxKeyLen = len(k)
		}
	}

	// Format the output with aligned colons
	for k, v := range m {
		sb.WriteString(k)
		sb.WriteString(strings.Repeat(" ", maxKeyLen-len(k)))
		sb.WriteString(": ")
		if str, ok := v.(string); ok {
			sb.WriteString(str)
		} else {
			sb.WriteString(fmt.Sprintf("%v", v))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
