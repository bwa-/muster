package validation

import (
	"fmt"
	"reflect"
	"regexp"
)

// Validator defines the interface for resource validators
type Validator interface {
	// Validate validates the given YAML data and returns validation errors
	Validate(data []byte, filePath string) ValidationResult
}

// Helper functions for common validation patterns

// validateRequired checks if a field exists and is not empty
func validateRequired(fieldPath string, value interface{}) *ValidationError {
	if value == nil {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeRequired,
			Message: "field is required but not provided",
		}
	}

	// Check for empty strings
	if str, ok := value.(string); ok && str == "" {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeRequired,
			Message: "field is required but empty",
		}
	}

	// Check for empty slices
	if reflect.TypeOf(value).Kind() == reflect.Slice && reflect.ValueOf(value).Len() == 0 {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeRequired,
			Message: "field is required but empty array",
		}
	}

	return nil
}

// validateType checks if a value matches the expected type
func validateType(fieldPath string, value interface{}, expectedType string) *ValidationError {
	if value == nil {
		return nil // nil values are handled by validateRequired
	}

	actualType := reflect.TypeOf(value).Kind().String()
	
	// Map Go types to expected types
	typeMap := map[string][]string{
		"string": {"string"},
		"bool":   {"bool"},
		"int":    {"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64"},
		"float":  {"float32", "float64"},
		"array":  {"slice", "array"},
		"object": {"map", "struct"},
	}

	validTypes, ok := typeMap[expectedType]
	if !ok {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeInvalidType,
			Message: fmt.Sprintf("unknown expected type: %s", expectedType),
		}
	}

	for _, validType := range validTypes {
		if actualType == validType {
			return nil
		}
	}

	return &ValidationError{
		Field:   fieldPath,
		Type:    ErrorTypeInvalidType,
		Message: fmt.Sprintf("expected %s, got %s", expectedType, actualType),
		Value:   value,
	}
}

// validateEnum checks if a value is in the allowed set
func validateEnum(fieldPath string, value interface{}, allowedValues []string) *ValidationError {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeInvalidType,
			Message: fmt.Sprintf("expected string for enum, got %T", value),
			Value:   value,
		}
	}

	for _, allowed := range allowedValues {
		if str == allowed {
			return nil
		}
	}

	return &ValidationError{
		Field:      fieldPath,
		Type:       ErrorTypeInvalidValue,
		Message:    fmt.Sprintf("invalid value %q, must be one of: %v", str, allowedValues),
		Value:      value,
		Suggestion: findClosestMatch(str, allowedValues),
	}
}

// validatePattern checks if a string matches a regex pattern
func validatePattern(fieldPath string, value interface{}, pattern string, description string) *ValidationError {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeInvalidType,
			Message: fmt.Sprintf("expected string, got %T", value),
			Value:   value,
		}
	}

	if str == "" {
		return nil // Empty strings handled by validateRequired
	}

	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeInvalidFormat,
			Message: fmt.Sprintf("pattern validation error: %v", err),
		}
	}

	if !matched {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeInvalidFormat,
			Message: fmt.Sprintf("invalid format: %s", description),
			Value:   value,
		}
	}

	return nil
}

// validateMaxLength checks string length constraint
func validateMaxLength(fieldPath string, value interface{}, maxLength int) *ValidationError {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil // Type validation handled elsewhere
	}

	if len(str) > maxLength {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeConstraintViolation,
			Message: fmt.Sprintf("exceeds maximum length of %d characters (got %d)", maxLength, len(str)),
			Value:   value,
		}
	}

	return nil
}

// validateMinItems checks slice minimum length
func validateMinItems(fieldPath string, value interface{}, minItems int) *ValidationError {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil // Type validation handled elsewhere
	}

	if v.Len() < minItems {
		return &ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeConstraintViolation,
			Message: fmt.Sprintf("must have at least %d item(s), got %d", minItems, v.Len()),
			Value:   value,
		}
	}

	return nil
}

// findClosestMatch finds the closest string match for suggestions (simple implementation)
func findClosestMatch(input string, candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	// Simple matching: case-insensitive contains
	input = toLower(input)
	for _, candidate := range candidates {
		if toLower(candidate) == input {
			return fmt.Sprintf("did you mean %q?", candidate)
		}
	}

	// Check if input is contained in any candidate
	for _, candidate := range candidates {
		if contains(toLower(candidate), input) {
			return fmt.Sprintf("did you mean %q?", candidate)
		}
	}

	return ""
}

func toLower(s string) string {
	return regexp.MustCompile("[A-Z]").ReplaceAllStringFunc(s, func(m string) string {
		return string(m[0] + 32)
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		regexp.MustCompile(regexp.QuoteMeta(substr)).MatchString(s)))
}
