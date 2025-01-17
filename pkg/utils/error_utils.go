package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ErrorDetailLevel represents the level of error detail to display
type ErrorDetailLevel int

const (
	// ErrorDetailNone suppresses all additional error information
	ErrorDetailNone ErrorDetailLevel = iota
	// ErrorDetailSimple shows basic file, line and function information (default)
	ErrorDetailSimple
	// ErrorDetailFull shows complete error information including stack traces
	ErrorDetailFull
)

// getErrorDetailLevel returns the configured error detail level from environment
func getErrorDetailLevel() ErrorDetailLevel {
	level := strings.ToLower(os.Getenv("ERROR_DETAIL_LEVEL"))
	switch level {
	case "none":
		return ErrorDetailNone
	case "full":
		return ErrorDetailFull
	default:
		return ErrorDetailSimple // Default to simple error output
	}
}

// formatError formats the error based on the detail level
func formatError(err error) error {
	if err == nil {
		return nil
	}

	// Get the caller information
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return fmt.Errorf("error occurred: %v", err)
	}

	// Get function name
	fn := runtime.FuncForPC(pc)
	fnName := fn.Name()

	// Only return full error if specifically requested
	if getErrorDetailLevel() == ErrorDetailFull {
		// Capture stack trace
		buffer := make([]byte, 4096)
		n := runtime.Stack(buffer, false)
		stackTrace := string(buffer[:n])

		// Format stack trace
		stackLines := strings.Split(stackTrace, "\n")
		if len(stackLines) > 0 {
			stackLines = stackLines[1:]
		}
		cleanedStack := strings.Join(stackLines, "\n")

		return fmt.Errorf(`
Error Location:
  Full Path: %s
  File: %s
  Line: %d
  Function: %s
Error Details:
  %v
Stack Trace:
%s`, file, filepath.Base(file), line, fnName, err, cleanedStack)
	}

	// Create and return simple error format for both None and Simple detail levels
	return fmt.Errorf("%s:%d [%s]: %v",
		filepath.Base(file),
		line,
		filepath.Base(fnName),
		err)
}

// ErrorWithLocation wraps an error with location information based on detail level
func ErrorWithLocation(err error) error {
	if err == nil {
		return nil
	}
	return formatError(err)
}

// PrintErrorAndReturn prints the error to stderr (if detail level is not None) and returns it
func PrintErrorAndReturn(err error) error {
	if err == nil {
		return nil
	}

	wrappedErr := formatError(err)

	// Only print to console if detail level is not None
	if getErrorDetailLevel() != ErrorDetailNone {
		fmt.Fprintln(os.Stderr, wrappedErr)
	}

	return wrappedErr
}
