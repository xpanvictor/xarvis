package dbtypes

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type XVector []float32

// To DB: convert to "[0.12,0.34,...]"
func (v XVector) Value() (driver.Value, error) {
	if len(v) == 0 {
		return "[]", nil
	}
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%f", f)
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ",")), nil
}

// From DB: parse "[0.12,0.34,...]"
func (v *XVector) Scan(value interface{}) error {
	if value == nil {
		*v = nil
		return nil
	}
	switch data := value.(type) {
	case string:
		return v.parse(data)
	case []byte:
		return v.parse(string(data))
	default:
		return fmt.Errorf("unsupported type for XVector: %T", value)
	}
}

func (v *XVector) parse(s string) error {
	s = strings.Trim(s, "[] ")
	if s == "" {
		*v = []float32{}
		return nil
	}
	parts := strings.Split(s, ",")
	vec := make([]float32, len(parts))
	for i, p := range parts {
		var f float32
		_, err := fmt.Sscanf(strings.TrimSpace(p), "%f", &f)
		if err != nil {
			return err
		}
		vec[i] = f
	}
	*v = vec
	return nil
}
