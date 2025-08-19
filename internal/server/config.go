package server

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RootConfig struct {
	Server struct {
		Listen    string `yaml:"listen"`
		TLS       struct {
			Enabled  bool   `yaml:"enabled"`
			CertFile string `yaml:"cert_file"`
			KeyFile  string `yaml:"key_file"`
		} `yaml:"tls"`
		Shards             int `yaml:"shards"`
		QueueSize          int `yaml:"queue_size"`
		ValidatorsPerShard int `yaml:"validators_per_shard"`
		Batch              struct {
			Size     int `yaml:"size"`
			LingerMS int `yaml:"linger_ms"`
		} `yaml:"batch"`
	} `yaml:"server"`

	Logging struct {
		Level string `yaml:"level"`
		File  string `yaml:"file"`
	} `yaml:"logging"`

	Metrics struct {
		PrometheusListen string `yaml:"prometheus_listen"`
	} `yaml:"metrics"`

	Tracing struct {
		OTLPEndpoint string  `yaml:"otlp_endpoint"`
		SampleRatio  float64 `yaml:"sample_ratio"`
	} `yaml:"tracing"`

	Zoom struct {
		SecretEnv string `yaml:"secret_env"`
	} `yaml:"zoom"`

	Outputs []OutputConfig `yaml:"outputs"`
}

type OutputConfig struct {
	Type string `yaml:"type"` // "file" | "http"

	// file
	Path     string `yaml:"path,omitempty"`
	Rotation struct {
		MaxSizeMB  int  `yaml:"max_size_mb"`
		MaxBackups int  `yaml:"max_backups"`
		MaxAgeDays int  `yaml:"max_age_days"`
		Compress   bool `yaml:"compress"`
	} `yaml:"rotation,omitempty"`

	// http
	URL         string `yaml:"url,omitempty"`
	TimeoutMS   int    `yaml:"timeout_ms,omitempty"`
	Parallel    int    `yaml:"parallel,omitempty"`
	HECTokenEnv string `yaml:"hec_token_env,omitempty"`
}

func LoadConfig(path string) (RootConfig, error) {
	var cfg RootConfig
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("yaml: %w", err)
	}
	// defaults
	if cfg.Server.Shards <= 0 {
		cfg.Server.Shards = 4
	}
	if cfg.Server.QueueSize <= 0 {
		cfg.Server.QueueSize = 4096
	}
	if cfg.Server.ValidatorsPerShard <= 0 {
		cfg.Server.ValidatorsPerShard = 1
	}
	if cfg.Server.Batch.Size <= 0 {
		cfg.Server.Batch.Size = 256
	}
	if cfg.Server.Batch.LingerMS <= 0 {
		cfg.Server.Batch.LingerMS = 1
	}
	if cfg.Tracing.SampleRatio < 0 || cfg.Tracing.SampleRatio > 1 {
		cfg.Tracing.SampleRatio = 0.001
	}
	return cfg, nil
}
