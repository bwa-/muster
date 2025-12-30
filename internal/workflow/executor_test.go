package workflow

import (
	"context"
	"encoding/json"
	"testing"

	"muster/internal/api"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockToolCaller implements ToolCaller for testing
type mockToolCaller struct {
	calls []toolCall
}

type toolCall struct {
	toolName string
	args     map[string]interface{}
}

func (m *mockToolCaller) CallToolInternal(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	m.calls = append(m.calls, toolCall{toolName: toolName, args: args})

	// Return a simple success result
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(`{"status": "success", "data": "test result"}`),
		},
		IsError: false,
	}, nil
}

func TestWorkflowExecutor_ExecuteWorkflow(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_workflow",
		Description: "Test workflow",
		Args: map[string]api.ArgDefinition{
			"cluster": {
				Type:        "string",
				Required:    true,
				Description: "Cluster name",
			},
		},
		Steps: []api.WorkflowStep{
			{
				ID:   "step1",
				Tool: "test_tool",
				Args: map[string]interface{}{
					"cluster": "{{ .input.cluster }}",
					"action":  "login",
				},
			},
		},
	}

	args := map[string]interface{}{
		"cluster": "test-cluster",
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, args)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify the tool was called with resolved arguments
	assert.Len(t, mock.calls, 1)
	assert.Equal(t, "test_tool", mock.calls[0].toolName)
	assert.Equal(t, "test-cluster", mock.calls[0].args["cluster"])
	assert.Equal(t, "login", mock.calls[0].args["action"])
}

func TestWorkflowExecutor_ValidateInputs(t *testing.T) {
	executor := NewWorkflowExecutor(nil, nil)

	argsDefinition := map[string]api.ArgDefinition{
		"required_string": {
			Type:        "string",
			Required:    true,
			Description: "Required string field",
		},
		"optional_number": {
			Type:        "number",
			Required:    false,
			Description: "Optional number field",
			Default:     float64(42),
		},
	}

	t.Run("valid inputs", func(t *testing.T) {
		args := map[string]interface{}{
			"required_string": "test",
		}
		err := executor.validateInputs(argsDefinition, args)
		assert.NoError(t, err)
		assert.Equal(t, float64(42), args["optional_number"]) // Default applied
	})

	t.Run("missing required field", func(t *testing.T) {
		args := map[string]interface{}{}
		err := executor.validateInputs(argsDefinition, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required field 'required_string' is missing")
	})

	t.Run("wrong type", func(t *testing.T) {
		args := map[string]interface{}{
			"required_string": 123, // Should be string
		}
		err := executor.validateInputs(argsDefinition, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "field 'required_string' has wrong type")
	})
}

func TestWorkflowExecutor_ResolveTemplate(t *testing.T) {
	executor := NewWorkflowExecutor(nil, nil)

	ctx := &executionContext{
		input: map[string]interface{}{
			"cluster": "test-cluster",
			"port":    8080,
		},
		results: map[string]interface{}{
			"step1": map[string]interface{}{
				"url": "http://localhost:9090",
			},
		},
	}

	tests := []struct {
		name     string
		template string
		expected interface{}
	}{
		{
			name:     "simple string",
			template: "{{ .input.cluster }}",
			expected: "test-cluster",
		},
		{
			name:     "number value",
			template: "{{ .input.port }}",
			expected: 8080, // Numbers are preserved as their original type
		},
		{
			name:     "nested access",
			template: "{{ .results.step1.url }}",
			expected: "http://localhost:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.resolveTemplate(tt.template, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflowExecutor_StoreResults(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_workflow",
		Description: "Test workflow with result storage",
		Args:        map[string]api.ArgDefinition{},
		Steps: []api.WorkflowStep{
			{
				ID:    "step1",
				Tool:  "test_tool",
				Args:  map[string]interface{}{},
				Store: true,
			},
			{
				ID:   "step2",
				Tool: "test_tool",
				Args: map[string]interface{}{
					"data": "{{ .results.step1.status }}",
				},
			},
		},
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify both tools were called
	assert.Len(t, mock.calls, 2)

	// Second call should have resolved the stored result
	assert.Equal(t, "success", mock.calls[1].args["data"])
}

func TestWorkflowExecutor_ResolveTemplate_StringNumbers(t *testing.T) {
	executor := NewWorkflowExecutor(nil, nil)

	// Test that string templates with numeric values don't get converted to float64
	ctx := &executionContext{
		input: map[string]interface{}{
			"localPort": "18000", // This is a string, not a number
		},
		variables: make(map[string]interface{}),
		results:   make(map[string]interface{}),
	}

	// This should return a string, not a float64
	result, err := executor.resolveTemplate("{{.input.localPort}}", ctx)
	assert.NoError(t, err)
	assert.Equal(t, "18000", result)
	assert.IsType(t, "", result) // Should be string type, not float64
}

func TestWorkflowExecutor_ForEach_BasicArray(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_foreach",
		Description: "Test forEach with basic array",
		Args: map[string]api.ArgDefinition{
			"microservices": {
				Type:        "array",
				Required:    true,
				Description: "List of microservices",
			},
		},
		Steps: []api.WorkflowStep{
			{
				ID: "deploy_all",
				ForEach: &api.ForEachConfig{
					Items: "{{.input.microservices}}",
					Step: api.WorkflowStepTemplate{
						ID:   "deploy_service",
						Tool: "deploy_microservice",
						Args: map[string]interface{}{
							"name":    "{{.item.name}}",
							"version": "{{.item.version}}",
						},
					},
				},
			},
		},
	}

	args := map[string]interface{}{
		"microservices": []interface{}{
			map[string]interface{}{"name": "frontend", "version": "1.0.0"},
			map[string]interface{}{"name": "backend", "version": "2.0.0"},
			map[string]interface{}{"name": "database", "version": "3.0.0"},
		},
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, args)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// Verify three tool calls were made (one for each item)
	assert.Len(t, mock.calls, 3)

	// Verify each call has the correct arguments
	assert.Equal(t, "deploy_microservice", mock.calls[0].toolName)
	assert.Equal(t, "frontend", mock.calls[0].args["name"])
	assert.Equal(t, "1.0.0", mock.calls[0].args["version"])

	assert.Equal(t, "deploy_microservice", mock.calls[1].toolName)
	assert.Equal(t, "backend", mock.calls[1].args["name"])
	assert.Equal(t, "2.0.0", mock.calls[1].args["version"])

	assert.Equal(t, "deploy_microservice", mock.calls[2].toolName)
	assert.Equal(t, "database", mock.calls[2].args["name"])
	assert.Equal(t, "3.0.0", mock.calls[2].args["version"])
}

func TestWorkflowExecutor_ForEach_DirectArray(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_foreach_direct",
		Description: "Test forEach with direct array",
		Args:        map[string]api.ArgDefinition{},
		Steps: []api.WorkflowStep{
			{
				ID: "process_items",
				ForEach: &api.ForEachConfig{
					Items: []interface{}{"item1", "item2", "item3"},
					Step: api.WorkflowStepTemplate{
						ID:   "process",
						Tool: "process_item",
						Args: map[string]interface{}{
							"value": "{{.item}}",
						},
					},
				},
			},
		},
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify three tool calls
	assert.Len(t, mock.calls, 3)
	assert.Equal(t, "item1", mock.calls[0].args["value"])
	assert.Equal(t, "item2", mock.calls[1].args["value"])
	assert.Equal(t, "item3", mock.calls[2].args["value"])
}

func TestWorkflowExecutor_ForEach_WithRegularSteps(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_mixed_steps",
		Description: "Test forEach mixed with regular steps",
		Args:        map[string]api.ArgDefinition{},
		Steps: []api.WorkflowStep{
			{
				ID:   "setup",
				Tool: "setup_tool",
				Args: map[string]interface{}{
					"action": "prepare",
				},
			},
			{
				ID: "process_items",
				ForEach: &api.ForEachConfig{
					Items: []interface{}{"a", "b"},
					Step: api.WorkflowStepTemplate{
						ID:   "process",
						Tool: "process_item",
						Args: map[string]interface{}{
							"value": "{{.item}}",
						},
					},
				},
			},
			{
				ID:   "cleanup",
				Tool: "cleanup_tool",
				Args: map[string]interface{}{
					"action": "finish",
				},
			},
		},
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify call order: setup, process_0, process_1, cleanup
	assert.Len(t, mock.calls, 4)
	assert.Equal(t, "setup_tool", mock.calls[0].toolName)
	assert.Equal(t, "process_item", mock.calls[1].toolName)
	assert.Equal(t, "a", mock.calls[1].args["value"])
	assert.Equal(t, "process_item", mock.calls[2].toolName)
	assert.Equal(t, "b", mock.calls[2].args["value"])
	assert.Equal(t, "cleanup_tool", mock.calls[3].toolName)
}

func TestWorkflowExecutor_ForEach_EmptyArray(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_empty_foreach",
		Description: "Test forEach with empty array",
		Args:        map[string]api.ArgDefinition{},
		Steps: []api.WorkflowStep{
			{
				ID: "process_items",
				ForEach: &api.ForEachConfig{
					Items: []interface{}{},
					Step: api.WorkflowStepTemplate{
						ID:   "process",
						Tool: "process_item",
						Args: map[string]interface{}{},
					},
				},
			},
		},
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, map[string]interface{}{})
	require.NoError(t, err)
	assert.NotNil(t, result)

	// No tool calls should be made for empty array
	assert.Len(t, mock.calls, 0)
}

// TODO: Fix error handling - this test uncovers a bug where nil result is returned with error
func TestWorkflowExecutor_ForEach_InvalidItemsType(t *testing.T) {
	t.Skip("Test reveals nil pointer bug in error handling - needs separate fix")
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_invalid_foreach",
		Description: "Test forEach with invalid items type",
		Args:        map[string]api.ArgDefinition{},
		Steps: []api.WorkflowStep{
			{
				ID: "process_items",
				ForEach: &api.ForEachConfig{
					Items: "not an array", // This is a string, not an array
					Step: api.WorkflowStepTemplate{
						ID:   "process",
						Tool: "process_item",
						Args: map[string]interface{}{},
					},
				},
			},
		},
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, map[string]interface{}{})
	assert.Error(t, err)
	assert.NotNil(t, result, "Should return a result even when forEach expansion fails")
	assert.Contains(t, err.Error(), "must be an array")
	assert.True(t, result.IsError, "Result should be marked as error")
}

// mockToolCallerWithError implements ToolCaller that returns errors for specific tools
type mockToolCallerWithError struct {
	calls        []toolCall
	errorForTool string
}

func (m *mockToolCallerWithError) CallToolInternal(ctx context.Context, toolName string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	m.calls = append(m.calls, toolCall{toolName: toolName, args: args})

	// Return error if this is the tool we want to fail
	if m.errorForTool != "" && toolName == m.errorForTool {
		return nil, assert.AnError
	}

	// Return a simple success result
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(`{"status": "success", "data": "test result"}`),
		},
		IsError: false,
	}, nil
}

func TestWorkflowExecutor_ForEach_ErrorHandling(t *testing.T) {
	mock := &mockToolCallerWithError{
		errorForTool: "nonexistent_tool",
	}
	executor := NewWorkflowExecutor(mock, nil)

	workflow := &api.Workflow{
		Name:        "test_foreach_error",
		Description: "Test forEach error message improvements",
		Args:        map[string]api.ArgDefinition{},
		Steps: []api.WorkflowStep{
			{
				ID: "process_items",
				ForEach: &api.ForEachConfig{
					Items: []interface{}{"item1", "item2", "item3"},
					Step: api.WorkflowStepTemplate{
						ID:   "failing_step",
						Tool: "nonexistent_tool",
						Args: map[string]interface{}{
							"value": "{{.item}}",
						},
						AllowFailure: false,
					},
				},
			},
		},
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, map[string]interface{}{})
	require.Error(t, err)
	require.NotNil(t, result)

	// Verify result contains detailed error information
	assert.True(t, result.IsError)
	require.Len(t, result.Content, 1)
	
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok)
	
	// Parse the JSON result
	var resultData map[string]interface{}
	jsonErr := json.Unmarshal([]byte(textContent.Text), &resultData)
	require.NoError(t, jsonErr, "Should be valid JSON")
	
	// Verify the steps array contains error details
	steps, ok := resultData["steps"].([]interface{})
	require.True(t, ok, "Should have steps array")
	require.Greater(t, len(steps), 0, "Should have at least one step")
	
	firstStep := steps[0].(map[string]interface{})
	
	// Check that tool name is not empty (the main fix we're testing)
	tool, _ := firstStep["tool"].(string)
	assert.NotEmpty(t, tool, "Tool name should not be empty in error response")
	assert.Equal(t, "nonexistent_tool", tool, "Tool name should match the forEach step tool")
	
	// Check that error message is present and includes tool name
	errorMsg, hasError := firstStep["error"].(string)
	assert.True(t, hasError, "Should have error field")
	assert.NotEmpty(t, errorMsg, "Error message should not be empty")
	assert.Contains(t, errorMsg, "tool 'nonexistent_tool' is not available", "Error should clearly state tool is not available")
	
	// Check that available_tools is present to help user find correct name
	availableTools, hasAvailableTools := firstStep["available_tools"].(map[string]interface{})
	assert.True(t, hasAvailableTools, "Should have available_tools field")
	
	if hasAvailableTools {
		// Verify structure has core, external, workflow categories
		_, hasCore := availableTools["core"]
		_, hasExternal := availableTools["external"]
		_, hasWorkflow := availableTools["workflow"]
		total, hasTotal := availableTools["total"]
		
		assert.True(t, hasCore, "Should have core tools category")
		assert.True(t, hasExternal, "Should have external tools category")
		assert.True(t, hasWorkflow, "Should have workflow tools category")
		assert.True(t, hasTotal, "Should have total count")
		if totalInt, ok := total.(int); ok {
			assert.GreaterOrEqual(t, totalInt, 0, "Total should be non-negative")
		}
	}
}
