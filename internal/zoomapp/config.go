package zoomapp

import (
	"os"
	"gopkg.in/yaml.v3"
)

type Config struct {
	CRC struct {
		RatePerSec float64 `yaml:"rate_per_sec"`
		Burst      int     `yaml:"burst"`
	} `yaml:"crc"`

	LegacySignatureFallback bool `yaml:"legacy_signature_fallback"`

	Fastpath struct {
		Enabled       bool   `yaml:"enabled"`
		Shards        int    `yaml:"shards"`
		RingSize      int    `yaml:"ring_size"`
		ValidatorsPer int    `yaml:"validators_per_shard"`
		BatchSize     int    `yaml:"batch_size"`
		BatchLingerMS int    `yaml:"batch_linger_ms"`
		BaseDir       string `yaml:"base_dir"`
	} `yaml:"fastpath"`
}

func Load(path string) (Config, error) {
	var root map[string]any
	var out Config
	b, err := os.ReadFile(path)
	if err != nil { return out, err }
	if err := yaml.Unmarshal(b, &root); err != nil { return out, err }
	if z, ok := root["zoom_app"]; ok {
		b2, _ := yaml.Marshal(z)
		_ = yaml.Unmarshal(b2, &out)
	}
	// defaults
	if out.CRC.RatePerSec <= 0 { out.CRC.RatePerSec = 5 }
	if out.CRC.Burst <= 0 { out.CRC.Burst = 10 }
	if out.Fastpath.Shards == 0 { out.Fastpath.Shards = 4 }
	if out.Fastpath.RingSize == 0 { out.Fastpath.RingSize = 4096 }
	if out.Fastpath.ValidatorsPer == 0 { out.Fastpath.ValidatorsPer = 1 }
	if out.Fastpath.BatchSize == 0 { out.Fastpath.BatchSize = 128 }
	if out.Fastpath.BatchLingerMS == 0 { out.Fastpath.BatchLingerMS = 2 }
	if out.Fastpath.BaseDir == "" { out.Fastpath.BaseDir = "data/validated.fast.dev" }
	return out, nil
}
