package db

import (
	"context"

	"telegram_webapp/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(dsn string) *pgxpool.Pool {
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		logger.Fatal("failed to create database pool", "error", err)
	}

	if err := db.Ping(context.Background()); err != nil {
		logger.Fatal("failed to ping database", "error", err)
	}

	logger.Info("database connected")
	return db
}
