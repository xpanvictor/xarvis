package utils

import "fmt"

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
