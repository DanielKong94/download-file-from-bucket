package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application
type Config struct {
	Providers map[string]ProviderConfig `yaml:"providers"`
}

// ProviderConfig holds configuration for a specific cloud provider
type ProviderConfig struct {
	Type      string            `yaml:"type"`      // "s3", "digitalocean", etc.
	Region    string            `yaml:"region"`
	Endpoint  string            `yaml:"endpoint"`  // For DigitalOcean Spaces or custom S3 endpoints
	AccessKey string            `yaml:"access_key"`
	SecretKey string            `yaml:"secret_key"`
	Bucket    string            `yaml:"bucket"`
	Options   map[string]string `yaml:"options"`   // Additional provider-specific options
}

// LoadConfig loads configuration from file or environment variables
func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.download-bucket")
	viper.AddConfigPath("/etc/download-bucket/")

	// Set environment variable prefix
	viper.SetEnvPrefix("BUCKET")
	viper.AutomaticEnv()

	config := &Config{
		Providers: make(map[string]ProviderConfig),
	}

	// Try to read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, that's okay, we'll use environment variables
	}

	// Unmarshal into our config struct
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// If no providers configured, try to load from environment
	if len(config.Providers) == 0 {
		if err := loadFromEnvironment(config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment(config *Config) error {
	// Try AWS S3 configuration
	if awsKey := os.Getenv("AWS_ACCESS_KEY_ID"); awsKey != "" {
		config.Providers["aws"] = ProviderConfig{
			Type:      "s3",
			Region:    getEnvOrDefault("AWS_REGION", "us-east-1"),
			AccessKey: awsKey,
			SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			Bucket:    os.Getenv("AWS_BUCKET"),
		}
	}

	// Try DigitalOcean Spaces configuration
	if doKey := os.Getenv("DO_ACCESS_KEY_ID"); doKey != "" {
		config.Providers["digitalocean"] = ProviderConfig{
			Type:      "digitalocean",
			Region:    getEnvOrDefault("DO_REGION", "nyc3"),
			Endpoint:  fmt.Sprintf("https://%s.digitaloceanspaces.com", getEnvOrDefault("DO_REGION", "nyc3")),
			AccessKey: doKey,
			SecretKey: os.Getenv("DO_SECRET_ACCESS_KEY"),
			Bucket:    os.Getenv("DO_BUCKET"),
		}
	}

	return nil
}

// SaveConfig saves the current configuration to a file
func (c *Config) SaveConfig(filename string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("error marshaling config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
} 