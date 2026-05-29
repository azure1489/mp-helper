// Package config 加载 mp-server 的运行配置（YAML + 环境变量覆盖）。
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type CacheConfig struct {
	Type string `yaml:"type"`
}

type Config struct {
	Listen               string      `yaml:"listen"`
	DBPath               string      `yaml:"db_path"`
	AdminToken           string      `yaml:"admin_token"`
	Cache                CacheConfig `yaml:"cache"`
	WechatTimeoutSeconds int         `yaml:"wechat_timeout_seconds"`
}

// Default 返回带默认值的配置。
func Default() *Config {
	return &Config{
		Listen:               ":8080",
		DBPath:               "./mp-helper.db",
		Cache:                CacheConfig{Type: "memory"},
		WechatTimeoutSeconds: 30,
	}
}

// Load 读取 YAML（path 为空则跳过文件），再用环境变量覆盖，最后校验。
func Load(path string) (*Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}
	applyEnv(cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("MP_HELPER_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := os.Getenv("MP_HELPER_DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("MP_HELPER_ADMIN_TOKEN"); v != "" {
		cfg.AdminToken = v
	}
}

func (c *Config) validate() error {
	if c.AdminToken == "" {
		return fmt.Errorf("admin_token must be set (config file or MP_HELPER_ADMIN_TOKEN)")
	}
	if c.Cache.Type != "memory" {
		return fmt.Errorf("cache.type %q not supported in v1 (only \"memory\")", c.Cache.Type)
	}
	return nil
}
