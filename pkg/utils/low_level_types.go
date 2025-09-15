package utils

import (
	"fmt"
	"strings"
)

type Range[T any] struct {
	Min *T
	Max *T
}

type XError struct {
	Reason string
	Meta   any
}

func (xe XError) ToError() error {
	return fmt.Errorf("xerror: %v\nmeta: %v", xe.Reason, xe.Meta)
}

func CheckContainsSubStrings(statement string, subs []string) bool {
	for _, word := range subs {
		if strings.Contains(strings.ToLower(statement), strings.ToLower(word)) {
			return true
		}
	}
	return false
}
