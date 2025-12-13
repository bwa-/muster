package validation

import (
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      ValidationError
		expected string
	}{
		{
			name: "error without suggestion",
			err: ValidationError{
				Field:   "spec.type",
				Type:    ErrorTypeInvalidValue,
				Message: "invalid value",
			},
			expected: "spec.type: invalid value",
		},
		{
			name: "error with suggestion",
			err: ValidationError{
				Field:      "spec.type",
				Type:       ErrorTypeInvalidValue,
				Message:    "invalid value",
				Suggestion: "did you mean \"localCommand\"?",
			},
			expected: "spec.type: invalid value (suggestion: did you mean \"localCommand\"?)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("ValidationError.Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestValidationResult_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		result ValidationResult
		want   bool
	}{
		{
			name: "valid result with no errors",
			result: ValidationResult{
				FilePath: "test.yaml",
				Errors:   []ValidationError{},
			},
			want: true,
		},
		{
			name: "invalid result with errors",
			result: ValidationResult{
				FilePath: "test.yaml",
				Errors: []ValidationError{
					{Field: "spec.type", Type: ErrorTypeRequired, Message: "required"},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.IsValid(); got != tt.want {
				t.Errorf("ValidationResult.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		fieldPath string
		value     interface{}
		wantError bool
	}{
		{
			name:      "nil value",
			fieldPath: "field",
			value:     nil,
			wantError: true,
		},
		{
			name:      "empty string",
			fieldPath: "field",
			value:     "",
			wantError: true,
		},
		{
			name:      "valid string",
			fieldPath: "field",
			value:     "value",
			wantError: false,
		},
		{
			name:      "empty slice",
			fieldPath: "field",
			value:     []string{},
			wantError: true,
		},
		{
			name:      "non-empty slice",
			fieldPath: "field",
			value:     []string{"item"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequired(tt.fieldPath, tt.value)
			if (err != nil) != tt.wantError {
				t.Errorf("validateRequired() error = %v, wantError %v", err, tt.wantError)
			}
			if err != nil && err.Type != ErrorTypeRequired {
				t.Errorf("validateRequired() error type = %v, want %v", err.Type, ErrorTypeRequired)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	tests := []struct {
		name          string
		fieldPath     string
		value         interface{}
		allowedValues []string
		wantError     bool
	}{
		{
			name:          "valid enum value",
			fieldPath:     "spec.type",
			value:         "localCommand",
			allowedValues: []string{"localCommand", "remote"},
			wantError:     false,
		},
		{
			name:          "invalid enum value",
			fieldPath:     "spec.type",
			value:         "local",
			allowedValues: []string{"localCommand", "remote"},
			wantError:     true,
		},
		{
			name:          "nil value",
			fieldPath:     "spec.type",
			value:         nil,
			allowedValues: []string{"localCommand"},
			wantError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnum(tt.fieldPath, tt.value, tt.allowedValues)
			if (err != nil) != tt.wantError {
				t.Errorf("validateEnum() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
