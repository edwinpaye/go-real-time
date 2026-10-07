package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/sales-system/go-real-time/config"
	"github.com/sales-system/go-real-time/internal/infra/logger"
)

// DB encapsulates the connection pool and transaction runner.
type DB struct {
	*sql.DB
}

// NewDB initializes connection pool with robust timeouts and limits.
func NewDB(cfg *config.Config) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLife)
	db.SetConnMaxIdleTime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		logger.Warn("Database initial ping failed (may still connect once online)", logger.Fields{
			"error": err.Error(),
			"host":  cfg.DBHost,
			"db":    cfg.DBName,
		})
	} else {
		logger.Info("Database connection pool established successfully", logger.Fields{
			"host": cfg.DBHost,
			"db":   cfg.DBName,
		})
	}

	return &DB{DB: db}, nil
}

// WithTransaction executes a block of queries atomically inside an isolated transaction.
func (db *DB) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			logger.Error("Failed to rollback transaction", logger.Fields{"error": rbErr.Error()})
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
