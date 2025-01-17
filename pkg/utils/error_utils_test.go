package utils

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

func TestGetErrorDetailLevel(t *testing.T) {
	tests := []struct {
		name          string
		envValue      string
		expectedLevel ErrorDetailLevel
	}{
		{
			name:          "none detail level",
			envValue:      "none",
			expectedLevel: ErrorDetailNone,
		},
		{
			name:          "full detail level",
			envValue:      "full",
			expectedLevel: ErrorDetailFull,
		},
		{
			name:          "simple detail level (default)",
			envValue:      "simple",
			expectedLevel: ErrorDetailSimple,
		},
		{
			name:          "empty env defaults to simple",
			envValue:      "",
			expectedLevel: ErrorDetailSimple,
		},
		{
			name:          "invalid value defaults to simple",
			envValue:      "invalid",
			expectedLevel: ErrorDetailSimple,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ERROR_DETAIL_LEVEL", tt.envValue)
			defer os.Unsetenv("ERROR_DETAIL_LEVEL")

			if got := getErrorDetailLevel(); got != tt.expectedLevel {
				t.Errorf("getErrorDetailLevel() = %v, want %v", got, tt.expectedLevel)
			}
		})
	}
}

func TestErrorWithLocation(t *testing.T) {
	// First, show example formats
	t.Run("show_formats", func(t *testing.T) {
		ExampleErrorFormats(t)
	})

	tests := []struct {
		name            string
		err             error
		detailLevel     string
		expectedParts   []string
		unexpectedParts []string
	}{
		{
			name:        "nil error returns nil",
			err:         nil,
			detailLevel: "simple",
		},
		{
			name:        "simple detail level",
			err:         errors.New("test error"),
			detailLevel: "simple",
			expectedParts: []string{
				"error_utils.go",
				"test error",
			},
			unexpectedParts: []string{
				"Stack Trace:",
				"Error Location:",
			},
		},
		{
			name:        "full detail level",
			err:         errors.New("test error"),
			detailLevel: "full",
			expectedParts: []string{
				"Error Location:",
				"Full Path:",
				"File: error_utils.go",
				"Line:",
				"Function:",
				"Error Details:",
				"test error",
				"Stack Trace:",
			},
		},
		{
			name:        "none detail level",
			err:         errors.New("test error"),
			detailLevel: "none",
			expectedParts: []string{
				"error_utils.go",
				"test error",
			},
			unexpectedParts: []string{
				"Stack Trace:",
				"Error Location:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ERROR_DETAIL_LEVEL", tt.detailLevel)
			defer os.Unsetenv("ERROR_DETAIL_LEVEL")

			got := ErrorWithLocation(tt.err)

			if tt.err == nil {
				if got != nil {
					t.Errorf("ErrorWithLocation() = %v, want nil", got)
				}
				return
			}

			gotStr := got.Error()

			// Check for expected parts
			for _, expected := range tt.expectedParts {
				if !strings.Contains(gotStr, expected) {
					t.Errorf("ErrorWithLocation() output missing expected part %q in:\n%s", expected, gotStr)
				}
			}

			// Check that unexpected parts are not present
			for _, unexpected := range tt.unexpectedParts {
				if strings.Contains(gotStr, unexpected) {
					t.Errorf("ErrorWithLocation() output contains unexpected part %q in:\n%s", unexpected, gotStr)
				}
			}
		})
	}
}

func TestPrintErrorAndReturn(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		detailLevel string
		shouldPrint bool
	}{
		{
			name:        "nil error returns nil",
			err:         nil,
			detailLevel: "simple",
			shouldPrint: false,
		},
		{
			name:        "prints with simple detail level",
			err:         errors.New("test error"),
			detailLevel: "simple",
			shouldPrint: true,
		},
		{
			name:        "prints with full detail level",
			err:         errors.New("test error"),
			detailLevel: "full",
			shouldPrint: true,
		},
		{
			name:        "suppresses print with none detail level",
			err:         errors.New("test error"),
			detailLevel: "none",
			shouldPrint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ERROR_DETAIL_LEVEL", tt.detailLevel)
			defer os.Unsetenv("ERROR_DETAIL_LEVEL")

			// Redirect stderr to capture output
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			got := PrintErrorAndReturn(tt.err)

			// Restore stderr
			w.Close()
			os.Stderr = oldStderr

			// Read captured output
			output, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("Failed to read captured output: %v", err)
			}
			outputStr := string(output)

			// Verify return value
			if tt.err == nil {
				if got != nil {
					t.Errorf("PrintErrorAndReturn() = %v, want nil", got)
				}
				return
			}

			// Verify printing behavior
			outputPresent := outputStr != ""
			if outputPresent != tt.shouldPrint {
				t.Errorf("PrintErrorAndReturn() printing = %v, want %v", outputPresent, tt.shouldPrint)
			}

			// Verify error is wrapped correctly
			if got == nil {
				t.Error("PrintErrorAndReturn() returned nil for non-nil error")
			}
		})
	}
}

func ExampleErrorFormats(t *testing.T) {
	// Save original env var to restore later
	origEnv := os.Getenv("ERROR_DETAIL_LEVEL")
	defer os.Setenv("ERROR_DETAIL_LEVEL", origEnv)

	t.Log(strings.Repeat("=", 80))
	t.Log("ERROR FORMAT EXAMPLES")
	t.Log(strings.Repeat("=", 80))

	// Create a sample error
	originalErr := errors.New("something went wrong")
	t.Logf("\n>>> Original error:\n%v\n\n", originalErr)

	// Show Simple format (default)
	os.Setenv("ERROR_DETAIL_LEVEL", "simple")
	simpleErr := ErrorWithLocation(originalErr)
	t.Logf("\n>>> Simple format (default):\n%v\n\n", simpleErr)

	// Show Full format
	os.Setenv("ERROR_DETAIL_LEVEL", "full")
	fullErr := ErrorWithLocation(originalErr)
	t.Logf("\n>>> Full format:\n%v\n\n", fullErr)

	// Show None format
	os.Setenv("ERROR_DETAIL_LEVEL", "none")
	noneErr := ErrorWithLocation(originalErr)
	t.Logf("\n>>> None format:\n%v\n\n", noneErr)

	t.Log("\n" + strings.Repeat("=", 80) + "\n")
}
