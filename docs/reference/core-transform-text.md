# core_workflow_transform_text Tool Reference

## Overview

The `core_workflow_transform_text` tool provides a pipeline-based approach to transforming unstructured text into structured data. It's particularly useful for parsing tool outputs that don't return JSON arrays.

## Tool Information

- **Name**: `core_workflow_transform_text` (prefixed as `core_workflow_transform_text` in aggregator)
- **Category**: Utility / Text Processing
- **Input**: String
- **Output**: String or Array (JSON)

## Usage

```json
{
  "name": "core_workflow_transform_text",
  "arguments": {
    "input": "text to transform",
    "steps": [
      {
        "type": "transformation_type",
        "args": {
          "arg1": "value1"
        }
      }
    ]
  }
}
```

## Transformation Reference

### Extraction Operations

#### extract_lines_starting_with
Extracts lines that start with a specific prefix.

**Arguments:**
- `prefix` (string, required): The prefix to match

**Input**: string  
**Output**: array

**Example:**
```json
{
  "type": "extract_lines_starting_with",
  "args": { "prefix": "- " }
}
```

---

#### extract_lines_matching
Extracts lines matching a regular expression.

**Arguments:**
- `pattern` (string, required): Regular expression pattern

**Input**: string  
**Output**: array

**Example:**
```json
{
  "type": "extract_lines_matching",
  "args": { "pattern": "^ERROR:" }
}
```

---

#### extract_between
Extracts text between two markers.

**Arguments:**
- `start` (string, required): Start marker
- `end` (string, required): End marker

**Input**: string  
**Output**: string

**Example:**
```json
{
  "type": "extract_between",
  "args": { "start": "[START]", "end": "[END]" }
}
```

---

#### extract_after
Extracts all text after a marker.

**Arguments:**
- `marker` (string, required): The marker to search for

**Input**: string  
**Output**: string

**Example:**
```json
{
  "type": "extract_after",
  "args": { "marker": "Results:" }
}
```

---

#### extract_before
Extracts all text before a marker.

**Arguments:**
- `marker` (string, required): The marker to search for

**Input**: string  
**Output**: string

**Example:**
```json
{
  "type": "extract_before",
  "args": { "marker": "---" }
}
```

---

### Modification Operations

#### replace
Performs simple string replacement.

**Arguments:**
- `old` (string, required): String to find
- `new` (string, required): String to replace with

**Input**: string or array  
**Output**: same as input

**Example:**
```json
{
  "type": "replace",
  "args": { "old": " and ", "new": ", " }
}
```

---

#### replace_regex
Performs regex-based replacement.

**Arguments:**
- `pattern` (string, required): Regular expression pattern
- `replacement` (string, required): Replacement string

**Input**: string or array  
**Output**: same as input

**Example:**
```json
{
  "type": "replace_regex",
  "args": { "pattern": "\\s+", "replacement": " " }
}
```

---

#### trim
Removes leading and trailing whitespace.

**Arguments**: None

**Input**: string or array  
**Output**: same as input

**Example:**
```json
{
  "type": "trim"
}
```

---

#### trim_prefix
Removes a specific prefix from text.

**Arguments:**
- `prefix` (string, required): Prefix to remove

**Input**: string or array  
**Output**: same as input

**Example:**
```json
{
  "type": "trim_prefix",
  "args": { "prefix": "- " }
}
```

---

#### trim_suffix
Removes a specific suffix from text.

**Arguments:**
- `suffix` (string, required): Suffix to remove

**Input**: string or array  
**Output**: same as input

**Example:**
```json
{
  "type": "trim_suffix",
  "args": { "suffix": ".txt" }
}
```

---

#### trim_each
Trims whitespace from each element in an array.

**Arguments**: None

**Input**: array  
**Output**: array

**Example:**
```json
{
  "type": "trim_each"
}
```

---

#### to_lower
Converts text to lowercase.

**Arguments**: None

**Input**: string or array  
**Output**: same as input

**Example:**
```json
{
  "type": "to_lower"
}
```

---

#### to_upper
Converts text to uppercase.

**Arguments**: None

**Input**: string or array  
**Output**: same as input

**Example:**
```json
{
  "type": "to_upper"
}
```

---

### Splitting/Filtering Operations

#### split
Splits a string by a delimiter.

**Arguments:**
- `delimiter` (string, optional, default: ","): Delimiter to split on

**Input**: string  
**Output**: array

**Example:**
```json
{
  "type": "split",
  "args": { "delimiter": "," }
}
```

---

#### split_lines
Splits text by newlines.

**Arguments**: None

**Input**: string  
**Output**: array

**Example:**
```json
{
  "type": "split_lines"
}
```

---

#### remove_empty
Removes empty strings from an array.

**Arguments**: None

**Input**: array  
**Output**: array

**Example:**
```json
{
  "type": "remove_empty"
}
```

---

#### remove_duplicates
Removes duplicate strings from an array.

**Arguments**: None

**Input**: array  
**Output**: array

**Example:**
```json
{
  "type": "remove_duplicates"
}
```

---

## Common Patterns

### Parse Markdown Bullet List

```yaml
tool: core_workflow_transform_text
args:
  input: "{{.results.list_output}}"
  steps:
    - type: extract_lines_starting_with
      args:
        prefix: "- "
    - type: trim_prefix
      args:
        prefix: "- "
    - type: trim_each
    - type: remove_empty
```

### Parse CSV String

```yaml
tool: core_workflow_transform_text
args:
  input: "item1, item2, item3"
  steps:
    - type: split
      args:
        delimiter: ","
    - type: trim_each
    - type: remove_empty
```

### Extract Service Names from Status Output

```yaml
tool: core_workflow_transform_text
args:
  input: "{{.results.service_status}}"
  steps:
    - type: split_lines
    - type: extract_lines_matching
      args:
        pattern: "^[a-zA-Z0-9_-]+:"
    - type: replace_regex
      args:
        pattern: ":.*$"
        replacement: ""
    - type: trim_each
    - type: remove_empty
```

### Clean and Normalize List

```yaml
tool: core_workflow_transform_text
args:
  input: "{{.results.raw_list}}"
  steps:
    - type: split_lines
    - type: trim_each
    - type: to_lower
    - type: remove_empty
    - type: remove_duplicates
```

## Error Handling

### Common Errors

**"prefix argument is required"**
- Cause: Missing required argument
- Solution: Add the required argument to the `args` object

**"operation requires string input, got array"**
- Cause: Using a string-only operation on an array
- Solution: Use the array equivalent (e.g., `trim_each` instead of `trim`)

**"invalid regex pattern"**
- Cause: Malformed regular expression
- Solution: Test your regex pattern and escape special characters

## Type Compatibility Matrix

| Operation | String Input | Array Input | Output Type |
|-----------|--------------|-------------|-------------|
| extract_* | ✅ | ❌ | varies |
| replace | ✅ | ✅ | same |
| replace_regex | ✅ | ✅ | same |
| trim | ✅ | ✅ | same |
| trim_prefix | ✅ | ✅ | same |
| trim_suffix | ✅ | ✅ | same |
| trim_each | ❌ | ✅ | array |
| to_lower | ✅ | ✅ | same |
| to_upper | ✅ | ✅ | same |
| split | ✅ | ❌ | array |
| split_lines | ✅ | ❌ | array |
| remove_empty | ❌ | ✅ | array |
| remove_duplicates | ❌ | ✅ | array |

## See Also

- [ForEach and Text Transformation Guide](forEach-and-text-transformation.md)
- [Workflow Examples](../../examples/workflow-foreach-text-transform-example.yaml)
- [MCP Tools Reference](mcp-tools.md)
