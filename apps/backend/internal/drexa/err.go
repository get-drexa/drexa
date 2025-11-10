package drexa

import (
	"fmt"
	"strings"
)

type ServerConfigError struct {
	Errors []error
}

func NewServerConfigError(errs ...error) *ServerConfigError {
	return &ServerConfigError{Errors: errs}
}

func (e *ServerConfigError) Error() string {
	sb := strings.Builder{}
	sb.WriteString("invalid server config:\n")
	for _, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return sb.String()
}
