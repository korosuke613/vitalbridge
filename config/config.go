package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the entire application configuration.
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Auth    AuthConfig    `yaml:"auth"`
	Metrics MetricsConfig `yaml:"metrics"`
	Log     LogConfig     `yaml:"log"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	APIKey string `yaml:"api_key"`
}

// MetricsConfig holds metrics retention settings.
type MetricsConfig struct {
	TTLHours               int `yaml:"ttl_hours"`
	CleanupIntervalMinutes int `yaml:"cleanup_interval_minutes"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// SlogLevel converts the Level string to slog.Level.
func (lc *LogConfig) SlogLevel() slog.Level {
	switch strings.ToLower(lc.Level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Load reads and parses the configuration file at the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	content := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.Server.Port)
	}

	if c.Metrics.TTLHours <= 0 {
		c.Metrics.TTLHours = 48
	}

	if c.Metrics.CleanupIntervalMinutes <= 0 {
		c.Metrics.CleanupIntervalMinutes = 60
	}

	if c.Auth.APIKey == "" {
		return fmt.Errorf("auth.api_key is required (set HEALTH_INGEST_API_KEY environment variable)")
	}

	switch strings.ToLower(c.Log.Level) {
	case "debug", "info", "warn", "error":
		// OK
	default:
		return fmt.Errorf("invalid log level: %q", c.Log.Level)
	}

	switch strings.ToLower(c.Log.Format) {
	case "json", "text":
		// OK
	case "":
		c.Log.Format = "json"
	default:
		return fmt.Errorf("invalid log format: %q (must be json or text)", c.Log.Format)
	}

	return nil
}
