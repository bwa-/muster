# YAML Configuration Validation

Muster performs comprehensive validation of all YAML configuration files to ensure correctness and catch errors early. This document describes the validation rules, common errors, and troubleshooting steps.

## Overview

Validation happens automatically when:
- Loading configuration files via the filesystem client
- Application startup (validates all MCPServers and Workflows)
- Creating or updating resources

Invalid configurations are rejected with detailed error messages indicating:
- The file path where the error occurred
- The field that failed validation
- The specific validation rule that was violated
- Suggestions for fixing the error (when applicable)

## Validation Rules

### MCPServer Validation

MCPServer configurations must conform to the following rules:

#### Top-Level Fields

| Field | Required | Validation |
|-------|----------|------------|
| `apiVersion` | Yes | Must be `muster.giantswarm.io/v1alpha1` |
| `kind` | Yes | Must be `MCPServer` |
| `metadata.name` | Yes | Must be a non-empty string |

#### Spec Fields

| Field | Required | Validation |
|-------|----------|------------|
| `spec.type` | Yes | Must be one of: `localCommand` |
| `spec.command` | Conditional | Required when `type` is `localCommand`. Must be a non-empty array of strings |
| `spec.toolPrefix` | No | If provided, must match pattern: `^[a-zA-Z][a-zA-Z0-9_-]*$` (alphanumeric, underscore, hyphen; must start with letter) |
| `spec.description` | No | If provided, maximum length is 500 characters |
| `spec.env` | No | If provided, must be a map of string key-value pairs |
| `spec.args` | No | If provided, must be an array of strings |

#### Unknown Fields

Any fields not explicitly defined in the CRD schema will be rejected with an error message listing the unknown field names.

#### Example Valid MCPServer

```yaml
apiVersion: muster.giantswarm.io/v1alpha1
kind: MCPServer
metadata:
  name: filesystem-server
spec:
  type: localCommand
  command:
    - npx
    - "@modelcontextprotocol/server-filesystem"
    - "/tmp"
  toolPrefix: fs
  description: "Access local filesystem operations"
  env:
    LOG_LEVEL: "info"
```

### Workflow Validation

Workflow configurations must conform to the following rules:

#### Top-Level Fields

| Field | Required | Validation |
|-------|----------|------------|
| `apiVersion` | Yes | Must be `muster.giantswarm.io/v1alpha1` |
| `kind` | Yes | Must be `Workflow` |
| `metadata.name` | Yes | Must be a non-empty string |

#### Spec Fields

| Field | Required | Validation |
|-------|----------|------------|
| `spec.steps` | Yes | Must be a non-empty array of step definitions |
| `spec.description` | No | If provided, maximum length is 1000 characters |
| `spec.args` | No | If provided, must be a map of argument definitions |

#### Argument Definition (`spec.args.<name>`)

| Field | Required | Validation |
|-------|----------|------------|
| `type` | Yes | Must be one of: `string`, `integer`, `boolean`, `number`, `object`, `array` |
| `required` | No | If provided, must be a boolean |
| `default` | No | No validation |
| `description` | No | If provided, maximum length is 500 characters |

#### Step Definition (`spec.steps[]`)

| Field | Required | Validation |
|-------|----------|------------|
| `id` | Yes | Must be unique within the workflow |
| `tool` | Conditional | Required if `forEach` is not present. Cannot be used together with `forEach` |
| `forEach` | Conditional | Required if `tool` is not present. Cannot be used together with `tool` |
| `args` | No | If provided, must be a map |
| `store` | No | If provided, must be a boolean |
| `allowFailure` | No | If provided, must be a boolean |
| `condition` | No | If provided, must be a valid condition definition |

#### Condition Definition (`spec.steps[].condition`)

| Field | Required | Validation |
|-------|----------|------------|
| `tool` | Conditional | Required if `fromStep` is not present. Cannot be used together with `fromStep` |
| `fromStep` | Conditional | Required if `tool` is not present. Cannot be used together with `tool` |
| `expect` | No | If provided, must be a valid expectation definition |
| `expectNot` | No | If provided, must be a valid expectation definition |

#### Expectation Definition

| Field | Required | Validation |
|-------|----------|------------|
| `success` | No | If provided, must be a boolean |
| `result` | No | If provided, must be a map with `jsonPath` and `value` |
| `result.jsonPath` | Conditional | Required if `result` is present |
| `result.value` | No | No validation |

#### ForEach Definition (`spec.steps[].forEach`)

| Field | Required | Validation |
|-------|----------|------------|
| `items` | Yes | Must be a non-empty array or string reference |
| `stepTemplate` | Yes | Must be a valid step definition (without `id`) |

#### Unknown Fields

Any fields not explicitly defined in the CRD schema will be rejected with an error message.

#### Example Valid Workflow

```yaml
apiVersion: muster.giantswarm.io/v1alpha1
kind: Workflow
metadata:
  name: deploy-application
spec:
  description: "Deploy application to production"
  args:
    app_name:
      type: string
      required: true
      description: "Name of the application"
    replicas:
      type: integer
      default: 3
      description: "Number of replicas"
  steps:
    - id: build
      tool: docker_build
      args:
        name: "{{.app_name}}"
      store: true
    
    - id: deploy
      tool: k8s_deploy
      args:
        image: "{{.results.build.image}}"
        replicas: "{{.replicas}}"
      condition:
        fromStep: build
        expect:
          success: true
```

## Common Validation Errors

### MCPServer Errors

#### Invalid Type

```
spec.type: invalid value "invalidType", must be one of: [localCommand]
```

**Fix**: Change `spec.type` to `localCommand`.

#### Missing Command

```
spec.command: required field missing when type is "localCommand"
```

**Fix**: Add a `command` array to the spec:
```yaml
spec:
  type: localCommand
  command:
    - npx
    - "@modelcontextprotocol/server-example"
```

#### Invalid Tool Prefix Pattern

```
spec.toolPrefix: invalid format "123invalid", must match pattern ^[a-zA-Z][a-zA-Z0-9_-]*$
```

**Fix**: Tool prefix must start with a letter:
```yaml
spec:
  toolPrefix: valid_prefix
```

#### Description Too Long

```
spec.description: constraint violation, value exceeds maximum length of 500
```

**Fix**: Shorten the description to 500 characters or less.

### Workflow Errors

#### Missing Steps

```
spec.steps: required field missing
```

**Fix**: Add at least one step to the workflow:
```yaml
spec:
  steps:
    - id: first_step
      tool: example_tool
```

#### Duplicate Step IDs

```
spec.steps[1].id: duplicate step ID "build_image"
```

**Fix**: Ensure each step has a unique ID:
```yaml
steps:
  - id: build_image_1
    tool: docker_build
  - id: build_image_2  # Changed from build_image
    tool: docker_build
```

#### Tool and ForEach Conflict

```
spec.steps[0]: invalid configuration, step cannot have both 'tool' and 'forEach'
```

**Fix**: Use either `tool` or `forEach`, not both:
```yaml
# Option 1: Use tool
- id: single_task
  tool: example_tool

# Option 2: Use forEach
- id: multiple_tasks
  forEach:
    items: ["a", "b", "c"]
    stepTemplate:
      tool: example_tool
```

#### Invalid Argument Type

```
spec.args.count.type: invalid value "num", must be one of: [string integer boolean number object array]
```

**Fix**: Use a valid type:
```yaml
args:
  count:
    type: integer  # Changed from "num"
```

#### Condition Tool and FromStep Conflict

```
spec.steps[1].condition: invalid configuration, condition cannot have both 'tool' and 'fromStep'
```

**Fix**: Use either `tool` or `fromStep`:
```yaml
# Option 1: Check another step's result
condition:
  fromStep: previous_step
  expect:
    success: true

# Option 2: Run a tool as condition
condition:
  tool: check_status
  expect:
    success: true
```

## Validation Behavior

### Startup Validation

During application startup (`muster serve`), all MCPServer and Workflow configuration files are loaded and validated:

1. **List Phase**: The application reads all `.yaml` files in the `mcpservers/` and `workflows/` directories
2. **Individual Validation**: Each file is validated using its respective validator
3. **Error Logging**: Invalid files are logged at ERROR level with detailed error messages
4. **Graceful Degradation**: The application continues to start even if some files are invalid

**Example startup logs:**
```
time=2025-12-13T21:42:26 level=INFO msg="Validating configuration files..."
time=2025-12-13T21:42:26 level=ERROR msg="Failed to load MCPServer invalid.yaml" 
  error="MCPServer validation failed:\n/path/mcpservers/invalid.yaml:\n  - spec.type: invalid value"
time=2025-12-13T21:42:26 level=INFO msg="Validated 5 MCPServer(s)"
time=2025-12-13T21:42:26 level=INFO msg="Validated 3 Workflow(s)"
time=2025-12-13T21:42:26 level=INFO msg="Configuration validation successful"
```

### Runtime Validation

When loading individual files during runtime (e.g., via API calls), validation is strict:
- Invalid files cause the operation to fail immediately
- Detailed error messages are returned to the caller
- No partial success - the entire file must be valid

## Troubleshooting

### Finding Validation Errors

1. **Check startup logs**: Look for ERROR level messages mentioning "validation failed"
2. **Examine the error message**: It includes:
   - File path
   - Field name
   - Validation rule violated
   - Sometimes a suggestion for fixing

### Fixing Validation Errors

1. **Read the error message carefully**: It tells you exactly what's wrong
2. **Check the field type**: Make sure you're using the correct type (string, integer, boolean, etc.)
3. **Verify required fields**: Ensure all required fields are present
4. **Check enums**: For fields with limited values (like `type`), use only allowed values
5. **Test the fix**: Save the file and check logs to confirm it loads successfully

### Testing Configuration Files

To test a configuration file without starting the full application:

```bash
# Option 1: Use muster serve with a test directory
muster serve --config-path /path/to/test/config

# Option 2: Check logs during startup
muster serve --silent=false | grep -E "(Validat|ERROR)"
```

### Validation vs Schema Validation

Muster performs two levels of validation:

1. **YAML Schema Validation** (this document): Validates structure, types, and constraints
2. **CRD Schema Validation** (Kubernetes only): Additional validation when using Kubernetes CRDs

Both levels must pass for a configuration to be valid.

## Extending Validation

The validation framework is extensible. To add validation for a new resource type:

1. Create a new validator in `internal/validation/`
2. Implement the `Validator` interface
3. Add validation rules using helper functions from `validator.go`
4. Integrate into the filesystem client's `Get<ResourceType>` method
5. Add unit tests in `internal/validation/<resource>_test.go`

For detailed implementation guidance, see the existing validators:
- `internal/validation/mcpserver.go` - MCPServer validation
- `internal/validation/workflow.go` - Workflow validation with nested structures
