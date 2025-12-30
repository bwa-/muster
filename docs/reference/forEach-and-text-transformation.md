# ForEach Loops and Text Transformation

## Overview

Muster workflows support two powerful features for handling dynamic data:

1. **forEach Loops**: Iterate over collections and execute step templates for each item
2. **Text Transformation**: Convert unstructured text output into structured data

These features work together to enable robust data processing workflows that can handle various tool output formats.

## forEach Loops

### Basic Concept

A forEach loop expands a single workflow step into multiple executions, one for each item in a collection. This is useful for:

- Processing lists of resources (services, servers, files)
- Batch operations on multiple items
- Parallel-style operations in workflows

### Syntax

```yaml
steps:
  - id: step_name
    forEach:
      items: "{{.input.collection}}"  # or direct array
      step:
        id: iteration_step
        tool: some_tool
        args:
          item_name: "{{.item}}"
          item_property: "{{.item.property}}"
```

### Key Features

- **Template Resolution**: `items` can be a template expression (e.g., `{{.input.services}}`) or a direct array
- **Item Access**: Current item available as `{{.item}}`
- **Object Properties**: Access nested properties with `{{.item.fieldName}}`
- **Unique IDs**: Each iteration gets a unique step ID (e.g., `deploy_0`, `deploy_1`)
- **Error Handling**: Use `allow_failure` to continue on individual failures
- **Result Storage**: Each iteration can store its result independently

### Examples

#### Simple Array Iteration

```yaml
steps:
  - id: notify_users
    forEach:
      items: ["alice", "bob", "charlie"]
      step:
        id: send_notification
        tool: send_email
        args:
          to: "{{.item}}@example.com"
          subject: "System Update"
```

#### Object Array Iteration

```yaml
args:
  services:
    type: array
    required: true
    description: "List of services to deploy"

steps:
  - id: deploy_all
    forEach:
      items: "{{.input.services}}"
      step:
        id: deploy_service
        tool: kubectl_apply
        args:
          name: "{{.item.name}}"
          version: "{{.item.version}}"
          namespace: "{{.item.namespace}}"
        store: true
        allow_failure: false
```

#### Using Previous Step Results

```yaml
steps:
  - id: list_pods
    tool: kubectl_get_pods
    args:
      namespace: "production"
    store: true

  - id: check_all_pods
    forEach:
      items: "{{.results.list_pods.items}}"
      step:
        id: check_pod
        tool: kubectl_describe_pod
        args:
          name: "{{.item.metadata.name}}"
          namespace: "{{.item.metadata.namespace}}"
        store: true
```

## Text Transformation

### The Problem

Many tools return unstructured text output instead of JSON arrays. For example:

```
Indexed libraries:

- ansible
- docker
- kubernetes
- muster
- proxmox
```

This can't be directly used in a forEach loop, which expects a JSON array like `["ansible", "docker", ...]`.

### The Solution: `core_workflow_transform_text`

The `core_workflow_transform_text` tool applies a series of transformations to convert text into structured data.

### Syntax

```yaml
- id: parse_output
  tool: core_workflow_transform_text
  args:
    input: "{{.results.some_step}}"
    steps:
      - type: transformation_type
        args:
          arg1: value1
          arg2: value2
```

### Transformation Types

#### Extraction Operations

Extract specific content from text:

| Type | Description | Arguments | Example |
|------|-------------|-----------|---------|
| `extract_lines_starting_with` | Get lines starting with a prefix | `prefix` (required) | Extract bullet points |
| `extract_lines_matching` | Get lines matching a regex | `pattern` (required) | Extract lines with specific format |
| `extract_between` | Get text between two markers | `start`, `end` (both required) | Extract content between tags |
| `extract_after` | Get all text after a marker | `marker` (required) | Extract content after header |
| `extract_before` | Get all text before a marker | `marker` (required) | Extract content before footer |

**Examples:**

```yaml
# Extract bullet points
- type: extract_lines_starting_with
  args:
    prefix: "- "

# Extract service names (format: "service: status")
- type: extract_lines_matching
  args:
    pattern: "^[a-zA-Z0-9_-]+:"

# Extract content between markers
- type: extract_between
  args:
    start: "START"
    end: "END"
```

#### Modification Operations

Modify text content:

| Type | Description | Arguments | Example |
|------|-------------|-----------|---------|
| `replace` | Simple string replacement | `old` (required), `new` (required) | Replace separator |
| `replace_regex` | Regex-based replacement | `pattern` (required), `replacement` (required) | Remove unwanted parts |
| `trim` | Remove leading/trailing whitespace | None | Clean up strings |
| `trim_prefix` | Remove specific prefix | `prefix` (required) | Remove bullet markers |
| `trim_suffix` | Remove specific suffix | `suffix` (required) | Remove file extensions |
| `trim_each` | Trim each array element | None | Clean array items |
| `to_lower` | Convert to lowercase | None | Normalize case |
| `to_upper` | Convert to uppercase | None | Normalize case |

**Examples:**

```yaml
# Remove "- " from lines
- type: trim_prefix
  args:
    prefix: "- "

# Remove everything after colon
- type: replace_regex
  args:
    pattern: ":.*$"
    replacement: ""

# Normalize to lowercase
- type: to_lower
```

#### Splitting/Filtering Operations

Split text into arrays and filter:

| Type | Description | Arguments | Example |
|------|-------------|-----------|---------|
| `split` | Split by delimiter | `delimiter` (default: ",") | Split CSV |
| `split_lines` | Split by newlines | None | Convert to line array |
| `remove_empty` | Remove empty strings | None | Clean up array |
| `remove_duplicates` | Remove duplicate strings | None | Deduplicate |

**Examples:**

```yaml
# Split CSV
- type: split
  args:
    delimiter: ","

# Split into lines
- type: split_lines

# Remove empty entries
- type: remove_empty
```

### Complete Transformation Example

Parse a markdown list into a JSON array:

```yaml
steps:
  - id: list_items
    tool: some_tool
    store: true
    # Returns:
    # "Available items:\n\n- item1\n- item2\n- item3"

  - id: parse_items
    tool: core_workflow_transform_text
    args:
      input: "{{.results.list_items}}"
      steps:
        # 1. Extract lines starting with "- "
        - type: extract_lines_starting_with
          args:
            prefix: "- "
        # 2. Remove the "- " prefix
        - type: trim_prefix
          args:
            prefix: "- "
        # 3. Trim whitespace
        - type: trim_each
        # 4. Remove empty entries
        - type: remove_empty
    store: true
    # Result: ["item1", "item2", "item3"]
```

## Combining forEach and Text Transformation

The real power comes from combining these features:

```yaml
steps:
  # 1. Get unstructured data
  - id: list_libraries
    tool: x_docs-mcp-server_list_libraries
    store: true
    # Returns markdown list

  # 2. Transform to structured array
  - id: parse_libraries
    tool: core_workflow_transform_text
    args:
      input: "{{.results.list_libraries}}"
      steps:
        - type: extract_lines_starting_with
          args:
            prefix: "- "
        - type: trim_prefix
          args:
            prefix: "- "
        - type: trim_each
        - type: remove_empty
    store: true
    # Returns: ["lib1", "lib2", "lib3"]

  # 3. Iterate over structured array
  - id: search_all_docs
    forEach:
      items: "{{.results.parse_libraries}}"
      step:
        id: search_library
        tool: x_docs-mcp-server_search_docs
        args:
          library: "{{.item}}"
          query: "{{.input.research_query}}"
        store: true
        allow_failure: true
```

## Data Flow Types

### Type Tracking

The text transformer tracks data types through the pipeline:

- **string**: Text data
- **array**: Array of strings

Some operations change the type:

```yaml
steps:
  # Input: string
  - type: split_lines  # Outputs: array
  - type: trim_each    # Still: array
  - type: remove_empty # Still: array
```

### Operations by Input Type

**String Operations** (input must be string):
- All extraction operations
- `split`, `split_lines`
- Modification operations (unless applied to array)

**Array Operations** (input must be array):
- `trim_each`
- `remove_empty`
- `remove_duplicates`

**Both** (works on string or array):
- `replace`, `replace_regex`
- `trim`, `trim_prefix`, `trim_suffix`
- `to_lower`, `to_upper`

## Best Practices

### 1. Store Intermediate Results

```yaml
- id: parse_data
  tool: core_workflow_transform_text
  args:
    input: "{{.results.raw_data}}"
    steps: [...]
  store: true  # ← Important for debugging
```

### 2. Use allow_failure in forEach

```yaml
forEach:
  items: "{{.results.servers}}"
  step:
    id: check_server
    tool: ping_server
    args:
      host: "{{.item}}"
    allow_failure: true  # ← Continue if one fails
```

### 3. Validate Transformation Output

Test your transformation pipeline separately before using it in forEach:

```yaml
# Test step (can be removed later)
- id: test_transform
  tool: core_workflow_transform_text
  args:
    input: "sample data"
    steps:
      - type: split
        args:
          delimiter: ","
  store: true
```

### 4. Chain Transformations Logically

```yaml
steps:
  - type: extract_lines_matching  # 1. Get relevant lines
    args:
      pattern: "^service:"
  - type: replace_regex           # 2. Clean up format
    args:
      pattern: "service:\\s*"
      replacement: ""
  - type: trim_each               # 3. Clean whitespace
  - type: remove_empty            # 4. Remove junk
  - type: remove_duplicates       # 5. Deduplicate
```

### 5. Handle Edge Cases

```yaml
# Always remove empty entries after extraction
- type: extract_lines_starting_with
  args:
    prefix: "- "
- type: trim_prefix
  args:
    prefix: "- "
- type: remove_empty  # ← Handles blank lines
```

## Troubleshooting

### forEach: "items must be an array"

**Problem**: The `items` value is not an array.

**Solution**: Check that your template expression resolves to an array. Use text transformation if needed:

```yaml
# Before forEach, parse the data:
- id: parse_list
  tool: core_workflow_transform_text
  args:
    input: "{{.results.some_step}}"
    steps:
      - type: split_lines
  store: true

- id: process_items
  forEach:
    items: "{{.results.parse_list}}"  # Now it's an array
```

### Transformation: "operation requires string input"

**Problem**: You're trying to use a string-only operation on an array.

**Solution**: Check the operation's input requirements. Use array operations or convert back to string:

```yaml
# Wrong: trim expects string, but we have an array
- type: split_lines  # Creates array
- type: trim         # ERROR!

# Right: use trim_each for arrays
- type: split_lines
- type: trim_each    # OK!
```

### forEach: Empty iterations

**Problem**: forEach runs but executes 0 times.

**Solution**: Verify the array is not empty. Add a condition or check:

```yaml
- id: check_has_items
  tool: core_workflow_transform_text
  args:
    input: "{{.results.items}}"
    steps:
      - type: remove_empty
  store: true

# Only run if we have items
- id: process
  condition:
    from_step: check_has_items
    expect:
      success: true
  forEach:
    items: "{{.results.check_has_items}}"
```

## Performance Considerations

### forEach Iterations

Each forEach iteration executes sequentially. For large collections:

- Consider batching operations at the tool level
- Use `allow_failure` judiciously (affects execution time)
- Store results only when needed (`store: false` by default)

### Text Transformations

Transformations are fast but sequential:

- Minimize transformation steps when possible
- Combine operations where logical (e.g., use `split` instead of multiple `replace` calls)
- Use specific extractions before general ones (faster)

## See Also

- [Workflow Creation Guide](../how-to/workflow-creation.md)
- [Template Expressions](configuration-examples.md#template-expressions)
- [Workflow Examples](../../examples/)
