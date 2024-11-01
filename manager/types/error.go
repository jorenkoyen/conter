package types

import (
	"fmt"
	"strings"
)

type ValidationError struct {
	Reasons map[string]string
}

// Append will append a new reason to the validation error structure.
func (v *ValidationError) Append(field string, reason string) {
	if v.Reasons == nil {
		v.Reasons = make(map[string]string)
	}

	v.Reasons[field] = reason
}

// Appendf will append a formatted reason to the validation error structure.
func (v *ValidationError) Appendf(field string, format string, args ...interface{}) {
	formatted := fmt.Sprintf(format, args...)
	v.Append(field, formatted)
}

// HasFailures will return true if the validation error contains any reasons for failure.
func (v *ValidationError) HasFailures() bool {
	return len(v.Reasons) > 0
}

func (v *ValidationError) Error() string {
	if !v.HasFailures() {
		return ""
	}

	messages := make([]string, 0, len(v.Reasons))
	for field, reason := range v.Reasons {
		msg := fmt.Sprintf("%s: %s", field, reason)
		messages = append(messages, msg)
	}

	return strings.Join(messages, ", ")
}
