package config

import (
	"os"
	"strings"
)

// Environment variable names
const (
	EnvConfigPath    = "MUSTER_CONFIG_PATH"
	EnvEndpoint      = "MUSTER_ENDPOINT"
	EnvLogLevel      = "MUSTER_LOG_LEVEL"
	EnvOutputFormat  = "MUSTER_OUTPUT_FORMAT"
	EnvHTTPPort      = "MUSTER_HTTP_PORT"
	EnvMCPPort       = "MUSTER_MCP_PORT"
)

// GetEnvOrDefault returns the environment variable value if set, otherwise returns the default value.
// This function provides a consistent way to handle environment variable lookups across the application.
func GetEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// GetConfigPathFromEnv returns the config path from the environment variable if set,
// otherwise returns the provided default value.
// This function is specifically for handling the MUSTER_CONFIG_PATH environment variable.
func GetConfigPathFromEnv(defaultValue string) string {
	return GetEnvOrDefault(EnvConfigPath, defaultValue)
}

// GetEndpointFromEnv returns the endpoint from the environment variable if set,
// otherwise returns the provided default value.
// This function is specifically for handling the MUSTER_ENDPOINT environment variable.
func GetEndpointFromEnv(defaultValue string) string {
	return GetEnvOrDefault(EnvEndpoint, defaultValue)
}

// GetLogLevelFromEnv returns the log level from the environment variable if set,
// otherwise returns the provided default value.
// Valid values are: debug, info, warn, error
func GetLogLevelFromEnv(defaultValue string) string {
	value := GetEnvOrDefault(EnvLogLevel, defaultValue)
	// Normalize to lowercase for consistency
	return strings.ToLower(value)
}

// GetOutputFormatFromEnv returns the output format from the environment variable if set,
// otherwise returns the provided default value.
// Valid values are: table, json, yaml
func GetOutputFormatFromEnv(defaultValue string) string {
	value := GetEnvOrDefault(EnvOutputFormat, defaultValue)
	// Normalize to lowercase for consistency
	return strings.ToLower(value)
}

// ValidateConfigPath checks if the provided config path exists and is accessible.
// Returns an error if the path doesn't exist or isn't a directory.
func ValidateConfigPath(path string) error {
	if path == "" {
		return ErrConfigPathRequired
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrConfigPathNotFound
		}
		return err
	}

	if !info.IsDir() {
		return ErrConfigPathNotDirectory
	}

	return nil
}
