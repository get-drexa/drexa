package drexa

import (
	"fmt"
	"strings"
)

type ConfigError struct {
	Errors []error
}

func NewConfigError(errs ...error) *ConfigError {
	return &ConfigError{Errors: errs}
}

func (e *ConfigError) Error() string {
	sb := strings.Builder{}
	sb.WriteString("invalid config:\n")
	for _, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}

	return sb.String()
}
