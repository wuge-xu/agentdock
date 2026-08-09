package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wuge-xu/agentdock/internal/config"
)

func NewPool(
	ctx context.Context,
	databaseConfig config.DatabaseConfig,
) (*pgxpool.Pool, error) {
	poolConfig, err := newPoolConfig(
		databaseConfig,
	)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(
		ctx,
		poolConfig,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create PostgreSQL pool: %w",
			err,
		)
	}

	return pool, nil
}

func newPoolConfig(
	databaseConfig config.DatabaseConfig,
) (*pgxpool.Config, error) {
	poolConfig, err := pgxpool.ParseConfig(
		databaseConfig.URL,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"parse PostgreSQL pool configuration: %w",
			err,
		)
	}

	poolConfig.MaxConns = databaseConfig.MaxConns
	poolConfig.MinConns = databaseConfig.MinConns
	poolConfig.ConnConfig.ConnectTimeout =
		databaseConfig.ConnectTimeout

	return poolConfig, nil
}
