package pg

import (
	"context"
	"errors"
	"time"

	"github.com/geruz/rizotto/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	ErrConnectionFailed = errors.New("failed to connect to database")
)

const tryCount = 10
const sleepBetweenRetries = 1 * time.Second

func OpenConnection(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	config.ConnConfig.Tracer = pgTracer{}
	for range tryCount {
		pool, err := pgxpool.NewWithConfig(ctx, config)

		if err == nil {
			return pool, nil
		}

		logger.Error(ctx, "Failed to connect to database, retrying...", err)
		time.Sleep(sleepBetweenRetries)
	}

	return nil, ErrConnectionFailed
}

func MustOpenConnection(ctx context.Context, connString string) *pgxpool.Pool {
	pool, err := OpenConnection(ctx, connString)
	if err != nil {
		panic(err)
	}

	return pool
}
