package server

import (
	"os"
	"gopkg.in/yaml.v3"
)

type TLSCfg struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}
type ServerCfg struct {
	Addr            string `yaml:"addr"`
	BasePath        string `yaml:"base_path"`
	ReadTimeoutMS   int    `yaml:"read_timeout_ms"`
	WriteTimeoutMS  int    `yaml:"write_timeout_ms"`
	TLS             TLSCfg `yaml:"tls"`
}
type LoggingCfg struct {
	Level     string `yaml:"level"`
	File      string `yaml:"file"`
	TraceFile string `yaml:"trace_file"`
}
type MetricsCfg struct {
	Enable bool   `yaml:"enable"`
	Path   string `yaml:"path"`
}
type TracingCfg struct {
	SampleRatio  float64 `yaml:"sample_ratio"`
	OTLPEndpoint string  `yaml:"otlp_endpoint"`
}
type ValidatorsCfg struct {
	Zoom struct {
		Secret string `yaml:"secret"`
	} `yaml:"zoom"`
}
type RootConfig struct {
	Server     ServerCfg     `yaml:"server"`
	Logging    LoggingCfg    `yaml:"logging"`
	Metrics    MetricsCfg    `yaml:"metrics"`
	Tracing    TracingCfg    `yaml:"tracing"`
	Validators ValidatorsCfg `yaml:"validators"`
}

func LoadRootConfig(path string) (RootConfig, error) {
	var root RootConfig
	b, err := os.ReadFile(path)
	if err != nil { return root, err }
	if err := yaml.Unmarshal(b, &root); err != nil { return root, err }
	return root, nil
}
