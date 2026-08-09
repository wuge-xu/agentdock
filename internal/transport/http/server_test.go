package httptransport

import (
	"net/http"
	"testing"
	"time"

	"github.com/wuge-xu/agentdock/internal/config"
)

func TestNewServerUsesHTTPConfiguration(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(
		func(
			writer http.ResponseWriter,
			_ *http.Request,
		) {
			writer.WriteHeader(http.StatusNoContent)
		},
	)

	httpConfig := config.HTTPConfig{
		Address:           "127.0.0.1:18080",
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      7 * time.Second,
		IdleTimeout:       30 * time.Second,
	}

	server := NewServer(
		httpConfig,
		handler,
	)

	if server.Addr != httpConfig.Address {
		t.Fatalf(
			"Addr = %q, want %q",
			server.Addr,
			httpConfig.Address,
		)
	}

	if server.Handler == nil {
		t.Fatal("Handler = nil, want non-nil")
	}

	if server.ReadHeaderTimeout != httpConfig.ReadHeaderTimeout {
		t.Fatalf(
			"ReadHeaderTimeout = %s, want %s",
			server.ReadHeaderTimeout,
			httpConfig.ReadHeaderTimeout,
		)
	}

	if server.ReadTimeout != httpConfig.ReadTimeout {
		t.Fatalf(
			"ReadTimeout = %s, want %s",
			server.ReadTimeout,
			httpConfig.ReadTimeout,
		)
	}

	if server.WriteTimeout != httpConfig.WriteTimeout {
		t.Fatalf(
			"WriteTimeout = %s, want %s",
			server.WriteTimeout,
			httpConfig.WriteTimeout,
		)
	}

	if server.IdleTimeout != httpConfig.IdleTimeout {
		t.Fatalf(
			"IdleTimeout = %s, want %s",
			server.IdleTimeout,
			httpConfig.IdleTimeout,
		)
	}
}
