package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MIDTRANS_SERVER_KEY", "SB-Mid-server-x")
	t.Setenv("MIDTRANS_CLIENT_KEY", "SB-Mid-client-x")
	t.Setenv("MIDTRANS_MERCHANT_ID", "Gtest")
	t.Setenv("META_CATALOG_ID", "1017309634048260")
	t.Setenv("META_ACCESS_TOKEN", "EAAtest-system-user-token")
}

func TestLoad_Defaults(t *testing.T) {
	setRequiredEnv(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://api.sandbox.midtrans.com", cfg.MidtransAPIBase)
	require.True(t, cfg.CustomerRequired)
	require.Equal(t, []string{"other_qris"}, cfg.EnabledPayments)
	require.Equal(t, 15, cfg.ExpiryDuration)
	require.Equal(t, "minutes", cfg.ExpiryUnit)
	require.Equal(t, "1017309634048260", cfg.MetaCatalogID)
	require.Equal(t, "EAAtest-system-user-token", cfg.MetaAccessToken)
	require.Equal(t, "https://graph.facebook.com", cfg.MetaGraphAPIBase)
	require.Equal(t, "v25.0", cfg.MetaGraphVersion)
	require.Equal(t, 8080, cfg.Port)
	require.Equal(t, "info", cfg.LogLevel)
	require.Equal(t, "development", cfg.AppEnv)
	require.Equal(t, 60, cfg.RateLimitPerMin)
	require.True(t, cfg.IsDevelopment())
	require.False(t, cfg.IsProduction())
}

func TestLoad_OverridesFromEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENABLED_PAYMENTS", "other_qris,gopay,shopeepay")
	t.Setenv("EXPIRY_DURATION", "30")
	t.Setenv("EXPIRY_UNIT", "hours")
	t.Setenv("APP_ENV", "production")
	t.Setenv("PORT", "9090")
	t.Setenv("RATE_LIMIT_PER_MIN", "120")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"other_qris", "gopay", "shopeepay"}, cfg.EnabledPayments)
	require.Equal(t, 30, cfg.ExpiryDuration)
	require.Equal(t, "hours", cfg.ExpiryUnit)
	require.True(t, cfg.IsProduction())
	require.Equal(t, 9090, cfg.Port)
	require.Equal(t, 120, cfg.RateLimitPerMin)
}

func TestLoad_InvalidEnums(t *testing.T) {
	cases := []struct {
		name    string
		envKey  string
		envVal  string
		wantMsg string
	}{
		{"expiry unit singular", "EXPIRY_UNIT", "minute", "EXPIRY_UNIT"},
		{"app env unknown", "APP_ENV", "staging", "APP_ENV"},
		{"log level unknown", "LOG_LEVEL", "verbose", "LOG_LEVEL"},
		{"graph version missing v prefix", "META_GRAPH_VERSION", "25.0", "META_GRAPH_VERSION"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tc.envKey, tc.envVal)
			_, err := Load()
			require.ErrorContains(t, err, tc.wantMsg)
		})
	}
}

func TestLoad_InvalidRanges(t *testing.T) {
	cases := []struct {
		name    string
		envKey  string
		envVal  string
		wantMsg string
	}{
		{"port too low", "PORT", "0", "PORT"},
		{"port too high", "PORT", "70000", "PORT"},
		{"expiry duration zero", "EXPIRY_DURATION", "0", "EXPIRY_DURATION"},
		{"rate limit zero", "RATE_LIMIT_PER_MIN", "0", "RATE_LIMIT_PER_MIN"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setRequiredEnv(t)
			t.Setenv(tc.envKey, tc.envVal)
			_, err := Load()
			require.ErrorContains(t, err, tc.wantMsg)
		})
	}
}
