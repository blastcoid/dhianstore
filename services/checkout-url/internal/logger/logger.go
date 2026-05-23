// Package logger constructs a zerolog.Logger from Config.
//
// In development we render via ConsoleWriter (colored, human-readable). In
// test/production we emit structured JSON for log aggregators.
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/blastcoid/dhianstore/services/checkout-url/internal/config"
)

// New returns a configured zerolog.Logger. LogLevel is validated upstream.
func New(cfg *config.Config) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)
	zerolog.TimeFieldFormat = time.RFC3339Nano

	if cfg.IsDevelopment() {
		w := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05.000"}
		return zerolog.New(w).With().Timestamp().Logger()
	}
	return zerolog.New(os.Stdout).With().Timestamp().Logger()
}
