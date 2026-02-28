package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 全体の設定構造
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Auth    AuthConfig    `yaml:"auth"`
	Metrics MetricsConfig `yaml:"metrics"`
	Log     LogConfig     `yaml:"log"`
}

// ServerConfig サーバー設定
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// AuthConfig 認証設定
type AuthConfig struct {
	APIKey string `yaml:"api_key"`
}

// MetricsConfig メトリクス設定
type MetricsConfig struct {
	TTLHours               int `yaml:"ttl_hours"`
	CleanupIntervalMinutes int `yaml:"cleanup_interval_minutes"`
}

// LogConfig ログ設定
type LogConfig struct {
	Level string `yaml:"level"`
}

// Load 設定ファイルを読み込む
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗しました: %w", err)
	}

	// 環境変数を展開
	content := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(content), &cfg); err != nil {
		return nil, fmt.Errorf("設定ファイルのパースに失敗しました: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("設定の検証に失敗しました: %w", err)
	}

	return &cfg, nil
}

// validate 設定の妥当性をチェック
func (c *Config) validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("無効なポート番号です: %d", c.Server.Port)
	}

	if c.Metrics.TTLHours <= 0 {
		c.Metrics.TTLHours = 48
	}

	if c.Metrics.CleanupIntervalMinutes <= 0 {
		c.Metrics.CleanupIntervalMinutes = 60
	}

	if c.Auth.APIKey == "" {
		return fmt.Errorf("auth.api_key は必須です。環境変数 HEALTH_INGEST_API_KEY を設定してください")
	}

	validLogLevels := []string{"debug", "info", "warn", "error"}
	found := false
	for _, l := range validLogLevels {
		if l == c.Log.Level {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("無効なログレベルです: %s", c.Log.Level)
	}

	return nil
}
