package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

const validDatabaseURL = "postgres://agentdock:secret@localhost:5432/agentdock?sslmode=disable"

func TestLoadUsesDefaults(t *testing.T) {
	t.Parallel()

	cfg, err := load(mapLookup(map[string]string{
		EnvDatabaseURL: validDatabaseURL,
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.HTTP.Address != DefaultHTTPAddress {
		t.Errorf(
			"HTTP.Address = %q, want %q",
			cfg.HTTP.Address,
			DefaultHTTPAddress,
		)
	}

	if cfg.HTTP.ReadHeaderTimeout != DefaultHTTPReadHeaderTimeout {
		t.Errorf(
			"HTTP.ReadHeaderTimeout = %s, want %s",
			cfg.HTTP.ReadHeaderTimeout,
			DefaultHTTPReadHeaderTimeout,
		)
	}

	if cfg.Log.Level != slog.LevelInfo {
		t.Errorf("Log.Level = %s, want INFO", cfg.Log.Level)
	}

	if cfg.Database.URL != validDatabaseURL {
		t.Errorf(
			"Database.URL = %q, want %q",
			cfg.Database.URL,
			validDatabaseURL,
		)
	}

	if cfg.Database.MaxConns != DefaultDatabaseMaxConns {
		t.Errorf(
			"Database.MaxConns = %d, want %d",
			cfg.Database.MaxConns,
			DefaultDatabaseMaxConns,
		)
	}

	if cfg.Database.MinConns != DefaultDatabaseMinConns {
		t.Errorf(
			"Database.MinConns = %d, want %d",
			cfg.Database.MinConns,
			DefaultDatabaseMinConns,
		)
	}

	if cfg.Database.ConnectTimeout != DefaultDatabaseConnectTimeout {
		t.Errorf(
			"Database.ConnectTimeout = %s, want %s",
			cfg.Database.ConnectTimeout,
			DefaultDatabaseConnectTimeout,
		)
	}

	if cfg.ShutdownTimeout != DefaultShutdownTimeout {
		t.Errorf(
			"ShutdownTimeout = %s, want %s",
			cfg.ShutdownTimeout,
			DefaultShutdownTimeout,
		)
	}
}

func TestLoadOverridesValues(t *testing.T) {
	t.Parallel()

	cfg, err := load(mapLookup(map[string]string{
		EnvHTTPAddress:            "127.0.0.1:9090",
		EnvHTTPReadHeaderTimeout:  "2s",
		EnvHTTPReadTimeout:        "4s",
		EnvHTTPWriteTimeout:       "8s",
		EnvHTTPIdleTimeout:        "45s",
		EnvLogLevel:               "debug",
		EnvDatabaseURL:            validDatabaseURL,
		EnvDatabaseMaxConns:       "24",
		EnvDatabaseMinConns:       "4",
		EnvDatabaseConnectTimeout: "7s",
		EnvShutdownTimeout:        "20s",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}

	if cfg.HTTP.Address != "127.0.0.1:9090" {
		t.Errorf("HTTP.Address = %q", cfg.HTTP.Address)
	}

	if cfg.HTTP.ReadHeaderTimeout != 2*time.Second {
		t.Errorf("HTTP.ReadHeaderTimeout = %s", cfg.HTTP.ReadHeaderTimeout)
	}

	if cfg.HTTP.ReadTimeout != 4*time.Second {
		t.Errorf("HTTP.ReadTimeout = %s", cfg.HTTP.ReadTimeout)
	}

	if cfg.HTTP.WriteTimeout != 8*time.Second {
		t.Errorf("HTTP.WriteTimeout = %s", cfg.HTTP.WriteTimeout)
	}

	if cfg.HTTP.IdleTimeout != 45*time.Second {
		t.Errorf("HTTP.IdleTimeout = %s", cfg.HTTP.IdleTimeout)
	}

	if cfg.Log.Level != slog.LevelDebug {
		t.Errorf("Log.Level = %s, want DEBUG", cfg.Log.Level)
	}

	if cfg.Database.MaxConns != 24 {
		t.Errorf("Database.MaxConns = %d, want 24", cfg.Database.MaxConns)
	}

	if cfg.Database.MinConns != 4 {
		t.Errorf("Database.MinConns = %d, want 4", cfg.Database.MinConns)
	}

	if cfg.Database.ConnectTimeout != 7*time.Second {
		t.Errorf(
			"Database.ConnectTimeout = %s",
			cfg.Database.ConnectTimeout,
		)
	}

	if cfg.ShutdownTimeout != 20*time.Second {
		t.Errorf("ShutdownTimeout = %s", cfg.ShutdownTimeout)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		values      map[string]string
		wantErrText string
	}{
		{
			name:        "missing database URL",
			values:      map[string]string{},
			wantErrText: EnvDatabaseURL + " is required",
		},
		{
			name: "empty value",
			values: map[string]string{
				EnvDatabaseURL: validDatabaseURL,
				EnvHTTPAddress: "",
			},
			wantErrText: EnvHTTPAddress + " must not be empty",
		},
		{
			name: "invalid HTTP address",
			values: map[string]string{
				EnvDatabaseURL: validDatabaseURL,
				EnvHTTPAddress: "8080",
			},
			wantErrText: EnvHTTPAddress + " must use host:port format",
		},
		{
			name: "invalid HTTP port",
			values: map[string]string{
				EnvDatabaseURL: validDatabaseURL,
				EnvHTTPAddress: ":70000",
			},
			wantErrText: EnvHTTPAddress + " port must be between 1 and 65535",
		},
		{
			name: "invalid log level",
			values: map[string]string{
				EnvDatabaseURL: validDatabaseURL,
				EnvLogLevel:    "verbose",
			},
			wantErrText: EnvLogLevel + " must be one of",
		},
		{
			name: "invalid duration",
			values: map[string]string{
				EnvDatabaseURL:     validDatabaseURL,
				EnvShutdownTimeout: "later",
			},
			wantErrText: EnvShutdownTimeout + " must be a valid duration",
		},
		{
			name: "zero duration",
			values: map[string]string{
				EnvDatabaseURL:     validDatabaseURL,
				EnvShutdownTimeout: "0s",
			},
			wantErrText: EnvShutdownTimeout + " must be greater than zero",
		},
		{
			name: "invalid integer",
			values: map[string]string{
				EnvDatabaseURL:      validDatabaseURL,
				EnvDatabaseMaxConns: "many",
			},
			wantErrText: EnvDatabaseMaxConns + " must be a valid integer",
		},
		{
			name: "maximum connections out of range",
			values: map[string]string{
				EnvDatabaseURL:      validDatabaseURL,
				EnvDatabaseMaxConns: "101",
			},
			wantErrText: EnvDatabaseMaxConns + " must be between 1 and 100",
		},
		{
			name: "minimum exceeds maximum",
			values: map[string]string{
				EnvDatabaseURL:      validDatabaseURL,
				EnvDatabaseMaxConns: "5",
				EnvDatabaseMinConns: "6",
			},
			wantErrText: EnvDatabaseMinConns + " must not exceed",
		},
		{
			name: "invalid database scheme",
			values: map[string]string{
				EnvDatabaseURL: "mysql://localhost/agentdock",
			},
			wantErrText: EnvDatabaseURL + " scheme must be postgres or postgresql",
		},
		{
			name: "database host missing",
			values: map[string]string{
				EnvDatabaseURL: "postgres:///agentdock",
			},
			wantErrText: EnvDatabaseURL + " must contain a host",
		},
		{
			name: "database name missing",
			values: map[string]string{
				EnvDatabaseURL: "postgres://localhost",
			},
			wantErrText: EnvDatabaseURL + " must contain a database name",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := load(mapLookup(tt.values))
			if err == nil {
				t.Fatal("load() error = nil, want non-nil")
			}

			if !strings.Contains(err.Error(), tt.wantErrText) {
				t.Fatalf(
					"load() error = %q, want substring %q",
					err.Error(),
					tt.wantErrText,
				)
			}
		})
	}
}

func TestStringValueDistinguishesMissingAndEmpty(t *testing.T) {
	t.Parallel()

	value, err := stringValue(
		mapLookup(map[string]string{}),
		"OPTIONAL_VALUE",
		"default",
		false,
	)
	if err != nil {
		t.Fatalf("stringValue() error = %v", err)
	}

	if value != "default" {
		t.Fatalf("stringValue() = %q, want default", value)
	}

	_, err = stringValue(
		mapLookup(map[string]string{
			"OPTIONAL_VALUE": "",
		}),
		"OPTIONAL_VALUE",
		"default",
		false,
	)
	if err == nil {
		t.Fatal("stringValue() accepted an explicitly empty value")
	}
}

func mapLookup(values map[string]string) LookupEnv {
	return func(name string) (string, bool) {
		value, exists := values[name]
		return value, exists
	}
}
