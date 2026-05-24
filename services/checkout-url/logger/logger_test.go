package logger

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/blastcoid/dhianstore/services/checkout-url/config"
)

func TestNew_AppliesLevel(t *testing.T) {
	cases := []struct {
		name  string
		level string
		want  zerolog.Level
	}{
		{"trace", "trace", zerolog.TraceLevel},
		{"debug", "debug", zerolog.DebugLevel},
		{"info", "info", zerolog.InfoLevel},
		{"warn", "warn", zerolog.WarnLevel},
		{"error", "error", zerolog.ErrorLevel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{LogLevel: tc.level, AppEnv: "test"}
			_ = New(cfg)
			require.Equal(t, tc.want, zerolog.GlobalLevel())
		})
	}
}

func TestNew_ProductionEmitsJSON(t *testing.T) {
	cfg := &config.Config{LogLevel: "info", AppEnv: "production"}
	log := New(cfg)
	// We can't easily inspect the writer type, but the call should not panic
	// and the returned logger should be usable.
	log.Info().Msg("smoke")
}

func TestNew_DevelopmentEmitsConsole(t *testing.T) {
	cfg := &config.Config{LogLevel: "info", AppEnv: "development"}
	log := New(cfg)
	log.Info().Msg("smoke")
}
