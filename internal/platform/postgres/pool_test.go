package postgres

import (
	"testing"
	"time"

	"github.com/wuge-xu/agentdock/internal/config"
)

func TestNewPoolConfigUsesDatabaseConfiguration(
	t *testing.T,
) {
	t.Parallel()

	databaseConfig := config.DatabaseConfig{
		URL:            "postgres://agentdock:secret@127.0.0.1:5432/agentdock?sslmode=disable",
		MaxConns:       12,
		MinConns:       3,
		ConnectTimeout: 4 * time.Second,
	}

	poolConfig, err := newPoolConfig(
		databaseConfig,
	)
	if err != nil {
		t.Fatalf(
			"newPoolConfig() error = %v",
			err,
		)
	}

	if poolConfig.MaxConns != databaseConfig.MaxConns {
		t.Fatalf(
			"MaxConns = %d, want %d",
			poolConfig.MaxConns,
			databaseConfig.MaxConns,
		)
	}

	if poolConfig.MinConns != databaseConfig.MinConns {
		t.Fatalf(
			"MinConns = %d, want %d",
			poolConfig.MinConns,
			databaseConfig.MinConns,
		)
	}

	if poolConfig.ConnConfig.ConnectTimeout !=
		databaseConfig.ConnectTimeout {
		t.Fatalf(
			"ConnectTimeout = %s, want %s",
			poolConfig.ConnConfig.ConnectTimeout,
			databaseConfig.ConnectTimeout,
		)
	}

	if poolConfig.ConnConfig.Host != "127.0.0.1" {
		t.Fatalf(
			"Host = %q, want 127.0.0.1",
			poolConfig.ConnConfig.Host,
		)
	}

	if poolConfig.ConnConfig.Port != 5432 {
		t.Fatalf(
			"Port = %d, want 5432",
			poolConfig.ConnConfig.Port,
		)
	}

	if poolConfig.ConnConfig.Database != "agentdock" {
		t.Fatalf(
			"Database = %q, want agentdock",
			poolConfig.ConnConfig.Database,
		)
	}

	if poolConfig.ConnConfig.User != "agentdock" {
		t.Fatalf(
			"User = %q, want agentdock",
			poolConfig.ConnConfig.User,
		)
	}
}

func TestNewPoolConfigRejectsInvalidDatabaseURL(
	t *testing.T,
) {
	t.Parallel()

	databaseConfig := config.DatabaseConfig{
		URL:            "postgres://%",
		MaxConns:       10,
		MinConns:       1,
		ConnectTimeout: 5 * time.Second,
	}

	poolConfig, err := newPoolConfig(
		databaseConfig,
	)

	if err == nil {
		t.Fatal(
			"newPoolConfig() error = nil, want error",
		)
	}

	if poolConfig != nil {
		t.Fatal(
			"newPoolConfig() returned non-nil config on error",
		)
	}
}
