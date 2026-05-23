package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func setMinimalEnv(t *testing.T) {
	t.Helper()
	t.Setenv("MIDTRANS_SERVER_KEY", "SB-Mid-server-x")
	t.Setenv("MIDTRANS_CLIENT_KEY", "SB-Mid-client-x")
	t.Setenv("MIDTRANS_MERCHANT_ID", "Gtest")
}

func TestLoad_Defaults(t *testing.T) {
	setMinimalEnv(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://api.sandbox.midtrans.com", cfg.MidtransAPIBase)
	require.True(t, cfg.CustomerRequired)
	require.Equal(t, []string{"other_qris"}, cfg.EnabledPayments)
	require.Equal(t, 15, cfg.ExpiryDuration)
	require.Equal(t, "minutes", cfg.ExpiryUnit)
	require.Equal(t, 8080, cfg.Port)
	require.Equal(t, "info", cfg.LogLevel)
	require.Equal(t, "development", cfg.AppEnv)
	require.Equal(t, 60, cfg.RateLimitPerMin)
	require.True(t, cfg.IsDevelopment())
	require.False(t, cfg.IsProduction())
}

func TestLoad_InvalidEnums(t *testing.T) {
	t.Run("expiry unit", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("EXPIRY_UNIT", "minute") // singular is rejected
		_, err := Load()
		require.ErrorContains(t, err, "EXPIRY_UNIT")
	})
	t.Run("app env", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("APP_ENV", "staging")
		_, err := Load()
		require.ErrorContains(t, err, "APP_ENV")
	})
	t.Run("log level", func(t *testing.T) {
		setMinimalEnv(t)
		t.Setenv("LOG_LEVEL", "verbose")
		_, err := Load()
		require.ErrorContains(t, err, "LOG_LEVEL")
	})
}

func TestLoad_EnabledPaymentsCSV(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("ENABLED_PAYMENTS", "other_qris,gopay,shopeepay")
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"other_qris", "gopay", "shopeepay"}, cfg.EnabledPayments)
}

func TestLoad_ProductionFlag(t *testing.T) {
	setMinimalEnv(t)
	t.Setenv("APP_ENV", "production")
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.IsProduction())
	require.False(t, cfg.IsDevelopment())
}
