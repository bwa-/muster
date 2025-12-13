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
	executor := NewWorkflowExecutor(mock)

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
	executor := NewWorkflowExecutor(mock)

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
	executor := NewWorkflowExecutor(mock)

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
	executor := NewWorkflowExecutor(mock)

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

func TestWorkflowExecutor_ForEach_InvalidItemsType(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock)

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
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "must be an array")
}

func TestWorkflowExecutor_InputOutputFields(t *testing.T) {
	mock := &mockToolCaller{}
	executor := NewWorkflowExecutor(mock)

	workflow := &api.Workflow{
		Name:        "test_input_output",
		Description: "Test workflow input/output visibility",
		Args: map[string]api.ArgDefinition{
			"test_arg": {
				Type:        "string",
				Required:    false,
				Description: "A test argument",
				Default:     "default_value",
			},
		},
		Steps: []api.WorkflowStep{
			{
				ID:   "step1",
				Tool: "test_tool_1",
				Args: map[string]interface{}{
					"input_param": "{{.input.test_arg}}",
					"extra":       "data",
				},
				Store: true,
			},
			{
				ID:   "step2",
				Tool: "test_tool_2",
				Args: map[string]interface{}{
					"prev": "{{.results.step1}}",
				},
				Store: true,
			},
		},
	}

	args := map[string]interface{}{
		"test_arg": "custom_value",
	}

	result, err := executor.ExecuteWorkflow(context.Background(), workflow, args)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsError)

	// The result contains the workflow execution data in JSON format
	require.Len(t, result.Content, 1)

	// Parse the JSON result to verify input/output fields
	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "Content should be TextContent")

	var resultData map[string]interface{}
	err = json.Unmarshal([]byte(textContent.Text), &resultData)
	require.NoError(t, err)

	// Check that steps array exists
	steps, ok := resultData["steps"].([]interface{})
	require.True(t, ok, "steps should be an array")
	require.Len(t, steps, 2)

	// Check step1 has input and output fields
	step1, ok := steps[0].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "step1", step1["id"])
	assert.Equal(t, "completed", step1["status"])

	// Verify input field exists and contains resolved arguments
	input1, ok := step1["input"].(map[string]interface{})
	require.True(t, ok, "step1 should have 'input' field with resolved arguments")
	assert.Equal(t, "custom_value", input1["input_param"], "input should contain resolved template")
	assert.Equal(t, "data", input1["extra"])

	// Verify output field exists
	_, hasOutput := step1["output"]
	assert.True(t, hasOutput, "step1 should have 'output' field")

	// Check step2 has input field with reference to step1 result
	step2, ok := steps[1].(map[string]interface{})
	require.True(t, ok)
	input2, ok := step2["input"].(map[string]interface{})
	require.True(t, ok, "step2 should have 'input' field")
	assert.NotNil(t, input2["prev"], "step2 input should contain result from step1")

	// Pretty print the result for visual verification
	prettyJSON, _ := json.MarshalIndent(resultData, "", "  ")
	t.Logf("Workflow result with input/output fields:\n%s", string(prettyJSON))
}
