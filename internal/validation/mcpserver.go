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
	if err := validateRequired("spec.type", spec["type"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else {
		if err := validateType("spec.type", spec["type"], "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if err := validateEnum("spec.type", spec["type"], []string{"localCommand"}); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	// Validate autoStart (optional, default false)
	if autoStart, ok := spec["autoStart"]; ok {
		if err := validateType("spec.autoStart", autoStart, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate toolPrefix (optional)
	if toolPrefix, ok := spec["toolPrefix"]; ok && toolPrefix != nil {
		if err := validateType("spec.toolPrefix", toolPrefix, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if err := validatePattern("spec.toolPrefix", toolPrefix, "^[a-zA-Z][a-zA-Z0-9_-]*$", "must start with a letter and contain only letters, numbers, underscores, and hyphens"); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	// Validate command (required for localCommand type)
	if command, ok := spec["command"]; ok {
		if err := validateType("spec.command", command, "array"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if err := validateMinItems("spec.command", command, 1); err != nil {
				result.Errors = append(result.Errors, *err)
			} else {
				// Validate each command element is a string
				if cmdArray, ok := command.([]interface{}); ok {
					for i, item := range cmdArray {
						if err := validateType(fmt.Sprintf("spec.command[%d]", i), item, "string"); err != nil {
							result.Errors = append(result.Errors, *err)
						}
					}
				}
			}
		}
	} else {
		// Command is required for localCommand type
		if specType, ok := spec["type"].(string); ok && specType == "localCommand" {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "spec.command",
				Type:    ErrorTypeRequired,
				Message: "command is required when type is \"localCommand\"",
			})
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
		"type", "autoStart", "toolPrefix", "command", "env", "description",
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
