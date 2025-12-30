package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"muster/pkg/logging"
)

// TextTransformer handles text transformation pipelines
type TextTransformer struct{}

// NewTextTransformer creates a new text transformer
func NewTextTransformer() *TextTransformer {
	return &TextTransformer{}
}

// TransformationStep defines a single transformation operation
type TransformationStep struct {
	Type string                 `json:"type"`
	Args map[string]interface{} `json:"args,omitempty"`
}

// TransformRequest defines the input for text transformation
type TransformRequest struct {
	Input string               `json:"input"`
	Steps []TransformationStep `json:"steps"`
}

// TransformResult defines the output of text transformation
type TransformResult struct {
	Result interface{} `json:"result"` // Can be string or []string
	Type   string      `json:"type"`   // "string" or "array"
}

// Transform applies a series of transformation steps to input text
func (tt *TextTransformer) Transform(req TransformRequest) (*TransformResult, error) {
	logging.Debug("TextTransformer", "Starting transformation with %d steps", len(req.Steps))

	// Current state can be either a string or array of strings
	var currentValue interface{} = req.Input
	var currentType string = "string"

	for i, step := range req.Steps {
		logging.Debug("TextTransformer", "Step %d/%d: type=%s", i+1, len(req.Steps), step.Type)

		var err error
		currentValue, currentType, err = tt.applyStep(currentValue, currentType, step)
		if err != nil {
			return nil, fmt.Errorf("transformation step %d (%s) failed: %w", i+1, step.Type, err)
		}

		logging.Debug("TextTransformer", "Step %d result type: %s", i+1, currentType)
	}

	return &TransformResult{
		Result: currentValue,
		Type:   currentType,
	}, nil
}

// applyStep applies a single transformation step
func (tt *TextTransformer) applyStep(value interface{}, valueType string, step TransformationStep) (interface{}, string, error) {
	switch step.Type {
	// Extraction operations
	case "extract_lines_starting_with":
		return tt.extractLinesStartingWith(value, valueType, step.Args)
	case "extract_lines_matching":
		return tt.extractLinesMatching(value, valueType, step.Args)
	case "extract_between":
		return tt.extractBetween(value, valueType, step.Args)
	case "extract_after":
		return tt.extractAfter(value, valueType, step.Args)
	case "extract_before":
		return tt.extractBefore(value, valueType, step.Args)

	// Modification operations
	case "replace":
		return tt.replace(value, valueType, step.Args)
	case "replace_regex":
		return tt.replaceRegex(value, valueType, step.Args)
	case "trim":
		return tt.trim(value, valueType, step.Args)
	case "trim_prefix":
		return tt.trimPrefix(value, valueType, step.Args)
	case "trim_suffix":
		return tt.trimSuffix(value, valueType, step.Args)
	case "trim_each":
		return tt.trimEach(value, valueType, step.Args)
	case "to_lower":
		return tt.toLower(value, valueType, step.Args)
	case "to_upper":
		return tt.toUpper(value, valueType, step.Args)

	// Splitting/filtering operations
	case "split":
		return tt.split(value, valueType, step.Args)
	case "split_lines":
		return tt.splitLines(value, valueType, step.Args)
	case "remove_empty":
		return tt.removeEmpty(value, valueType, step.Args)
	case "remove_duplicates":
		return tt.removeDuplicates(value, valueType, step.Args)

	default:
		return nil, "", fmt.Errorf("unknown transformation type: %s", step.Type)
	}
}

// Helper functions to get arguments with defaults
func getStringArg(args map[string]interface{}, key string, defaultValue string) string {
	if val, ok := args[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := args[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

// ensureString converts value to string, returns error if it's an array
func ensureString(value interface{}, valueType string) (string, error) {
	if valueType != "string" {
		return "", fmt.Errorf("operation requires string input, got %s", valueType)
	}
	if str, ok := value.(string); ok {
		return str, nil
	}
	return "", fmt.Errorf("invalid string value")
}

// ensureArray converts value to string array, returns error if it's not an array
func ensureArray(value interface{}, valueType string) ([]string, error) {
	if valueType != "array" {
		return nil, fmt.Errorf("operation requires array input, got %s", valueType)
	}
	if arr, ok := value.([]string); ok {
		return arr, nil
	}
	return nil, fmt.Errorf("invalid array value")
}

// ============================================================================
// EXTRACTION OPERATIONS
// ============================================================================

// extractLinesStartingWith extracts lines that start with a specific prefix
func (tt *TextTransformer) extractLinesStartingWith(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	str, err := ensureString(value, valueType)
	if err != nil {
		return nil, "", err
	}

	prefix := getStringArg(args, "prefix", "")
	if prefix == "" {
		return nil, "", fmt.Errorf("prefix argument is required")
	}

	lines := strings.Split(str, "\n")
	var result []string
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			result = append(result, line)
		}
	}

	return result, "array", nil
}

// extractLinesMatching extracts lines matching a regex pattern
func (tt *TextTransformer) extractLinesMatching(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	str, err := ensureString(value, valueType)
	if err != nil {
		return nil, "", err
	}

	pattern := getStringArg(args, "pattern", "")
	if pattern == "" {
		return nil, "", fmt.Errorf("pattern argument is required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	lines := strings.Split(str, "\n")
	var result []string
	for _, line := range lines {
		if re.MatchString(line) {
			result = append(result, line)
		}
	}

	return result, "array", nil
}

// extractBetween extracts text between two markers
func (tt *TextTransformer) extractBetween(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	str, err := ensureString(value, valueType)
	if err != nil {
		return nil, "", err
	}

	start := getStringArg(args, "start", "")
	end := getStringArg(args, "end", "")
	if start == "" || end == "" {
		return nil, "", fmt.Errorf("both start and end arguments are required")
	}

	startIdx := strings.Index(str, start)
	if startIdx == -1 {
		return "", "string", nil // Return empty if start marker not found
	}

	startIdx += len(start)
	endIdx := strings.Index(str[startIdx:], end)
	if endIdx == -1 {
		return "", "string", nil // Return empty if end marker not found
	}

	result := str[startIdx : startIdx+endIdx]
	return result, "string", nil
}

// extractAfter extracts all text after a marker
func (tt *TextTransformer) extractAfter(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	str, err := ensureString(value, valueType)
	if err != nil {
		return nil, "", err
	}

	marker := getStringArg(args, "marker", "")
	if marker == "" {
		return nil, "", fmt.Errorf("marker argument is required")
	}

	idx := strings.Index(str, marker)
	if idx == -1 {
		return "", "string", nil // Return empty if marker not found
	}

	result := str[idx+len(marker):]
	return result, "string", nil
}

// extractBefore extracts all text before a marker
func (tt *TextTransformer) extractBefore(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	str, err := ensureString(value, valueType)
	if err != nil {
		return nil, "", err
	}

	marker := getStringArg(args, "marker", "")
	if marker == "" {
		return nil, "", fmt.Errorf("marker argument is required")
	}

	idx := strings.Index(str, marker)
	if idx == -1 {
		return str, "string", nil // Return original if marker not found
	}

	result := str[:idx]
	return result, "string", nil
}

// ============================================================================
// MODIFICATION OPERATIONS
// ============================================================================

// replace performs string replacement
func (tt *TextTransformer) replace(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	old := getStringArg(args, "old", "")
	new := getStringArg(args, "new", "")
	if old == "" {
		return nil, "", fmt.Errorf("old argument is required")
	}

	if valueType == "string" {
		str, err := ensureString(value, valueType)
		if err != nil {
			return nil, "", err
		}
		result := strings.ReplaceAll(str, old, new)
		return result, "string", nil
	}

	// Apply to each array element
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = strings.ReplaceAll(item, old, new)
	}
	return result, "array", nil
}

// replaceRegex performs regex replacement
func (tt *TextTransformer) replaceRegex(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	pattern := getStringArg(args, "pattern", "")
	replacement := getStringArg(args, "replacement", "")
	if pattern == "" {
		return nil, "", fmt.Errorf("pattern argument is required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	if valueType == "string" {
		str, err := ensureString(value, valueType)
		if err != nil {
			return nil, "", err
		}
		result := re.ReplaceAllString(str, replacement)
		return result, "string", nil
	}

	// Apply to each array element
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = re.ReplaceAllString(item, replacement)
	}
	return result, "array", nil
}

// trim removes leading and trailing whitespace
func (tt *TextTransformer) trim(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	if valueType == "string" {
		str, err := ensureString(value, valueType)
		if err != nil {
			return nil, "", err
		}
		result := strings.TrimSpace(str)
		return result, "string", nil
	}

	// Apply to each array element
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = strings.TrimSpace(item)
	}
	return result, "array", nil
}

// trimPrefix removes a specific prefix
func (tt *TextTransformer) trimPrefix(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	prefix := getStringArg(args, "prefix", "")
	if prefix == "" {
		return nil, "", fmt.Errorf("prefix argument is required")
	}

	if valueType == "string" {
		str, err := ensureString(value, valueType)
		if err != nil {
			return nil, "", err
		}
		result := strings.TrimPrefix(str, prefix)
		return result, "string", nil
	}

	// Apply to each array element
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = strings.TrimPrefix(item, prefix)
	}
	return result, "array", nil
}

// trimSuffix removes a specific suffix
func (tt *TextTransformer) trimSuffix(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	suffix := getStringArg(args, "suffix", "")
	if suffix == "" {
		return nil, "", fmt.Errorf("suffix argument is required")
	}

	if valueType == "string" {
		str, err := ensureString(value, valueType)
		if err != nil {
			return nil, "", err
		}
		result := strings.TrimSuffix(str, suffix)
		return result, "string", nil
	}

	// Apply to each array element
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = strings.TrimSuffix(item, suffix)
	}
	return result, "array", nil
}

// trimEach trims whitespace from each array element
func (tt *TextTransformer) trimEach(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = strings.TrimSpace(item)
	}
	return result, "array", nil
}

// toLower converts text to lowercase
func (tt *TextTransformer) toLower(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	if valueType == "string" {
		str, err := ensureString(value, valueType)
		if err != nil {
			return nil, "", err
		}
		result := strings.ToLower(str)
		return result, "string", nil
	}

	// Apply to each array element
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = strings.ToLower(item)
	}
	return result, "array", nil
}

// toUpper converts text to uppercase
func (tt *TextTransformer) toUpper(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	if valueType == "string" {
		str, err := ensureString(value, valueType)
		if err != nil {
			return nil, "", err
		}
		result := strings.ToUpper(str)
		return result, "string", nil
	}

	// Apply to each array element
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := make([]string, len(arr))
	for i, item := range arr {
		result[i] = strings.ToUpper(item)
	}
	return result, "array", nil
}

// ============================================================================
// SPLITTING/FILTERING OPERATIONS
// ============================================================================

// split splits a string by delimiter into an array
func (tt *TextTransformer) split(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	str, err := ensureString(value, valueType)
	if err != nil {
		return nil, "", err
	}

	delimiter := getStringArg(args, "delimiter", ",")
	result := strings.Split(str, delimiter)
	return result, "array", nil
}

// splitLines splits a string by newlines into an array
func (tt *TextTransformer) splitLines(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	str, err := ensureString(value, valueType)
	if err != nil {
		return nil, "", err
	}

	result := strings.Split(str, "\n")
	return result, "array", nil
}

// removeEmpty removes empty strings from an array
func (tt *TextTransformer) removeEmpty(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	var result []string
	for _, item := range arr {
		if strings.TrimSpace(item) != "" {
			result = append(result, item)
		}
	}
	return result, "array", nil
}

// removeDuplicates removes duplicate strings from an array
func (tt *TextTransformer) removeDuplicates(value interface{}, valueType string, args map[string]interface{}) (interface{}, string, error) {
	arr, err := ensureArray(value, valueType)
	if err != nil {
		return nil, "", err
	}

	seen := make(map[string]bool)
	var result []string
	for _, item := range arr {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result, "array", nil
}
