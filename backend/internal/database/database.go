package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/watch-tower-org/watchtower/backend/internal/config"
	"github.com/watch-tower-org/watchtower/backend/internal/logger"
)

type Database struct {
	DB *bun.DB
}

func New(cfg config.DatabaseConfig) (*Database, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))

	sqldb.SetMaxOpenConns(cfg.MaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.MaxIdleConns)
	sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	db := bun.NewDB(sqldb, pgdialect.New())

	err := db.Ping()
	if err != nil && !strings.Contains(err.Error(), "does not exist") {
		return nil, fmt.Errorf("failed to ping database %q: %w", cfg.Name, err)
	} else if err != nil && strings.Contains(err.Error(), "does not exist") {
		logger.Warn().Msg("Database doesn't exist, attempting to create it")
		adminDSN := fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=%s",
			cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.SSLMode)
		adminDB := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(adminDSN)))

		defer func(adminDB *sql.DB) {
			err := adminDB.Close()
			if err != nil {
				logger.Error().Err(err).Msg("Failed to close admin database connection")
			}
		}(adminDB)

		_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.Name))
		if err != nil {
			return nil, fmt.Errorf("failed to create database %q: %w", cfg.Name, err)
		}

		logger.Info().Msgf("Database %s created successfully", cfg.Name)

		sqldb.Close()
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		sqldb.SetMaxOpenConns(cfg.MaxOpenConns)
		sqldb.SetMaxIdleConns(cfg.MaxIdleConns)
		sqldb.SetConnMaxLifetime(cfg.ConnMaxLifetime)
		db = bun.NewDB(sqldb, pgdialect.New())
		err = db.Ping()
		if err != nil {
			return nil, fmt.Errorf("failed to ping newly created database %q: %w", cfg.Name, err)
		}
	}

	logger.Info().Msgf("Successfully connected to database %s", cfg.Name)

	ctx := context.Background()

	if err := AutoMigration(db, ctx); err != nil {
		logger.Error().Msgf("Failed to migrate database: %v", err)
		return nil, err
	}

	return &Database{DB: db}, nil
}

func (d *Database) Close() error {
	if err := d.DB.Close(); err != nil {
		logger.Error().Err(err).Msg("Failed to close database connection")
		return err
	}
	logger.Info().Msg("Database connection closed")
	return nil
}
