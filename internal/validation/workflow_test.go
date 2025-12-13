package validation

import (
	"testing"
)

func TestWorkflowValidator_Validate_ValidConfig(t *testing.T) {
	validator := NewWorkflowValidator()

	validYAML := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test-workflow
  namespace: default
spec:
  description: "Test workflow"
  args:
    app_name:
      type: string
      required: true
      description: "Application name"
    replicas:
      type: integer
      default: 3
  steps:
    - id: build
      tool: docker_build
      args:
        name: "{{.app_name}}"
      store: true
    - id: deploy
      tool: kubectl_apply
      args:
        image: "{{.results.build.image}}"
        replicas: "{{.replicas}}"
      condition:
        fromStep: build
        expect:
          success: true
      allowFailure: false
`

	result := validator.Validate([]byte(validYAML), "test.yaml")

	if !result.IsValid() {
		t.Errorf("Expected valid config, got errors: %v", result.Errors)
	}
}

func TestWorkflowValidator_Validate_ValidForEach(t *testing.T) {
	validator := NewWorkflowValidator()

	validYAML := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test-workflow
spec:
  args:
    services:
      type: array
      required: true
  steps:
    - id: deploy_all
      forEach:
        items: "{{.services}}"
        step:
          id: deploy_service
          tool: kubectl_apply
          args:
            name: "{{.item.name}}"
          store: true
`

	result := validator.Validate([]byte(validYAML), "test.yaml")

	if !result.IsValid() {
		t.Errorf("Expected valid forEach config, got errors: %v", result.Errors)
	}
}

func TestWorkflowValidator_Validate_MissingRequired(t *testing.T) {
	validator := NewWorkflowValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
	}{
		{
			name: "missing apiVersion",
			yaml: `
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - id: step1
      tool: test
`,
			expectedField: "apiVersion",
		},
		{
			name: "missing kind",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
metadata:
  name: test
spec:
  steps:
    - id: step1
      tool: test
`,
			expectedField: "kind",
		},
		{
			name: "missing metadata",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
spec:
  steps:
    - id: step1
      tool: test
`,
			expectedField: "metadata",
		},
		{
			name: "missing metadata.name",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  namespace: default
spec:
  steps:
    - id: step1
      tool: test
`,
			expectedField: "metadata.name",
		},
		{
			name: "missing spec",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
`,
			expectedField: "spec",
		},
		{
			name: "missing spec.steps",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  description: "test"
`,
			expectedField: "spec.steps",
		},
		{
			name: "missing step.id",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - tool: test
`,
			expectedField: "spec.steps[0].id",
		},
		{
			name: "missing step.tool",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - id: step1
`,
			expectedField: "spec.steps[0].tool",
		},
		{
			name: "missing arg.type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  args:
    myarg:
      required: true
  steps:
    - id: step1
      tool: test
`,
			expectedField: "spec.args.myarg.type",
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
				if err.Field == tt.expectedField && err.Type == ErrorTypeRequired {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected required error for field %q, got errors: %v",
					tt.expectedField, result.Errors)
			}
		})
	}
}

func TestWorkflowValidator_Validate_InvalidValues(t *testing.T) {
	validator := NewWorkflowValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
		expectedType  ErrorType
	}{
		{
			name: "invalid kind",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: WorkflowWrong
metadata:
  name: test
spec:
  steps:
    - id: step1
      tool: test
`,
			expectedField: "kind",
			expectedType:  ErrorTypeInvalidValue,
		},
		{
			name: "invalid arg type",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  args:
    myarg:
      type: int
  steps:
    - id: step1
      tool: test
`,
			expectedField: "spec.args.myarg.type",
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

func TestWorkflowValidator_Validate_DuplicateStepIDs(t *testing.T) {
	validator := NewWorkflowValidator()

	yaml := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - id: build
      tool: docker_build
    - id: build
      tool: docker_push
`

	result := validator.Validate([]byte(yaml), "test.yaml")

	if result.IsValid() {
		t.Error("Expected validation error for duplicate step ID")
		return
	}

	found := false
	for _, err := range result.Errors {
		if err.Type == ErrorTypeDuplicate && err.Field == "spec.steps[1].id" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected duplicate error for step ID, got errors: %v", result.Errors)
	}
}

func TestWorkflowValidator_Validate_ConflictingFields(t *testing.T) {
	validator := NewWorkflowValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
	}{
		{
			name: "tool and forEach conflict",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - id: step1
      tool: test
      forEach:
        items: []
        step:
          id: substep
          tool: subtool
`,
			expectedField: "spec.steps[0]",
		},
		{
			name: "condition with both tool and fromStep",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - id: step1
      tool: test
      condition:
        tool: check_tool
        fromStep: previous_step
`,
			expectedField: "spec.steps[0].condition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate([]byte(tt.yaml), "test.yaml")

			if result.IsValid() {
				t.Error("Expected validation error for conflicting fields")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Type == ErrorTypeConflict && err.Field == tt.expectedField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Expected conflict error for field %q, got errors: %v",
					tt.expectedField, result.Errors)
			}
		})
	}
}

func TestWorkflowValidator_Validate_ConstraintViolations(t *testing.T) {
	validator := NewWorkflowValidator()

	tests := []struct {
		name          string
		yaml          string
		expectedField string
	}{
		{
			name: "description too long",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  description: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam, eaque ipsa quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt explicabo. Nemo enim ipsam voluptatem quia voluptas sit aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos qui ratione voluptatem sequi nesciunt. Neque porro quisquam est, qui dolorem ipsum quia dolor sit amet, consectetur, adipisci velit, sed quia non numquam eius modi tempora incidunt ut labore et dolore magnam aliquam quaerat voluptatem extra text to exceed limit."
  steps:
    - id: step1
      tool: test
`,
			expectedField: "spec.description",
		},
		{
			name: "arg description too long",
			yaml: `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  args:
    myarg:
      type: string
      description: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. Extra text to make this exceed 500 characters limit for validation testing purposes of long descriptions."
  steps:
    - id: step1
      tool: test
`,
			expectedField: "spec.args.myarg.description",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate([]byte(tt.yaml), "test.yaml")

			if result.IsValid() {
				t.Error("Expected validation error for constraint violation")
				return
			}

			found := false
			for _, err := range result.Errors {
				if err.Type == ErrorTypeConstraintViolation && err.Field == tt.expectedField {
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

func TestWorkflowValidator_Validate_ValidCondition(t *testing.T) {
	validator := NewWorkflowValidator()

	yaml := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - id: step1
      tool: test1
      store: true
    - id: step2
      tool: test2
      condition:
        fromStep: step1
        expect:
          success: true
          jsonPath:
            status: "completed"
`

	result := validator.Validate([]byte(yaml), "test.yaml")

	if !result.IsValid() {
		t.Errorf("Expected valid condition config, got errors: %v", result.Errors)
	}
}

func TestWorkflowValidator_Validate_MinimalConfig(t *testing.T) {
	validator := NewWorkflowValidator()

	yaml := `
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: test
spec:
  steps:
    - id: step1
      tool: test
`

	result := validator.Validate([]byte(yaml), "test.yaml")

	if !result.IsValid() {
		t.Errorf("Expected valid minimal config, got errors: %v", result.Errors)
	}
}
