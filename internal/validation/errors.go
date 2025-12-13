package validation

import (
	"fmt"
	"strings"
)

// ErrorType categorizes the type of validation error
type ErrorType string

const (
	// ErrorTypeRequired indicates a required field is missing
	ErrorTypeRequired ErrorType = "required"
	// ErrorTypeInvalidType indicates a field has wrong type
	ErrorTypeInvalidType ErrorType = "invalid_type"
	// ErrorTypeInvalidValue indicates a field value is not allowed
	ErrorTypeInvalidValue ErrorType = "invalid_value"
	// ErrorTypeInvalidFormat indicates a field format is incorrect
	ErrorTypeInvalidFormat ErrorType = "invalid_format"
	// ErrorTypeDuplicate indicates a duplicate value where uniqueness is required
	ErrorTypeDuplicate ErrorType = "duplicate"
	// ErrorTypeConflict indicates conflicting fields
	ErrorTypeConflict ErrorType = "conflict"
	// ErrorTypeUnknownField indicates an unsupported field
	ErrorTypeUnknownField ErrorType = "unknown_field"
	// ErrorTypeConstraintViolation indicates a constraint violation
	ErrorTypeConstraintViolation ErrorType = "constraint_violation"
)

// ValidationError represents a single validation error with context
type ValidationError struct {
	// Field is the JSON path to the field with the error (e.g., "spec.steps[2].condition.expect.success")
	Field string
	// Type categorizes the error
	Type ErrorType
	// Message is a human-readable error message
	Message string
	// Value is the actual value that caused the error (optional)
	Value interface{}
	// Suggestion provides a hint to fix the error (optional)
	Suggestion string
}

// Error implements the error interface
func (e ValidationError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s (suggestion: %s)", e.Field, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationResult contains all validation errors for a resource
type ValidationResult struct {
	// FilePath is the path to the YAML file being validated
	FilePath string
	// Errors contains all validation errors found
	Errors []ValidationError
}

// IsValid returns true if there are no validation errors
func (r ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// Error implements the error interface, formatting all errors
func (r ValidationResult) Error() string {
	if r.IsValid() {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s:\n", r.FilePath))
	for _, err := range r.Errors {
		b.WriteString(fmt.Sprintf("  - %s\n", err.Error()))
	}
	return b.String()
}

// ValidationResults aggregates multiple validation results
type ValidationResults struct {
	Results []ValidationResult
}

// IsValid returns true if all results are valid
func (vr ValidationResults) IsValid() bool {
	for _, result := range vr.Results {
		if !result.IsValid() {
			return false
		}
	}
	return true
}

// ErrorCount returns the total number of errors across all results
func (vr ValidationResults) ErrorCount() int {
	count := 0
	for _, result := range vr.Results {
		count += len(result.Errors)
	}
	return count
}

// FileCount returns the number of files with errors
func (vr ValidationResults) FileCount() int {
	count := 0
	for _, result := range vr.Results {
		if !result.IsValid() {
			count++
		}
	}
	return count
}

// Error implements the error interface, formatting all errors from all files
func (vr ValidationResults) Error() string {
	if vr.IsValid() {
		return ""
	}

	var b strings.Builder
	b.WriteString("❌ Configuration validation failed:\n\n")

	for _, result := range vr.Results {
		if !result.IsValid() {
			b.WriteString(result.Error())
			b.WriteString("\n")
		}
	}

	b.WriteString(fmt.Sprintf("Found %d validation error(s) across %d file(s). Please fix and restart.\n",
		vr.ErrorCount(), vr.FileCount()))

	return b.String()
}
