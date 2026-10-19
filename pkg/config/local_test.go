package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalKeepsCredentialsAndTelemetrySeparate(t *testing.T) {
	t.Setenv("ENVIRONMENT", "local")
	t.Setenv("DATABASE_PASSWORD", "local-database-secret")
	t.Setenv("JWT_SECRET", "local-session-secret")
	t.Setenv("DATABASE_NAME", "isolated_test_database")
	t.Setenv("PORT", "9898")
	cfg, err := New()
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", cfg.ListenHost)
	require.Equal(t, "127.0.0.1", cfg.DatabaseHost)
	require.Equal(t, "local-database-secret", cfg.DatabasePassword)
	require.Equal(t, []byte("local-session-secret"), cfg.JWTSecret)
	require.Equal(t, "isolated_test_database", cfg.DatabaseName)
	require.Equal(t, 9898, cfg.Port)
	require.Empty(t, cfg.GoogleAnalyticsID)
	require.Empty(t, cfg.RollbarClientToken)
	require.Empty(t, cfg.RollbarToken)
}
