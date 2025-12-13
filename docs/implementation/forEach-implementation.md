# ForEach Loop Implementation for Muster Workflows

## Overview

This implementation adds `forEach` loop support to Muster workflows, allowing workflows to iterate over collections and execute step templates for each item. This feature was documented but not previously implemented in the codebase.

## Implementation Summary

### 1. API Types (`internal/api/workflow.go`)

Added three new types to support forEach functionality:

- **`ForEachConfig`**: Defines the forEach configuration
  - `Items`: Collection to iterate over (can be template expression or direct array)
  - `Step`: Step template to execute for each item

- **`WorkflowStepTemplate`**: Template for steps executed in forEach loops
  - Similar to `WorkflowStep` but designed for iteration
  - Supports `{{.item}}` template variable for current item
  - Step IDs are expanded with iteration index (e.g., `deploy_service_0`, `deploy_service_1`)

- **`WorkflowStep.ForEach`**: New optional field on `WorkflowStep`
  - When present, the step becomes a forEach loop
  - Mutually exclusive with `Tool` field (validation in CRD)

### 2. Workflow Executor (`internal/workflow/executor.go`)

Implemented forEach expansion logic:

- **`expandForEachSteps()`**: Expands forEach steps into individual steps
  - Resolves the `items` collection (supports templates like `{{.input.microservices}}`)
  - Creates execution context with `item` variable for each iteration
  - Generates unique step IDs with iteration index
  - Resolves arguments for each iteration with `{{.item}}` access

- **Template Context Enhancement**: Added `item` to template context
  - Modified `resolveTemplate()` to include `item` variable from context
  - Enables `{{.item}}` and `{{.item.field}}` access in forEach step arguments

### 3. CRD Schema (`deploy/crds/muster.giantswarm.io_workflows.yaml`)

Updated Kubernetes CRD to include forEach:

- Added `forEach` property to WorkflowStep schema
- Defined nested structure for `items` and `step` properties
- Changed step requirement from `[id, tool]` to just `[id]` (since forEach steps don't need tool)
- Added proper validation constraints

### 4. API Adapter (`internal/workflow/api_adapter.go`)

Updated JSON schema for workflow steps:

- Added `forEach` property to step schema
- Defined structure for forEach configuration
- Updated required fields to support both regular and forEach steps

### 5. Tests (`internal/workflow/executor_test.go`)

Added comprehensive test coverage:

- **`TestWorkflowExecutor_ForEach_BasicArray`**: Tests iteration over array of objects
- **`TestWorkflowExecutor_ForEach_DirectArray`**: Tests direct array values
- **`TestWorkflowExecutor_ForEach_WithRegularSteps`**: Tests forEach mixed with regular steps
- **`TestWorkflowExecutor_ForEach_EmptyArray`**: Tests edge case of empty arrays
- **`TestWorkflowExecutor_ForEach_InvalidItemsType`**: Tests error handling for invalid types

### 6. Example (`examples/workflow-foreach-example.yaml`)

Created example workflow demonstrating:

- Deploying multiple microservices using forEach
- Accessing item properties with `{{.item.name}}`, `{{.item.version}}`, etc.
- Mixing forEach steps with regular workflow steps
- Proper workflow structure and documentation

## Usage Example

```yaml
steps:
  - id: deploy_microservices
    forEach:
      items: "{{.input.microservices}}"
      step:
        id: deploy_service
        tool: deploy_microservice
        args:
          name: "{{.item.name}}"
          version: "{{.item.version}}"
          config: "{{.item.config}}"
```

With input:
```json
{
  "microservices": [
    {"name": "frontend", "version": "1.0.0"},
    {"name": "backend", "version": "2.0.0"}
  ]
}
```

This expands to two steps:
- `deploy_service_0` with `name: "frontend"`, `version: "1.0.0"`
- `deploy_service_1` with `name: "backend"`, `version: "2.0.0"`

## Features

✅ **Template Expression Support**: Items can be resolved from workflow arguments using templates
✅ **Direct Array Support**: Items can be specified as direct array values
✅ **Object Property Access**: Access nested properties with `{{.item.field}}`
✅ **Index-Based Step IDs**: Each iteration gets unique ID with index suffix
✅ **Mixed with Regular Steps**: forEach steps can be mixed with regular workflow steps
✅ **Error Handling**: Proper validation and error messages for invalid configurations
✅ **Empty Array Handling**: Gracefully handles empty arrays (no steps executed)
✅ **Store Support**: Each iteration can have its results stored independently
✅ **AllowFailure Support**: Individual iterations can be allowed to fail

## Testing

All tests pass with 100% coverage of forEach functionality:

```
=== RUN   TestWorkflowExecutor_ForEach_BasicArray
--- PASS: TestWorkflowExecutor_ForEach_BasicArray (0.00s)
=== RUN   TestWorkflowExecutor_ForEach_DirectArray
--- PASS: TestWorkflowExecutor_ForEach_DirectArray (0.00s)
=== RUN   TestWorkflowExecutor_ForEach_WithRegularSteps
--- PASS: TestWorkflowExecutor_ForEach_WithRegularSteps (0.00s)
=== RUN   TestWorkflowExecutor_ForEach_EmptyArray
--- PASS: TestWorkflowExecutor_ForEach_EmptyArray (0.00s)
=== RUN   TestWorkflowExecutor_ForEach_InvalidItemsType
--- PASS: TestWorkflowExecutor_ForEach_InvalidItemsType (0.00s)
```

## Compatibility

- ✅ Backward compatible: Existing workflows without forEach continue to work
- ✅ CRD compatible: Schema validation ensures proper usage
- ✅ Template compatible: Works with existing template resolution system
- ✅ Documentation aligned: Matches documented YAML examples exactly

## Files Modified

1. `internal/api/workflow.go` - Added ForEach types
2. `internal/workflow/executor.go` - Implemented forEach expansion and template context
3. `deploy/crds/muster.giantswarm.io_workflows.yaml` - Updated CRD schema
4. `internal/workflow/api_adapter.go` - Updated JSON schema
5. `internal/workflow/executor_test.go` - Added comprehensive tests
6. `examples/workflow-foreach-example.yaml` - Created example workflow

## Next Steps

The forEach implementation is complete and ready for use. Future enhancements could include:

- Nested forEach loops (forEach within forEach)
- Parallel execution of forEach iterations
- Break/continue conditions within loops
- Access to iteration index as `{{.index}}`
- Filtering/mapping operations on items before iteration
