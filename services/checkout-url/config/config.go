// Package config loads, validates, and exposes environment-driven configuration.
//
// Load() must be called once at process startup. It fails fast if any required
// env var is missing or malformed so misconfiguration surfaces before any
// request flows through the system.
package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all runtime configuration. Tags are read by caarlos0/env.
type Config struct {
	MidtransServerKey  string `env:"MIDTRANS_SERVER_KEY,required"`
	MidtransClientKey  string `env:"MIDTRANS_CLIENT_KEY,required"`
	MidtransMerchantID string `env:"MIDTRANS_MERCHANT_ID,required"`
	MidtransAPIBase    string `env:"MIDTRANS_API_BASE" envDefault:"https://api.sandbox.midtrans.com"`

	// Midtrans Payment Link tunables. We intentionally do NOT send a
	// customer_details object — sandbox rejects empty/partial customer objects
	// with a 400. CustomerRequired=true alone is enough to make Midtrans show
	// the buyer-info form on the payment page.
	CustomerRequired bool     `env:"CUSTOMER_REQUIRED" envDefault:"true"`
	EnabledPayments  []string `env:"ENABLED_PAYMENTS" envSeparator:"," envDefault:"other_qris"`
	ExpiryDuration   int      `env:"EXPIRY_DURATION" envDefault:"15"`
	ExpiryUnit       string   `env:"EXPIRY_UNIT" envDefault:"minutes"`

	// Meta Catalog (Facebook Commerce) — source of truth for product name/price.
	// MetaAccessToken is a System User token with catalog_management scope.
	MetaCatalogID    string `env:"META_CATALOG_ID,required"`
	MetaAccessToken  string `env:"META_ACCESS_TOKEN,required"`
	MetaGraphAPIBase string `env:"META_GRAPH_API_BASE" envDefault:"https://graph.facebook.com"`
	MetaGraphVersion string `env:"META_GRAPH_VERSION" envDefault:"v25.0"`

	Port            int    `env:"PORT" envDefault:"8080"`
	LogLevel        string `env:"LOG_LEVEL" envDefault:"info"`
	AppEnv          string `env:"APP_ENV" envDefault:"development"`
	RateLimitPerMin int    `env:"RATE_LIMIT_PER_MIN" envDefault:"60"`
}

// Load reads .env (best-effort; ignored in container envs without one), parses
// environment variables into Config via struct tags, then validates enum-style
// fields that env tags cannot constrain.
func Load() (*Config, error) {
	_ = godotenv.Load()
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	switch c.ExpiryUnit {
	case "minutes", "hours", "days":
	default:
		return fmt.Errorf("EXPIRY_UNIT must be one of minutes|hours|days, got %q", c.ExpiryUnit)
	}
	switch c.AppEnv {
	case "development", "test", "production":
	default:
		return fmt.Errorf("APP_ENV must be one of development|test|production, got %q", c.AppEnv)
	}
	switch c.LogLevel {
	case "trace", "debug", "info", "warn", "error", "fatal", "panic":
	default:
		return fmt.Errorf("LOG_LEVEL must be a zerolog level, got %q", c.LogLevel)
	}
	if len(c.EnabledPayments) == 0 {
		return fmt.Errorf("ENABLED_PAYMENTS must contain at least one entry")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("PORT must be 1..65535, got %d", c.Port)
	}
	if c.ExpiryDuration <= 0 {
		return fmt.Errorf("EXPIRY_DURATION must be > 0, got %d", c.ExpiryDuration)
	}
	if c.RateLimitPerMin <= 0 {
		return fmt.Errorf("RATE_LIMIT_PER_MIN must be > 0, got %d", c.RateLimitPerMin)
	}
	if !strings.HasPrefix(c.MetaGraphVersion, "v") {
		return fmt.Errorf(`META_GRAPH_VERSION must start with "v" (e.g., "v25.0"), got %q`, c.MetaGraphVersion)
	}
	return nil
}

// IsDevelopment returns true when APP_ENV=development.
func (c *Config) IsDevelopment() bool { return c.AppEnv == "development" }

// IsProduction returns true when APP_ENV=production.
func (c *Config) IsProduction() bool { return c.AppEnv == "production" }
