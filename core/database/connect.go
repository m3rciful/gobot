package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"gobot/core/logger"
	"log/slog"
)

// Connect opens the database connection, configures the pool, and verifies connectivity.
func Connect(cfg Config) (*sqlx.DB, error) {
	dsn := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	sqlxDB, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	took := time.Since(start)
	if err != nil {
		logger.DB.Error("db connect failed",
			slog.String("event", "db.connect"),
			slog.String("driver", "postgres"),
			slog.String("host", cfg.Host),
			slog.String("port", cfg.Port),
			slog.String("db", cfg.Name),
			slog.Duration("duration", logger.RoundMS(took)),
			slog.String("err", err.Error()),
		)
		return nil, fmt.Errorf("db connect: %w", err)
	}

	if pingErr := sqlxDB.PingContext(ctx); pingErr != nil {
		logger.DB.Error("db ping failed",
			slog.String("event", "db.ping"),
			slog.String("driver", "postgres"),
			slog.String("host", cfg.Host),
			slog.String("port", cfg.Port),
			slog.String("db", cfg.Name),
			slog.String("err", pingErr.Error()),
		)
		return nil, fmt.Errorf("db ping: %w", pingErr)
	}

	sqlxDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlxDB.SetMaxIdleConns(cfg.MaxConnections)
	logger.DB.Debug("db pool configured",
		slog.String("event", "db.pool"),
		slog.Int("pool_open", cfg.MaxConnections),
	)

	// Final INFO line for connection
	logger.DB.Info("db connected",
		slog.String("event", "db.connect"),
		slog.String("driver", "postgres"),
		slog.String("host", cfg.Host),
		slog.String("port", cfg.Port),
		slog.String("db", cfg.Name),
		slog.Int("pool_open", cfg.MaxConnections),
		slog.Duration("duration", logger.RoundMS(took)),
	)

	return sqlxDB, nil
}

// WaitForPostgres tries to connect to the DB until it is ready or timeout is reached.
func WaitForPostgres(dsn string, timeout time.Duration) error {
	start := time.Now()
	var lastErr error
	for {
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			if err = db.Ping(); err == nil {
				_ = db.Close()
				return nil
			}
			_ = db.Close()
		}
		lastErr = err
		if time.Since(start) > timeout {
			return fmt.Errorf("timeout reached waiting for database: %w", lastErr)
		}
		time.Sleep(2 * time.Second)
	}
}
