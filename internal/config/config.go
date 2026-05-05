package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	APIKey      string `yaml:"api_key"`
	DefaultTeam string `yaml:"default_team"`
}

// Load reads configuration from environment variables and config file.
// Priority: WLLINEAR_API_KEY > LAZYLINEAR_API_KEY > ~/.config/wllinear/config.yaml >
// ~/.config/lazylinear/config.yaml.
func Load() (*Config, error) {
	cfg := &Config{}

	for _, p := range []string{configPath("wllinear"), configPath("lazylinear")} {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config %s: %w", p, err)
		}
		break
	}

	if envKey := os.Getenv("WLLINEAR_API_KEY"); envKey != "" {
		cfg.APIKey = envKey
	} else if envKey := os.Getenv("LAZYLINEAR_API_KEY"); envKey != "" && cfg.APIKey == "" {
		cfg.APIKey = envKey
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf(
			"no API key found\n\n" +
				"Set your Linear API key using one of:\n" +
				"  1. export WLLINEAR_API_KEY=lin_api_...\n" +
				"  2. ~/.config/wllinear/config.yaml with `api_key: lin_api_...`\n\n" +
				"Get your API key at: https://linear.app/settings/api",
		)
	}
	return cfg, nil
}

func configPath(app string) string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, app, "config.yaml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", app, "config.yaml")
}
