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

func TestNew_ProductionReturnsUsableLogger(t *testing.T) {
	cfg := &config.Config{LogLevel: "info", AppEnv: "production"}
	log := New(cfg)
	require.NotNil(t, log)
	log.Info().Msg("smoke")
	require.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestNew_DevelopmentReturnsUsableLogger(t *testing.T) {
	cfg := &config.Config{LogLevel: "debug", AppEnv: "development"}
	log := New(cfg)
	require.NotNil(t, log)
	log.Info().Msg("smoke")
	require.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}
