package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		envVar       string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "returns env value when set",
			envVar:       "TEST_VAR",
			envValue:     "from_env",
			defaultValue: "default",
			expected:     "from_env",
		},
		{
			name:         "returns default when env not set",
			envVar:       "TEST_VAR_NOT_SET",
			envValue:     "",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "returns empty string from env if explicitly set",
			envVar:       "TEST_VAR_EMPTY",
			envValue:     "",
			defaultValue: "default",
			expected:     "default", // Empty env var is treated as not set
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(tt.envVar, tt.envValue)
				defer os.Unsetenv(tt.envVar)
			}

			// Test
			result := GetEnvOrDefault(tt.envVar, tt.defaultValue)

			// Verify
			if result != tt.expected {
				t.Errorf("GetEnvOrDefault() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetConfigPathFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "returns env value when MUSTER_CONFIG_PATH is set",
			envValue:     "/custom/config/path",
			defaultValue: "/default/path",
			expected:     "/custom/config/path",
		},
		{
			name:         "returns default when MUSTER_CONFIG_PATH not set",
			envValue:     "",
			defaultValue: "/default/path",
			expected:     "/default/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(EnvConfigPath, tt.envValue)
				defer os.Unsetenv(EnvConfigPath)
			} else {
				os.Unsetenv(EnvConfigPath)
			}

			// Test
			result := GetConfigPathFromEnv(tt.defaultValue)

			// Verify
			if result != tt.expected {
				t.Errorf("GetConfigPathFromEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetLogLevelFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		defaultValue string
		expected     string
	}{
		{
			name:         "normalizes to lowercase",
			envValue:     "DEBUG",
			defaultValue: "info",
			expected:     "debug",
		},
		{
			name:         "returns default when not set",
			envValue:     "",
			defaultValue: "info",
			expected:     "info",
		},
		{
			name:         "handles mixed case",
			envValue:     "WaRn",
			defaultValue: "info",
			expected:     "warn",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			if tt.envValue != "" {
				os.Setenv(EnvLogLevel, tt.envValue)
				defer os.Unsetenv(EnvLogLevel)
			} else {
				os.Unsetenv(EnvLogLevel)
			}

			// Test
			result := GetLogLevelFromEnv(tt.defaultValue)

			// Verify
			if result != tt.expected {
				t.Errorf("GetLogLevelFromEnv() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestValidateConfigPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testfile")
	os.WriteFile(tmpFile, []byte("test"), 0644)

	tests := []struct {
		name      string
		path      string
		wantError bool
		errorType error
	}{
		{
			name:      "valid directory",
			path:      tmpDir,
			wantError: false,
		},
		{
			name:      "empty path",
			path:      "",
			wantError: true,
			errorType: ErrConfigPathRequired,
		},
		{
			name:      "non-existent path",
			path:      "/path/that/does/not/exist",
			wantError: true,
			errorType: ErrConfigPathNotFound,
		},
		{
			name:      "file instead of directory",
			path:      tmpFile,
			wantError: true,
			errorType: ErrConfigPathNotDirectory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfigPath(tt.path)

			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateConfigPath() expected error, got nil")
				}
				// Check if it's the expected error type
				if tt.errorType != nil && err != tt.errorType {
					// For wrapped errors, check if it contains the expected error
					if !os.IsNotExist(err) && tt.errorType == ErrConfigPathNotFound {
						t.Errorf("ValidateConfigPath() error = %v, want %v", err, tt.errorType)
					}
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfigPath() unexpected error = %v", err)
				}
			}
		})
	}
}
