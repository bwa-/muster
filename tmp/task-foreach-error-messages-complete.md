# Task: Complete forEach Error Message Improvements

## Problem
Despite recent improvements to forEach validation, error messages for failed forEach steps still show:
- Empty tool name: `"tool": ""`
- Missing context about what tool was attempted
- No information about available tools to help debugging

See `tmp/err.json` for example of current broken error format.

## Current State

### What Was Fixed
- ✅ `getMissingTools()` now checks forEach step tools (`step.ForEach.Step.Tool`)
- ✅ `isWorkflowAvailable()` returns detailed JSON with all available tools
- ✅ Validation order fixed (deferred until after MCP discovery)
- ✅ `expandForEachSteps()` enhanced to show tool name in some error cases

### What's Still Broken
Looking at the error in `tmp/err.json`:
```json
{
  "error": "tool  is not available",
  "tool": "",
  "status": "failed"
}
```

This happens when:
1. forEach expansion succeeds (items are resolved)
2. Individual step execution fails (tool not found during CallTool)
3. Error doesn't include the template expression from `step.ForEach.Step.Tool`

## Root Cause
The error occurs in `WorkflowExecutor.executeStep()` when calling the tool, NOT during forEach expansion. The step object at that point has the tool name from the template (could still contain `{{}}` if template didn't resolve), but this isn't being propagated to the error message.

## Files to Investigate

### 1. `internal/workflow/executor.go`
**Line ~200-250**: `executeStep()` method
- This is where `toolCaller.CallTool()` is invoked
- When tool call fails, need to preserve the tool name
- Current error handling may be stripping the tool name

**Line ~76-109**: `expandForEachSteps()` method
- Already has some error handling improvements
- May need to add tool name to expanded step metadata

### 2. `internal/workflow/api_adapter.go`
**Line ~85-110**: `ExecuteWorkflow()` method
- Catches errors from executor
- This is where `buildToolAvailabilityError()` could be called
- Need to detect forEach tool failures here

**Line ~682-795**: `buildToolAvailabilityError()` method
- Already provides detailed tool availability info
- Should be called when forEach step tool is unavailable
- May need to accept additional context (failed tool name, forEach context)

## Implementation Strategy

### Step 1: Preserve Tool Name in Expanded Steps
When forEach expands steps, ensure the tool name is preserved in step metadata:

```go
// In expandForEachSteps()
expandedStep := &Step{
    ID:        expandedID,
    Tool:      resolvedTool,  // Resolved tool name
    Arguments: resolvedArgs,
    // ADD: Store original template for error messages
    Metadata: map[string]interface{}{
        "forEach_template": step.ForEach.Step.Tool,
        "forEach_item": item,
        "forEach_index": i,
    },
}
```

### Step 2: Enhanced Error Handling in executeStep()
When tool execution fails, include both the resolved tool name AND original template:

```go
// In executeStep()
result, err := e.toolCaller.CallTool(ctx, step.Tool, step.Arguments)
if err != nil {
    errorMsg := fmt.Sprintf("tool %s is not available", step.Tool)
    
    // If this was from forEach, include template info
    if template, ok := step.Metadata["forEach_template"].(string); ok {
        errorMsg = fmt.Sprintf("forEach step failed: tool template '%s' resolved to '%s' which is not available", 
            template, step.Tool)
    }
    
    return &mcp.CallToolResult{
        Content: []interface{}{errorMsg},
        IsError: true,
    }, nil
}
```

### Step 3: Add Available Tools to Error Response
When a tool fails, include the detailed tool availability info:

```go
// In ExecuteWorkflow() or executeStep()
if strings.Contains(err.Error(), "not available") {
    // Get the workflow to check tool availability
    workflow := /* current workflow */
    errorResponse := a.buildToolAvailabilityError(workflowName, []string{step.Tool})
    return &api.CallToolResult{
        Content: []interface{}{errorResponse},
        IsError: true,
    }, nil
}
```

### Step 4: Update Error JSON Structure
Ensure error responses include:
```json
{
  "error": "forEach step failed: tool template 'x_docs-mcp-server_search_library' resolved to 'x_docs-mcp-server_search_library' which is not available",
  "tool": "x_docs-mcp-server_search_library",
  "forEach_context": {
    "template": "x_docs-mcp-server_search_library",
    "item": {"library": "ansible"},
    "index": 0
  },
  "available_tools": {
    "core": ["core_service_list", "core_transform_text", ...],
    "external": ["x_docs-mcp-server_list_libraries", ...],
    "workflow": ["workflow_agent-initialization", ...]
  },
  "status": "failed"
}
```

## Testing Plan

1. **Create test workflow** with forEach using non-existent tool:
```yaml
steps:
  - name: test-foreach
    forEach:
      items: ["item1", "item2"]
      step:
        tool: x_nonexistent_tool
        arguments:
          value: "{{.item}}"
```

2. **Execute workflow** and verify error shows:
   - Original tool template
   - Resolved tool name (if different)
   - forEach context (item, index)
   - List of available tools

3. **Check execution JSON** (like err.json) contains all expected fields

## Files to Modify
- `internal/workflow/executor.go` (executeStep, expandForEachSteps)
- `internal/workflow/api_adapter.go` (ExecuteWorkflow, error handling)
- `internal/workflow/types.go` (add Metadata field to Step if needed)

## Success Criteria
- [ ] Error shows attempted tool name (never empty string)
- [ ] Error includes forEach context when applicable
- [ ] Error includes original template expression
- [ ] Error includes list of available tools (grouped by category)
- [ ] Error JSON matches expected structure
- [ ] Manual test with failing forEach workflow shows correct error
- [ ] Error appears in execution JSON (`err.json` style output)

## Related Context
- Previous work: commits `9048872`, `019cfa5`, `a15ffff` on dev branch
- Test workflow location: `tmp/agent-initialization.yaml` (has forEach loop)
- Error example: `tmp/err.json` (current broken format)
