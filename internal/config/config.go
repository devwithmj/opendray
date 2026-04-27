// Package config loads opendray's TOML configuration with environment-variable
// overrides. The TOML file is the human-edited source of truth; env vars
// (prefix OPENDRAY_) override individual fields for 12-factor deploys.
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Listen   string         `toml:"listen"`
	Database DatabaseConfig `toml:"database"`
	Admin    AdminConfig    `toml:"admin"`
	Log      LogConfig      `toml:"log"`
	Session  SessionConfig  `toml:"session"`
}

type DatabaseConfig struct {
	URL string `toml:"url"`
}

type AdminConfig struct {
	User     string `toml:"user"`
	Password string `toml:"password"`
}

type LogConfig struct {
	Level  string `toml:"level"`  // debug|info|warn|error
	Format string `toml:"format"` // json|text
}

// SessionConfig drives the session.Manager idle detector. Empty values
// use Manager defaults (30s threshold, 5s poll interval).
type SessionConfig struct {
	IdleThreshold string `toml:"idle_threshold"` // e.g. "30s", "2m"
	IdleInterval  string `toml:"idle_interval"`  // e.g. "5s"
}

// Threshold parses IdleThreshold; returns 0 if unset or invalid (caller
// should call Validate first to surface invalid values).
func (s SessionConfig) Threshold() time.Duration {
	d, _ := time.ParseDuration(s.IdleThreshold)
	return d
}

// Interval parses IdleInterval; returns 0 if unset.
func (s SessionConfig) Interval() time.Duration {
	d, _ := time.ParseDuration(s.IdleInterval)
	return d
}

func defaults() Config {
	return Config{
		Listen: "127.0.0.1:8770",
		Log:    LogConfig{Level: "info", Format: "text"},
	}
}

// Load reads the TOML file at path, then applies env overrides.
// An empty path skips file loading and uses defaults + env only.
func Load(path string) (Config, error) {
	cfg := defaults()
	if path != "" {
		if _, err := toml.DecodeFile(path, &cfg); err != nil {
			return cfg, fmt.Errorf("config: decode %s: %w", path, err)
		}
	}
	applyEnv(&cfg)
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("OPENDRAY_LISTEN"); v != "" {
		cfg.Listen = v
	}
	if v := os.Getenv("OPENDRAY_DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}
	if v := os.Getenv("OPENDRAY_ADMIN_USER"); v != "" {
		cfg.Admin.User = v
	}
	if v := os.Getenv("OPENDRAY_ADMIN_PASSWORD"); v != "" {
		cfg.Admin.Password = v
	}
	if v := os.Getenv("OPENDRAY_LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("OPENDRAY_LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv("OPENDRAY_SESSION_IDLE_THRESHOLD"); v != "" {
		cfg.Session.IdleThreshold = v
	}
	if v := os.Getenv("OPENDRAY_SESSION_IDLE_INTERVAL"); v != "" {
		cfg.Session.IdleInterval = v
	}
}

func (c Config) Validate() error {
	if c.Listen == "" {
		return errors.New("config: listen address is empty")
	}
	if c.Database.URL == "" {
		return errors.New("config: database.url is empty (set OPENDRAY_DATABASE_URL or [database].url)")
	}
	if c.Session.IdleThreshold != "" {
		if _, err := time.ParseDuration(c.Session.IdleThreshold); err != nil {
			return fmt.Errorf("config: session.idle_threshold: %w", err)
		}
	}
	if c.Session.IdleInterval != "" {
		if _, err := time.ParseDuration(c.Session.IdleInterval); err != nil {
			return fmt.Errorf("config: session.idle_interval: %w", err)
		}
	}
	return nil
}
