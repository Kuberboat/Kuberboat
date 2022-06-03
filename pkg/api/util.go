package api

import (
	"crypto/sha256"
	"fmt"
)

// Hash provides a standard way to obtain the hash from any struct.
func Hash(o interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", o)))

	return fmt.Sprintf("%x", h.Sum(nil))
}

func Min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func Max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func IsSubset(inner, outer *map[string]string) bool {
	for k, v := range *inner {
		outerV, ok := (*outer)[k]
		if !ok || outerV != v {
			return false
		}
	}
	return true
}
