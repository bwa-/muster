package validation

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// WorkflowValidator validates Workflow YAML configurations
type WorkflowValidator struct{}

// NewWorkflowValidator creates a new Workflow validator
func NewWorkflowValidator() *WorkflowValidator {
	return &WorkflowValidator{}
}

// Validate validates a Workflow YAML configuration
func (v *WorkflowValidator) Validate(data []byte, filePath string) ValidationResult {
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
	v.checkUnknownFields(&result, raw, "(root)", []string{"apiVersion", "kind", "metadata", "spec", "status"})

	return result
}

func (v *WorkflowValidator) validateTopLevel(result *ValidationResult, raw map[string]interface{}) {
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
	} else if kind, ok := raw["kind"].(string); ok && kind != "Workflow" {
		result.Errors = append(result.Errors, ValidationError{
			Field:      "kind",
			Type:       ErrorTypeInvalidValue,
			Message:    fmt.Sprintf("invalid kind %q, expected \"Workflow\"", kind),
			Value:      kind,
			Suggestion: "use \"Workflow\"",
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

func (v *WorkflowValidator) validateMetadata(result *ValidationResult, metadata map[string]interface{}) {
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

	// Check for unknown metadata fields
	v.checkUnknownFields(result, metadata, "metadata", []string{
		"name", "namespace", "labels", "annotations",
		"creationTimestamp", "generation", "resourceVersion", "uid",
	})
}

func (v *WorkflowValidator) validateSpec(result *ValidationResult, spec map[string]interface{}) {
	// Validate steps (required)
	if err := validateRequired("spec.steps", spec["steps"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else {
		if err := validateType("spec.steps", spec["steps"], "array"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if steps, ok := spec["steps"].([]interface{}); ok {
				v.validateSteps(result, steps)
			}
		}
	}

	// Validate description (optional)
	if description, ok := spec["description"]; ok && description != nil {
		if err := validateType("spec.description", description, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if err := validateMaxLength("spec.description", description, 1000); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	// Validate args (optional)
	if args, ok := spec["args"]; ok && args != nil {
		if err := validateType("spec.args", args, "object"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if argsMap, ok := args.(map[string]interface{}); ok {
				v.validateArgs(result, argsMap)
			}
		}
	}

	// Check for unknown spec fields
	v.checkUnknownFields(result, spec, "spec", []string{
		"description", "args", "steps",
	})
}

func (v *WorkflowValidator) validateArgs(result *ValidationResult, args map[string]interface{}) {
	for argName, argDef := range args {
		if argDefMap, ok := argDef.(map[string]interface{}); ok {
			v.validateArgDefinition(result, fmt.Sprintf("spec.args.%s", argName), argDefMap)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("spec.args.%s", argName),
				Type:    ErrorTypeInvalidType,
				Message: "arg definition must be an object",
			})
		}
	}
}

func (v *WorkflowValidator) validateArgDefinition(result *ValidationResult, fieldPath string, argDef map[string]interface{}) {
	// Validate type (required)
	if err := validateRequired(fieldPath+".type", argDef["type"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else {
		if err := validateEnum(fieldPath+".type", argDef["type"], []string{"string", "integer", "boolean", "number", "object", "array"}); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate required (optional, default false)
	if required, ok := argDef["required"]; ok && required != nil {
		if err := validateType(fieldPath+".required", required, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate description (optional)
	if description, ok := argDef["description"]; ok && description != nil {
		if err := validateType(fieldPath+".description", description, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		} else {
			if err := validateMaxLength(fieldPath+".description", description, 500); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	// Validate default (optional) - can be any type
	// Type checking for default value would require knowing the arg type

	// Check for unknown arg definition fields
	v.checkUnknownFields(result, argDef, fieldPath, []string{"type", "required", "description", "default"})
}

func (v *WorkflowValidator) validateSteps(result *ValidationResult, steps []interface{}) {
	stepIDs := make(map[string]int)

	for i, step := range steps {
		if stepMap, ok := step.(map[string]interface{}); ok {
			v.validateStep(result, fmt.Sprintf("spec.steps[%d]", i), stepMap, stepIDs)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("spec.steps[%d]", i),
				Type:    ErrorTypeInvalidType,
				Message: "step must be an object",
			})
		}
	}
}

func (v *WorkflowValidator) validateStep(result *ValidationResult, fieldPath string, step map[string]interface{}, stepIDs map[string]int) {
	// Validate id (required)
	if err := validateRequired(fieldPath+".id", step["id"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType(fieldPath+".id", step["id"], "string"); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if id, ok := step["id"].(string); ok {
		// Check for duplicate IDs
		if prevIndex, exists := stepIDs[id]; exists {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath + ".id",
				Type:    ErrorTypeDuplicate,
				Message: fmt.Sprintf("duplicate step ID %q (already used in step %d)", id, prevIndex),
				Value:   id,
			})
		} else {
			// Extract index from fieldPath like "spec.steps[2]"
			var index int
			fmt.Sscanf(fieldPath, "spec.steps[%d]", &index)
			stepIDs[id] = index
		}
	}

	// Check for mutually exclusive fields: (tool + args) vs forEach
	hasToolOrArgs := step["tool"] != nil || step["args"] != nil
	hasForEach := step["forEach"] != nil

	if hasToolOrArgs && hasForEach {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeConflict,
			Message: "cannot specify both 'tool'/'args' and 'forEach' in the same step",
		})
	}

	if hasForEach {
		// Validate forEach structure
		if forEach, ok := step["forEach"].(map[string]interface{}); ok {
			v.validateForEach(result, fieldPath+".forEach", forEach)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath + ".forEach",
				Type:    ErrorTypeInvalidType,
				Message: "forEach must be an object",
			})
		}
	} else {
		// Validate tool (required if no forEach)
		if err := validateRequired(fieldPath+".tool", step["tool"]); err != nil {
			result.Errors = append(result.Errors, *err)
		} else if err := validateType(fieldPath+".tool", step["tool"], "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		}

		// Validate args (optional)
		if args, ok := step["args"]; ok && args != nil {
			if err := validateType(fieldPath+".args", args, "object"); err != nil {
				result.Errors = append(result.Errors, *err)
			}
		}
	}

	// Validate condition (optional)
	if condition, ok := step["condition"]; ok && condition != nil {
		if conditionMap, ok := condition.(map[string]interface{}); ok {
			v.validateCondition(result, fieldPath+".condition", conditionMap)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath + ".condition",
				Type:    ErrorTypeInvalidType,
				Message: "condition must be an object",
			})
		}
	}

	// Validate allowFailure (optional)
	if allowFailure, ok := step["allowFailure"]; ok && allowFailure != nil {
		if err := validateType(fieldPath+".allowFailure", allowFailure, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate store (optional)
	if store, ok := step["store"]; ok && store != nil {
		if err := validateType(fieldPath+".store", store, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate description (optional)
	if description, ok := step["description"]; ok && description != nil {
		if err := validateType(fieldPath+".description", description, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate outputs (optional)
	if outputs, ok := step["outputs"]; ok && outputs != nil {
		if err := validateType(fieldPath+".outputs", outputs, "object"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Check for unknown step fields
	v.checkUnknownFields(result, step, fieldPath, []string{
		"id", "tool", "args", "condition", "forEach",
		"allowFailure", "store", "description", "outputs",
	})
}

func (v *WorkflowValidator) validateCondition(result *ValidationResult, fieldPath string, condition map[string]interface{}) {
	// Validate mutually exclusive: tool vs fromStep
	hasTool := condition["tool"] != nil
	hasFromStep := condition["from_step"] != nil || condition["fromStep"] != nil

	if hasTool && hasFromStep {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeConflict,
			Message: "cannot specify both 'tool' and 'from_step'/'fromStep' in condition",
		})
	}

	if !hasTool && !hasFromStep {
		result.Errors = append(result.Errors, ValidationError{
			Field:   fieldPath,
			Type:    ErrorTypeRequired,
			Message: "must specify either 'tool' or 'from_step'/'fromStep' in condition",
		})
	}

	// Validate tool (optional)
	if tool, ok := condition["tool"]; ok && tool != nil {
		if err := validateType(fieldPath+".tool", tool, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate fromStep (optional) - support both snake_case and camelCase
	fromStepField := "from_step"
	fromStep := condition["from_step"]
	if fromStep == nil {
		fromStepField = "fromStep"
		fromStep = condition["fromStep"]
	}

	if fromStep != nil {
		if err := validateType(fieldPath+"."+fromStepField, fromStep, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate args (optional)
	if args, ok := condition["args"]; ok && args != nil {
		if err := validateType(fieldPath+".args", args, "object"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate expect (optional)
	if expect, ok := condition["expect"]; ok && expect != nil {
		if expectMap, ok := expect.(map[string]interface{}); ok {
			v.validateExpectation(result, fieldPath+".expect", expectMap)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath + ".expect",
				Type:    ErrorTypeInvalidType,
				Message: "expect must be an object",
			})
		}
	}

	// Validate expectNot (optional) - support both snake_case and camelCase
	expectNotField := "expect_not"
	expectNot := condition["expect_not"]
	if expectNot == nil {
		expectNotField = "expectNot"
		expectNot = condition["expectNot"]
	}

	if expectNot != nil {
		if expectNotMap, ok := expectNot.(map[string]interface{}); ok {
			v.validateExpectation(result, fieldPath+"."+expectNotField, expectNotMap)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath + "." + expectNotField,
				Type:    ErrorTypeInvalidType,
				Message: expectNotField + " must be an object",
			})
		}
	}

	// Check for unknown condition fields
	v.checkUnknownFields(result, condition, fieldPath, []string{
		"tool", "from_step", "fromStep", "args", "expect", "expect_not", "expectNot",
	})
}

func (v *WorkflowValidator) validateExpectation(result *ValidationResult, fieldPath string, expect map[string]interface{}) {
	// Validate success (optional)
	if success, ok := expect["success"]; ok && success != nil {
		if err := validateType(fieldPath+".success", success, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate jsonPath (optional) - support both snake_case and camelCase
	jsonPathField := "json_path"
	jsonPath := expect["json_path"]
	if jsonPath == nil {
		jsonPathField = "jsonPath"
		jsonPath = expect["jsonPath"]
	}

	if jsonPath != nil {
		if err := validateType(fieldPath+"."+jsonPathField, jsonPath, "object"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Check for unknown expectation fields
	v.checkUnknownFields(result, expect, fieldPath, []string{
		"success", "json_path", "jsonPath",
	})
}

func (v *WorkflowValidator) validateForEach(result *ValidationResult, fieldPath string, forEach map[string]interface{}) {
	// Validate items (required)
	if err := validateRequired(fieldPath+".items", forEach["items"]); err != nil {
		result.Errors = append(result.Errors, *err)
	}
	// items can be any type (array, string template, etc.)

	// Validate step (required)
	if err := validateRequired(fieldPath+".step", forEach["step"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else {
		if step, ok := forEach["step"].(map[string]interface{}); ok {
			v.validateStepTemplate(result, fieldPath+".step", step)
		} else {
			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath + ".step",
				Type:    ErrorTypeInvalidType,
				Message: "step must be an object",
			})
		}
	}

	// Check for unknown forEach fields
	v.checkUnknownFields(result, forEach, fieldPath, []string{"items", "step"})
}

func (v *WorkflowValidator) validateStepTemplate(result *ValidationResult, fieldPath string, step map[string]interface{}) {
	// Validate id (required)
	if err := validateRequired(fieldPath+".id", step["id"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType(fieldPath+".id", step["id"], "string"); err != nil {
		result.Errors = append(result.Errors, *err)
	}

	// Validate tool (required)
	if err := validateRequired(fieldPath+".tool", step["tool"]); err != nil {
		result.Errors = append(result.Errors, *err)
	} else if err := validateType(fieldPath+".tool", step["tool"], "string"); err != nil {
		result.Errors = append(result.Errors, *err)
	}

	// Validate args (optional)
	if args, ok := step["args"]; ok && args != nil {
		if err := validateType(fieldPath+".args", args, "object"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate allowFailure (optional) - support both snake_case and camelCase
	allowFailureField := "allow_failure"
	allowFailure := step["allow_failure"]
	if allowFailure == nil {
		allowFailureField = "allowFailure"
		allowFailure = step["allowFailure"]
	}

	if allowFailure != nil {
		if err := validateType(fieldPath+"."+allowFailureField, allowFailure, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate store (optional)
	if store, ok := step["store"]; ok && store != nil {
		if err := validateType(fieldPath+".store", store, "bool"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Validate description (optional)
	if description, ok := step["description"]; ok && description != nil {
		if err := validateType(fieldPath+".description", description, "string"); err != nil {
			result.Errors = append(result.Errors, *err)
		}
	}

	// Check for unknown step template fields
	v.checkUnknownFields(result, step, fieldPath, []string{
		"id", "tool", "args", "allow_failure", "allowFailure", "store", "description",
	})
}

func (v *WorkflowValidator) checkUnknownFields(result *ValidationResult, obj map[string]interface{}, parentPath string, knownFields []string) {
	knownMap := make(map[string]bool)
	for _, field := range knownFields {
		knownMap[field] = true
	}

	for field := range obj {
		if !knownMap[field] {
			fieldPath := field
			if parentPath != "" && parentPath != "(root)" {
				fieldPath = parentPath + "." + field
			}

			result.Errors = append(result.Errors, ValidationError{
				Field:   fieldPath,
				Type:    ErrorTypeUnknownField,
				Message: fmt.Sprintf("unknown field %q", field),
			})
		}
	}
}
