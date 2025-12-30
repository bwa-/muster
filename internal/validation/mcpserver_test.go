package validation

import (
	"testing"
)

func TestMCPServerValidator_Validate_ValidConfig(t *testing.T) {
	validator := NewMCPServerValidator()

	validYAML := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test-server
  namespace: default
spec:
  type: stdio
  autoStart: true
  toolPrefix: test
  command: "npx"
  args: ["@modelcontextprotocol/server-git"]
  env:
    LOG_LEVEL: "info"
  description: "Test MCP server"
`

	result := validator.Validate([]byte(validYAML), "test.yaml")

	if !result.IsValid() {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestMCPServerValidator_Validate_MissingRequired(t *testing.T) {
	validator := NewMCPServerValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
		expectedType  ErrorType
	}{
		{
			name: "missing apiVersion",
			yaml: `
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "apiVersion",
			expectedType:  ErrorTypeRequired,
		},
		{
			name: "missing kind",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "kind",
			expectedType:  ErrorTypeRequired,
		},
		{
			name: "missing metadata",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "metadata",
			expectedType:  ErrorTypeRequired,
		},
		{
			name: "missing metadata.name",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  namespace: default
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "metadata.name",
			expectedType:  ErrorTypeRequired,
		},
		{
			name: "missing spec",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
`,
			expectedField: "spec",
			expectedType:  ErrorTypeRequired,
		},
		{
			name: "missing spec.type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  command: "test"
  args: []
`,
			expectedField: "spec.type",
			expectedType:  ErrorTypeRequired,
		},
		{
			name: "missing spec.command for stdio",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
`,
			expectedField: "spec.command",
			expectedType:  ErrorTypeRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate([]byte(tt.yaml), "test.yaml")

			if result.IsValid() {
				t.Error("Expected validation errors, got none")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Field == tt.expectedField && err.Type == tt.expectedType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error for field %q with type %q, got errors: %v",
					tt.expectedField, tt.expectedType, result.Errors)
			}
		})
	}
}

func TestMCPServerValidator_Validate_InvalidTypes(t *testing.T) {
	validator := NewMCPServerValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
	}{
		{
			name: "apiVersion wrong type",
			yaml: `
apiVersion: 123
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "apiVersion",
		},
		{
			name: "kind wrong type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: 123
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "kind",
		},
		{
			name: "metadata.name wrong type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: 123
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "metadata.name",
		},
		{
			name: "spec.autoStart wrong type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  autoStart: "yes"
  command: "test"
  args: []
`,
			expectedField: "spec.autoStart",
		},
		{
			name: "spec.command wrong type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: 123
`,
			expectedField: "spec.command",
		},
		{
			name: "spec.env wrong type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
  env: ["not", "a", "map"]
`,
			expectedField: "spec.env",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate([]byte(tt.yaml), "test.yaml")

			if result.IsValid() {
				t.Error("Expected validation errors, got none")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Field == tt.expectedField && err.Type == ErrorTypeInvalidType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected invalid type error for field %q, got errors: %v",
					tt.expectedField, result.Errors)
			}
		})
	}
}

func TestMCPServerValidator_Validate_InvalidValues(t *testing.T) {
	validator := NewMCPServerValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
		expectedType  ErrorType
	}{
		{
			name: "invalid apiVersion",
			yaml: `
apiVersion: v1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "apiVersion",
			expectedType:  ErrorTypeInvalidValue,
		},
		{
			name: "invalid kind",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Server
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
`,
			expectedField: "kind",
			expectedType:  ErrorTypeInvalidValue,
		},
		{
			name: "invalid spec.type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: remote
  command: "test"
  args: []
`,
			expectedField: "spec.type",
			expectedType:  ErrorTypeInvalidValue,
		},
		{
			name: "spec.type with typo",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: localcommand
  command: "test"
  args: []
`,
			expectedField: "spec.type",
			expectedType:  ErrorTypeInvalidValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate([]byte(tt.yaml), "test.yaml")

			if result.IsValid() {
				t.Error("Expected validation errors, got none")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Field == tt.expectedField && err.Type == tt.expectedType {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected error for field %q with type %q, got errors: %v",
					tt.expectedField, tt.expectedType, result.Errors)
			}
		})
	}
}

func TestMCPServerValidator_Validate_FormatValidation(t *testing.T) {
	validator := NewMCPServerValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
	}{
		{
			name: "invalid toolPrefix - starts with number",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  toolPrefix: "123test"
  command: "test"
  args: []
`,
			expectedField: "spec.toolPrefix",
		},
		{
			name: "invalid toolPrefix - special chars",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  toolPrefix: "test@prefix"
  command: "test"
  args: []
`,
			expectedField: "spec.toolPrefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate([]byte(tt.yaml), "test.yaml")

			if result.IsValid() {
				t.Error("Expected validation errors, got none")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Field == tt.expectedField && err.Type == ErrorTypeInvalidFormat {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected format error for field %q, got errors: %v",
					tt.expectedField, result.Errors)
			}
		})
	}
}

func TestMCPServerValidator_Validate_ConstraintViolations(t *testing.T) {
	validator := NewMCPServerValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
	}{
		{
			name: "description too long",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
  description: "This is a very long description that exceeds the maximum allowed length of 500 characters. Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum plus extra text"
`,
			expectedField: "spec.description",
		},
		{
			name: "command array empty",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: ""
`,
			expectedField: "spec.command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate([]byte(tt.yaml), "test.yaml")

			if result.IsValid() {
				t.Error("Expected validation errors, got none")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Field == tt.expectedField && err.Type == ErrorTypeConstraintViolation {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected constraint violation for field %q, got errors: %v",
					tt.expectedField, result.Errors)
			}
		})
	}
}

func TestMCPServerValidator_Validate_ValidOptionalFields(t *testing.T) {
	validator := NewMCPServerValidator()

	// Test with all optional fields
	yaml := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test-server
  namespace: default
  labels:
    app: test
  annotations:
    description: "test annotation"
spec:
  type: stdio
  autoStart: true
  toolPrefix: myprefix
  command: "npx"
  args: []
  env:
    LOG_LEVEL: "debug"
    DEBUG: "true"
  description: "A test MCP server with all optional fields"
`

	result := validator.Validate([]byte(yaml), "test.yaml")

	if !result.IsValid() {
		t.Errorf("Expected valid config with optional fields, got errors: %v", result.Errors)
	}
}

func TestMCPServerValidator_Validate_MinimalConfig(t *testing.T) {
	validator := NewMCPServerValidator()

	// Test minimal valid config
	yaml := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: test
spec:
  type: stdio
  command: "test"
  args: []
`

	result := validator.Validate([]byte(yaml), "test.yaml")

	if !result.IsValid() {
		t.Errorf("Expected valid minimal config, got errors: %v", result.Errors)
	}
}

func TestMCPServerValidator_Validate_InvalidYAML(t *testing.T) {
	validator := NewMCPServerValidator()

	invalidYAML := `
this is not valid YAML:
  - missing proper structure
  [unclosed bracket
`

	result := validator.Validate([]byte(invalidYAML), "test.yaml")

	if result.IsValid() {
		t.Error("Expected validation errors for invalid YAML, got none")
	}

	// Should have an error about invalid YAML format
	found := false
	for _, err := range result.Errors {
		if err.Type == ErrorTypeInvalidFormat {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected invalid format error, got: %v", result.Errors)
	}
}
