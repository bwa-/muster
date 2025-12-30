package validation

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// MCPServerValidator validates MCPServer YAML configurations
type MCPServerValidator struct{}

// NewMCPServerValidator creates a new MCPServer validator
func NewMCPServerValidator() *MCPServerValidator {
	return &MCPServerValidator{}
}

// Validate validates an MCPServer YAML configuration
func (v *MCPServerValidator) Validate(data []byte, filePath string) ValidationResult {
	result := ValidationResult{
		FilePath: filePath,
		Errors:   []ValidationError{},
	}

	// Parse YAML into generic structure
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "(root)",
			Type:    ErrorTypeInvalidFormat,
			Message: fmt.Sprintf("invalid YAML: %v", err),
		})
		return result
	}

	// Validate required top-level fields
	v.validateTopLevel(&result, raw)

	// If basic structure is invalid, return early
	if len(result.Errors) > 0 {
		return result
	}

	// Validate metadata section
	if metadata, ok := raw["metadata"].(map[string]interface{}); ok {
		v.validateMetadata(&result, metadata)
	}

	// Validate spec section
	if spec, ok := raw["spec"].(map[string]interface{}); ok {
		v.validateSpec(&result, spec)
	}

	// Check for unknown top-level fields
	v.checkUnknownFields(&result, raw, []string{"apiVersion", "kind", "metadata", "spec", "status"})

	return result
}

func (v *MCPServerValidator) validateTopLevel(result *ValidationResult, raw map[string]interface{}) {
	// Validate apiVersion
	if err := validateRequired("apiVersion", raw["apiVersion"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType("apiVersion", raw["apiVersion"], "string"); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if apiVersion, ok := raw["apiVersion"].(string); ok && apiVersion != "muster.giantswarm.io/v1alpha1" {
		result.Errors = append(result.Errors, ValidationError{
			Field:      "apiVersion",
			Type:       ErrorTypeInvalidValue,
			Message:    fmt.Sprintf("invalid apiVersion %q, expected \"muster.giantswarm.io/v1alpha1\"", apiVersion),
			Value:      apiVersion,
			Suggestion: "use \"muster.giantswarm.io/v1alpha1\"",
		})
	}

	// Validate kind
	if err := validateRequired("kind", raw["kind"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType("kind", raw["kind"], "string"); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if kind, ok := raw["kind"].(string); ok && kind != "MCPServer" {
		result.Errors = append(result.Errors, ValidationError{
			Field:      "kind",
			Type:       ErrorTypeInvalidValue,
			Message:    fmt.Sprintf("invalid kind %q, expected \"MCPServer\"", kind),
			Value:      kind,
			Suggestion: "use \"MCPServer\"",
		})
	}

	// Validate metadata exists
	if err := validateRequired("metadata", raw["metadata"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType("metadata", raw["metadata"], "object"); err != nil {
		result.Errors = append(result.Errors, *err)
	}

	// Validate spec exists
	if err := validateRequired("spec", raw["spec"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType("spec", raw["spec"], "object"); err != nil {
		result.Errors = append(result.Errors, *err)
	}
}

func (v *MCPServerValidator) validateMetadata(result *ValidationResult, metadata map[string]interface{}) {
	// Validate name is required
	if err := validateRequired("metadata.name", metadata["name"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType("metadata.name", metadata["name"], "string"); err != nil {
		result.Errors = append(result.Errors, *err)
	}

	// Validate namespace if provided
	if namespace, ok := metadata["namespace"]; ok && namespace != nil {
		if err := validateType("metadata.namespace", namespace, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Check for unknown metadata fields (allow standard k8s fields)
	v.checkUnknownFields(result, metadata, []string{
		"name", "namespace", "labels", "annotations",
		"creationTimestamp", "generation", "resourceVersion", "uid",
	})
}

func (v *MCPServerValidator) validateSpec(result *ValidationResult, spec map[string]interface{}) {
	// Validate type (required)
	var specType string
	if err := validateRequired("spec.type", spec["type"]); err != nil {
		result.Errors = append(result.Errors, *err)
		return // Can't validate further without type
	} else {
		if err := validateType("spec.type", spec["type"], "string"); err != nil {
			result.Errors = append(result.Errors, *err)
			return
		} else {
			if err := validateEnum("spec.type", spec["type"], []string{"stdio", "streamable-http", "sse"}); err != nil {
				result.Errors = append(result.Errors, *err)
				return
			}
			specType = spec["type"].(string)
		}
	}

	// Validate autoStart (optional, default false)
	if autoStart, ok := spec["autoStart"]; ok {
		if err := validateType("spec.autoStart", autoStart, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate toolPrefix (optional)
	if toolPrefix, ok := spec["toolPrefix"]; ok && toolPrefix != nil && toolPrefix != "" {
		if err := validateType("spec.toolPrefix", toolPrefix, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if err := validatePattern("spec.toolPrefix", toolPrefix, "^[a-zA-Z][a-zA-Z0-9_-]*$", "must start with a letter and contain only letters, numbers, underscores, and hyphens"); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	// Validate command (required for stdio type, should be a string - the executable)
	if command, ok := spec["command"]; ok {
		if specType != "stdio" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      "spec.command",
				Type:       ErrorTypeConstraintViolation,
				Message:    "command is only allowed when type is \"stdio\"",
				Suggestion: "remove command field or change type to \"stdio\"",
			})
		} else if err := validateType("spec.command", command, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			// Check that command is not empty
			if cmdStr, ok := command.(string); ok && cmdStr == "" {
				result.Errors = append(result.Errors, ValidationError{
					Field:      "spec.command",
					Type:       ErrorTypeConstraintViolation,
					Message:    "command must not be empty when provided",
					Suggestion: "provide a valid executable path or command",
				})
			}
		}
	} else {
		// Command is required for stdio type
		if specType == "stdio" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "spec.command",
				Type:    ErrorTypeRequired,
				Message: "command is required when type is \"stdio\"",
			})
		}
	}

	// Validate args (optional, array of strings - command arguments, only for stdio)
	if args, ok := spec["args"]; ok && args != nil {
		if specType != "stdio" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      "spec.args",
				Type:       ErrorTypeConstraintViolation,
				Message:    "args is only allowed when type is \"stdio\"",
				Suggestion: "remove args field or change type to \"stdio\"",
			})
		} else if err := validateType("spec.args", args, "array"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			// Validate each arg element is a string
			if argsArray, ok := args.([]interface{}); ok {
				for i, item := range argsArray {
					if err := validateType(fmt.Sprintf("spec.args[%d]", i), item, "string"); err != nil {
						result.Errors = append(result.Errors, *err)
					}
				}
			}
		}
	}

	// Validate url (required for remote types: streamable-http, sse)
	if url, ok := spec["url"]; ok {
		if specType == "stdio" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      "spec.url",
				Type:       ErrorTypeConstraintViolation,
				Message:    "url is only allowed when type is \"streamable-http\" or \"sse\"",
				Suggestion: "remove url field or change type to \"streamable-http\" or \"sse\"",
			})
		} else if err := validateType("spec.url", url, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			// Validate URL pattern
			if err := validatePattern("spec.url", url, `^https?://[^\s/$.?#].[^\s]*$`, "must be a valid HTTP(S) URL"); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	} else {
		// URL is required for remote types
		if specType == "streamable-http" || specType == "sse" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "spec.url",
				Type:    ErrorTypeRequired,
				Message: fmt.Sprintf("url is required when type is %q", specType),
			})
		}
	}

	// Validate headers (optional, only for remote types)
	if headers, ok := spec["headers"]; ok && headers != nil {
		if specType == "stdio" {
			result.Errors = append(result.Errors, ValidationError{
				Field:      "spec.headers",
				Type:       ErrorTypeConstraintViolation,
				Message:    "headers is only allowed when type is \"streamable-http\" or \"sse\"",
				Suggestion: "remove headers field or change type to \"streamable-http\" or \"sse\"",
			})
		} else if err := validateType("spec.headers", headers, "object"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			// Validate each header value is a string
			if headersMap, ok := headers.(map[string]interface{}); ok {
				for key, value := range headersMap {
					if err := validateType(fmt.Sprintf("spec.headers.%s", key), value, "string"); err != nil {
						result.Errors = append(result.Errors, *err)
					}
				}
			}
		}
	}

	// Validate timeout (optional, integer with min/max constraints)
	if timeout, ok := spec["timeout"]; ok && timeout != nil {
		if err := validateType("spec.timeout", timeout, "number"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			// Convert to int for range validation
			var timeoutInt int
			switch v := timeout.(type) {
			case int:
				timeoutInt = v
			case float64:
				timeoutInt = int(v)
			default:
				result.Errors = append(result.Errors, ValidationError{
					Field:   "spec.timeout",
					Type:    ErrorTypeInvalidType,
					Message: "timeout must be an integer",
				})
			}
			if timeoutInt < 1 || timeoutInt > 300 {
				result.Errors = append(result.Errors, ValidationError{
					Field:      "spec.timeout",
					Type:       ErrorTypeInvalidValue,
					Message:    fmt.Sprintf("timeout must be between 1 and 300 seconds, got %d", timeoutInt),
					Value:      timeoutInt,
					Suggestion: "set timeout between 1 and 300",
				})
			}
		}
	}

	// Validate env (optional)
	if env, ok := spec["env"]; ok && env != nil {
		if err := validateType("spec.env", env, "object"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			// Validate each env value is a string
			if envMap, ok := env.(map[string]interface{}); ok {
				for key, value := range envMap {
					if err := validateType(fmt.Sprintf("spec.env.%s", key), value, "string"); err != nil {
						result.Errors = append(result.Errors, *err)
					}
				}
			}
		}
	}

	// Validate description (optional)
	if description, ok := spec["description"]; ok && description != nil {
		if err := validateType("spec.description", description, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if err := validateMaxLength("spec.description", description, 500); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	// Check for unknown spec fields
	v.checkUnknownFields(result, spec, []string{
		"type", "autoStart", "toolPrefix", "command", "args", "url", "headers", "timeout", "env", "description",
	})
}

func (v *MCPServerValidator) checkUnknownFields(result *ValidationResult, obj map[string]interface{}, knownFields []string) {
	knownMap := make(map[string]bool)
	for _, field := range knownFields {
		knownMap[field] = true
	}

	for field := range obj {
		if !knownMap[field] {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Type:    ErrorTypeUnknownField,
				Message: fmt.Sprintf("unknown field %q", field),
			})
		}
	}
}
