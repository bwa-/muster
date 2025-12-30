package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTextTransformer_ExtractLinesStartingWith tests extraction by prefix
func TestTextTransformer_ExtractLinesStartingWith(t *testing.T) {
	transformer := NewTextTransformer()

	tests := []struct {
		name     string
		input    string
		prefix   string
		expected []string
	}{
		{
			name: "markdown bullet list",
			input: `Available items:

- item1
- item2
- item3`,
			prefix:   "- ",
			expected: []string{"- item1", "- item2", "- item3"},
		},
		{
			name: "no matches",
			input: `line1
line2
line3`,
			prefix:   "* ",
			expected: nil,
		},
		{
			name:     "empty input",
			input:    "",
			prefix:   "- ",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.Transform(TransformRequest{
				Input: tt.input,
				Steps: []TransformationStep{
					{
						Type: "extract_lines_starting_with",
						Args: map[string]interface{}{"prefix": tt.prefix},
					},
				},
			})

			require.NoError(t, err)
			assert.Equal(t, "array", result.Type)
			assert.Equal(t, tt.expected, result.Result)
		})
	}
}

// TestTextTransformer_ExtractLinesMatching tests regex extraction
func TestTextTransformer_ExtractLinesMatching(t *testing.T) {
	transformer := NewTextTransformer()

	tests := []struct {
		name     string
		input    string
		pattern  string
		expected []string
	}{
		{
			name: "extract service names",
			input: `service1: running
service2: stopped
ignored line
service3: running`,
			pattern:  "^[a-zA-Z0-9]+:",
			expected: []string{"service1: running", "service2: stopped", "service3: running"},
		},
		{
			name: "extract error lines",
			input: `INFO: starting
ERROR: failed to connect
WARN: deprecated
ERROR: timeout`,
			pattern:  "^ERROR:",
			expected: []string{"ERROR: failed to connect", "ERROR: timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.Transform(TransformRequest{
				Input: tt.input,
				Steps: []TransformationStep{
					{
						Type: "extract_lines_matching",
						Args: map[string]interface{}{"pattern": tt.pattern},
					},
				},
			})

			require.NoError(t, err)
			assert.Equal(t, "array", result.Type)
			assert.Equal(t, tt.expected, result.Result)
		})
	}
}

// TestTextTransformer_ExtractBetween tests text extraction between markers
func TestTextTransformer_ExtractBetween(t *testing.T) {
	transformer := NewTextTransformer()

	tests := []struct {
		name     string
		input    string
		start    string
		end      string
		expected string
	}{
		{
			name:     "extract content",
			input:    "prefix [START] extracted content [END] suffix",
			start:    "[START]",
			end:      "[END]",
			expected: " extracted content ",
		},
		{
			name:     "start marker not found",
			input:    "no markers here",
			start:    "[START]",
			end:      "[END]",
			expected: "",
		},
		{
			name:     "end marker not found",
			input:    "prefix [START] no end",
			start:    "[START]",
			end:      "[END]",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.Transform(TransformRequest{
				Input: tt.input,
				Steps: []TransformationStep{
					{
						Type: "extract_between",
						Args: map[string]interface{}{
							"start": tt.start,
							"end":   tt.end,
						},
					},
				},
			})

			require.NoError(t, err)
			assert.Equal(t, "string", result.Type)
			assert.Equal(t, tt.expected, result.Result)
		})
	}
}

// TestTextTransformer_ExtractAfter tests extraction after marker
func TestTextTransformer_ExtractAfter(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "prefix Results: actual content",
		Steps: []TransformationStep{
			{
				Type: "extract_after",
				Args: map[string]interface{}{"marker": "Results:"},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, " actual content", result.Result)
}

// TestTextTransformer_ExtractBefore tests extraction before marker
func TestTextTransformer_ExtractBefore(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "actual content --- footer",
		Steps: []TransformationStep{
			{
				Type: "extract_before",
				Args: map[string]interface{}{"marker": "---"},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, "actual content ", result.Result)
}

// TestTextTransformer_Replace tests string replacement
func TestTextTransformer_Replace(t *testing.T) {
	transformer := NewTextTransformer()

	tests := []struct {
		name     string
		input    interface{}
		old      string
		new      string
		expected interface{}
	}{
		{
			name:     "replace in string",
			input:    "item1 and item2 and item3",
			old:      " and ",
			new:      ", ",
			expected: "item1, item2, item3",
		},
		{
			name:     "replace in array",
			input:    []string{"item1 and item2", "item3 and item4"},
			old:      " and ",
			new:      ", ",
			expected: []string{"item1, item2", "item3, item4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if arr, ok := tt.input.([]string); ok {
				result, resultType, err := transformer.applyStep(arr, "array", TransformationStep{
					Type: "replace",
					Args: map[string]interface{}{"old": tt.old, "new": tt.new},
				})
				require.NoError(t, err)
				assert.Equal(t, "array", resultType)
				assert.Equal(t, tt.expected, result)
			} else {
				inputStr := tt.input.(string)
				result, resultType, err := transformer.applyStep(inputStr, "string", TransformationStep{
					Type: "replace",
					Args: map[string]interface{}{"old": tt.old, "new": tt.new},
				})
				require.NoError(t, err)
				assert.Equal(t, "string", resultType)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestTextTransformer_ReplaceRegex tests regex replacement
func TestTextTransformer_ReplaceRegex(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "service1: running, service2: stopped",
		Steps: []TransformationStep{
			{
				Type: "replace_regex",
				Args: map[string]interface{}{
					"pattern":     ":\\s*\\w+",
					"replacement": "",
				},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, "service1, service2", result.Result)
}

// TestTextTransformer_Trim tests whitespace trimming
func TestTextTransformer_Trim(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "  content with spaces  ",
		Steps: []TransformationStep{
			{Type: "trim"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, "content with spaces", result.Result)
}

// TestTextTransformer_TrimPrefix tests prefix removal
func TestTextTransformer_TrimPrefix(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "- item with bullet",
		Steps: []TransformationStep{
			{
				Type: "trim_prefix",
				Args: map[string]interface{}{"prefix": "- "},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, "item with bullet", result.Result)
}

// TestTextTransformer_TrimSuffix tests suffix removal
func TestTextTransformer_TrimSuffix(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "file.txt",
		Steps: []TransformationStep{
			{
				Type: "trim_suffix",
				Args: map[string]interface{}{"suffix": ".txt"},
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, "file", result.Result)
}

// TestTextTransformer_TrimEach tests trimming array elements
func TestTextTransformer_TrimEach(t *testing.T) {
	transformer := NewTextTransformer()

	// First create an array, then trim each
	result, err := transformer.Transform(TransformRequest{
		Input: " item1 , item2 , item3 ",
		Steps: []TransformationStep{
			{
				Type: "split",
				Args: map[string]interface{}{"delimiter": ","},
			},
			{Type: "trim_each"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "array", result.Type)
	assert.Equal(t, []string{"item1", "item2", "item3"}, result.Result)
}

// TestTextTransformer_ToLower tests lowercase conversion
func TestTextTransformer_ToLower(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "MiXeD CaSe",
		Steps: []TransformationStep{
			{Type: "to_lower"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, "mixed case", result.Result)
}

// TestTextTransformer_ToUpper tests uppercase conversion
func TestTextTransformer_ToUpper(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "lowercase",
		Steps: []TransformationStep{
			{Type: "to_upper"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "string", result.Type)
	assert.Equal(t, "LOWERCASE", result.Result)
}

// TestTextTransformer_Split tests delimiter-based splitting
func TestTextTransformer_Split(t *testing.T) {
	transformer := NewTextTransformer()

	tests := []struct {
		name      string
		input     string
		delimiter string
		expected  []string
	}{
		{
			name:      "comma separated",
			input:     "item1,item2,item3",
			delimiter: ",",
			expected:  []string{"item1", "item2", "item3"},
		},
		{
			name:      "pipe separated",
			input:     "a|b|c",
			delimiter: "|",
			expected:  []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := transformer.Transform(TransformRequest{
				Input: tt.input,
				Steps: []TransformationStep{
					{
						Type: "split",
						Args: map[string]interface{}{"delimiter": tt.delimiter},
					},
				},
			})

			require.NoError(t, err)
			assert.Equal(t, "array", result.Type)
			assert.Equal(t, tt.expected, result.Result)
		})
	}
}

// TestTextTransformer_SplitLines tests newline splitting
func TestTextTransformer_SplitLines(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "line1\nline2\nline3",
		Steps: []TransformationStep{
			{Type: "split_lines"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "array", result.Type)
	assert.Equal(t, []string{"line1", "line2", "line3"}, result.Result)
}

// TestTextTransformer_RemoveEmpty tests empty string removal
func TestTextTransformer_RemoveEmpty(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "item1,,item2, ,item3",
		Steps: []TransformationStep{
			{
				Type: "split",
				Args: map[string]interface{}{"delimiter": ","},
			},
			{Type: "remove_empty"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "array", result.Type)
	// remove_empty removes both empty strings and whitespace-only strings
	expected := []string{"item1", "item2", "item3"}
	assert.Equal(t, expected, result.Result)
}

// TestTextTransformer_RemoveDuplicates tests duplicate removal
func TestTextTransformer_RemoveDuplicates(t *testing.T) {
	transformer := NewTextTransformer()

	result, err := transformer.Transform(TransformRequest{
		Input: "a,b,a,c,b,d",
		Steps: []TransformationStep{
			{
				Type: "split",
				Args: map[string]interface{}{"delimiter": ","},
			},
			{Type: "remove_duplicates"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "array", result.Type)
	assert.Equal(t, []string{"a", "b", "c", "d"}, result.Result)
}

// TestTextTransformer_ComplexPipeline tests a complete transformation pipeline
func TestTextTransformer_ComplexPipeline(t *testing.T) {
	transformer := NewTextTransformer()

	input := `Indexed libraries:

- ansible
- docker
- kubernetes
- muster
- proxmox`

	result, err := transformer.Transform(TransformRequest{
		Input: input,
		Steps: []TransformationStep{
			{
				Type: "extract_lines_starting_with",
				Args: map[string]interface{}{"prefix": "- "},
			},
			{
				Type: "trim_prefix",
				Args: map[string]interface{}{"prefix": "- "},
			},
			{Type: "trim_each"},
			{Type: "remove_empty"},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "array", result.Type)
	expected := []string{"ansible", "docker", "kubernetes", "muster", "proxmox"}
	assert.Equal(t, expected, result.Result)
}

// TestTextTransformer_ErrorHandling tests error cases
func TestTextTransformer_ErrorHandling(t *testing.T) {
	transformer := NewTextTransformer()

	tests := []struct {
		name        string
		input       string
		steps       []TransformationStep
		expectError bool
		errorMsg    string
	}{
		{
			name:  "missing required argument",
			input: "test",
			steps: []TransformationStep{
				{Type: "extract_lines_starting_with"},
			},
			expectError: true,
			errorMsg:    "prefix argument is required",
		},
		{
			name:  "invalid regex",
			input: "test",
			steps: []TransformationStep{
				{
					Type: "extract_lines_matching",
					Args: map[string]interface{}{"pattern": "[invalid"},
				},
			},
			expectError: true,
			errorMsg:    "invalid regex pattern",
		},
		{
			name:  "wrong input type",
			input: "test string",
			steps: []TransformationStep{
				{Type: "trim_each"},
			},
			expectError: true,
			errorMsg:    "operation requires array input",
		},
		{
			name:  "unknown transformation type",
			input: "test",
			steps: []TransformationStep{
				{Type: "nonexistent_transform"},
			},
			expectError: true,
			errorMsg:    "unknown transformation type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := transformer.Transform(TransformRequest{
				Input: tt.input,
				Steps: tt.steps,
			})

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
