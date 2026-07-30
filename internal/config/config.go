package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	EnvHTTPAddress            = "AGENTDOCK_HTTP_ADDRESS"
	EnvHTTPReadHeaderTimeout  = "AGENTDOCK_HTTP_READ_HEADER_TIMEOUT"
	EnvHTTPReadTimeout        = "AGENTDOCK_HTTP_READ_TIMEOUT"
	EnvHTTPWriteTimeout       = "AGENTDOCK_HTTP_WRITE_TIMEOUT"
	EnvHTTPIdleTimeout        = "AGENTDOCK_HTTP_IDLE_TIMEOUT"
	EnvLogLevel               = "AGENTDOCK_LOG_LEVEL"
	EnvDatabaseURL            = "AGENTDOCK_DATABASE_URL"
	EnvDatabaseMaxConns       = "AGENTDOCK_DATABASE_MAX_CONNS"
	EnvDatabaseMinConns       = "AGENTDOCK_DATABASE_MIN_CONNS"
	EnvDatabaseConnectTimeout = "AGENTDOCK_DATABASE_CONNECT_TIMEOUT"
	EnvShutdownTimeout        = "AGENTDOCK_SHUTDOWN_TIMEOUT"
)

const (
	DefaultHTTPAddress            = ":8080"
	DefaultHTTPReadHeaderTimeout  = 5 * time.Second
	DefaultHTTPReadTimeout        = 15 * time.Second
	DefaultHTTPWriteTimeout       = 30 * time.Second
	DefaultHTTPIdleTimeout        = 60 * time.Second
	DefaultDatabaseMaxConns       = int32(10)
	DefaultDatabaseMinConns       = int32(1)
	DefaultDatabaseConnectTimeout = 5 * time.Second
	DefaultShutdownTimeout        = 10 * time.Second
)

type LookupEnv func(string) (string, bool)

type Config struct {
	HTTP            HTTPConfig
	Log             LogConfig
	Database        DatabaseConfig
	ShutdownTimeout time.Duration
}

type HTTPConfig struct {
	Address           string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type LogConfig struct {
	Level slog.Level
}

type DatabaseConfig struct {
	URL            string
	MaxConns       int32
	MinConns       int32
	ConnectTimeout time.Duration
}

func Load() (Config, error) {
	cfg, err := load(os.LookupEnv)
	if err != nil {
		return Config{}, fmt.Errorf("load configuration: %w", err)
	}

	return cfg, nil
}

func load(lookup LookupEnv) (Config, error) {
	httpAddress, err := stringValue(
		lookup,
		EnvHTTPAddress,
		DefaultHTTPAddress,
		false,
	)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := durationValue(
		lookup,
		EnvHTTPReadHeaderTimeout,
		DefaultHTTPReadHeaderTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := durationValue(
		lookup,
		EnvHTTPReadTimeout,
		DefaultHTTPReadTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := durationValue(
		lookup,
		EnvHTTPWriteTimeout,
		DefaultHTTPWriteTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := durationValue(
		lookup,
		EnvHTTPIdleTimeout,
		DefaultHTTPIdleTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	logLevelText, err := stringValue(
		lookup,
		EnvLogLevel,
		"INFO",
		false,
	)
	if err != nil {
		return Config{}, err
	}

	var logLevel slog.Level
	if err := logLevel.UnmarshalText(
		[]byte(strings.ToUpper(logLevelText)),
	); err != nil {
		return Config{}, fmt.Errorf(
			"%s must be one of DEBUG, INFO, WARN or ERROR: %w",
			EnvLogLevel,
			err,
		)
	}

	databaseURL, err := stringValue(
		lookup,
		EnvDatabaseURL,
		"",
		true,
	)
	if err != nil {
		return Config{}, err
	}

	maxConns, err := int32Value(
		lookup,
		EnvDatabaseMaxConns,
		DefaultDatabaseMaxConns,
	)
	if err != nil {
		return Config{}, err
	}

	minConns, err := int32Value(
		lookup,
		EnvDatabaseMinConns,
		DefaultDatabaseMinConns,
	)
	if err != nil {
		return Config{}, err
	}

	connectTimeout, err := durationValue(
		lookup,
		EnvDatabaseConnectTimeout,
		DefaultDatabaseConnectTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := durationValue(
		lookup,
		EnvShutdownTimeout,
		DefaultShutdownTimeout,
	)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTP: HTTPConfig{
			Address:           httpAddress,
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		Log: LogConfig{
			Level: logLevel,
		},
		Database: DatabaseConfig{
			URL:            databaseURL,
			MaxConns:       maxConns,
			MinConns:       minConns,
			ConnectTimeout: connectTimeout,
		},
		ShutdownTimeout: shutdownTimeout,
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validate configuration: %w", err)
	}

	return cfg, nil
}

func (c Config) Validate() error {
	var validationErrors []error

	if err := validateHTTPAddress(c.HTTP.Address); err != nil {
		validationErrors = append(validationErrors, err)
	}

	timeoutChecks := []struct {
		name  string
		value time.Duration
	}{
		{EnvHTTPReadHeaderTimeout, c.HTTP.ReadHeaderTimeout},
		{EnvHTTPReadTimeout, c.HTTP.ReadTimeout},
		{EnvHTTPWriteTimeout, c.HTTP.WriteTimeout},
		{EnvHTTPIdleTimeout, c.HTTP.IdleTimeout},
		{EnvDatabaseConnectTimeout, c.Database.ConnectTimeout},
		{EnvShutdownTimeout, c.ShutdownTimeout},
	}

	for _, check := range timeoutChecks {
		if check.value <= 0 {
			validationErrors = append(
				validationErrors,
				fmt.Errorf("%s must be greater than zero", check.name),
			)
		}
	}

	if err := validateDatabaseURL(c.Database.URL); err != nil {
		validationErrors = append(validationErrors, err)
	}

	if c.Database.MaxConns < 1 || c.Database.MaxConns > 100 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("%s must be between 1 and 100", EnvDatabaseMaxConns),
		)
	}

	if c.Database.MinConns < 0 {
		validationErrors = append(
			validationErrors,
			fmt.Errorf("%s must be greater than or equal to zero", EnvDatabaseMinConns),
		)
	}

	if c.Database.MinConns > c.Database.MaxConns {
		validationErrors = append(
			validationErrors,
			fmt.Errorf(
				"%s must not exceed %s",
				EnvDatabaseMinConns,
				EnvDatabaseMaxConns,
			),
		)
	}

	return errors.Join(validationErrors...)
}

func stringValue(
	lookup LookupEnv,
	name string,
	defaultValue string,
	required bool,
) (string, error) {
	rawValue, exists := lookup(name)
	if !exists {
		if required {
			return "", fmt.Errorf("%s is required", name)
		}

		return defaultValue, nil
	}

	value := strings.TrimSpace(rawValue)
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}

	return value, nil
}

func int32Value(
	lookup LookupEnv,
	name string,
	defaultValue int32,
) (int32, error) {
	value, err := stringValue(
		lookup,
		name,
		strconv.FormatInt(int64(defaultValue), 10),
		false,
	)
	if err != nil {
		return 0, err
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer: %w", name, err)
	}

	return int32(parsed), nil
}

func durationValue(
	lookup LookupEnv,
	name string,
	defaultValue time.Duration,
) (time.Duration, error) {
	value, err := stringValue(
		lookup,
		name,
		defaultValue.String(),
		false,
	)
	if err != nil {
		return 0, err
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid duration: %w", name, err)
	}

	return parsed, nil
}

func validateHTTPAddress(address string) error {
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf(
			"%s must use host:port format: %w",
			EnvHTTPAddress,
			err,
		)
	}

	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf(
			"%s port must be between 1 and 65535",
			EnvHTTPAddress,
		)
	}

	return nil
}

func validateDatabaseURL(databaseURL string) error {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("%s is invalid: %w", EnvDatabaseURL, err)
	}

	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf(
			"%s scheme must be postgres or postgresql",
			EnvDatabaseURL,
		)
	}

	if parsed.Host == "" {
		return fmt.Errorf("%s must contain a host", EnvDatabaseURL)
	}

	if parsed.Path == "" || parsed.Path == "/" {
		return fmt.Errorf("%s must contain a database name", EnvDatabaseURL)
	}

	return nil
}
