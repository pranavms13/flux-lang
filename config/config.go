package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FluxConfig represents the configuration for the Flux language
type FluxConfig struct {
	TypeChecking TypeCheckingConfig `json:"typeChecking"`
	Compiler     CompilerConfig     `json:"compiler"`
}

// TypeCheckingConfig controls type checking behavior
type TypeCheckingConfig struct {
	Strict   bool `json:"strict"`
	WarnOnly bool `json:"warnOnly"`
	Enabled  bool `json:"enabled"`
}

// CompilerConfig controls compiler behavior
type CompilerConfig struct {
	OptimizationLevel int  `json:"optimizationLevel"`
	Debug             bool `json:"debug"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *FluxConfig {
	return &FluxConfig{
		TypeChecking: TypeCheckingConfig{
			Strict:   false, // Default to non-strict for backward compatibility
			WarnOnly: false,
			Enabled:  true,
		},
		Compiler: CompilerConfig{
			OptimizationLevel: 1,
			Debug:             false,
		},
	}
}

// LoadConfig loads configuration from flux.json file
func LoadConfig(dir string) (*FluxConfig, error) {
	configPath := filepath.Join(dir, "flux.json")

	// If no config file exists, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := json.Unmarshal(configData, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves configuration to flux.json file
func SaveConfig(config *FluxConfig, dir string) error {
	configPath := filepath.Join(dir, "flux.json")

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetConfigFromCurrentDir loads config from the current working directory
func GetConfigFromCurrentDir() (*FluxConfig, error) {
	wd, err := os.Getwd()
	if err != nil {
		return DefaultConfig(), nil // Fallback to default if can't get working directory
	}
	return LoadConfig(wd)
}
