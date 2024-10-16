package utils

import (
	"fmt"
	"runtime"
)

// ErrorWithLocation wraps an error with file and line information
func ErrorWithLocation(err error) error {
	if err == nil {
		return nil
	}

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("error occurred: %v", err)
	}

	return fmt.Errorf("%s:%d: %v", file, line, err)
}
