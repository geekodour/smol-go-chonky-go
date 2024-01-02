package main

// TODO:
// 1. logging & error
// 2. lifecycle
// 3. http server
// 4. sqlc + pgx
// 5. add the api endpoints
// 6. benchmark and stresstest

import (
	"context"
	// "fmt"
	"log/slog"
	// "net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func setupLogger() {
	logLevel := &slog.LevelVar{} // INFO
	logOpts := slog.HandlerOptions{Level: logLevel, AddSource: true}

	handler := func() slog.Handler {
		if getEnv() == "production" {
			return slog.NewJSONHandler(os.Stderr, &logOpts)
		}
		return slog.NewTextHandler(os.Stderr, &logOpts)
	}()
	logger := slog.New(handler)

	slog.SetDefault(logger)
	// logLevel.Set(slog.LevelDebug) // if debug needed
}

func getEnv() string {
	env, ok := os.LookupEnv("PROJECT_ENV")
	if !ok {

		slog.Error("unset env var")
		os.Exit(1)
	}
	if env == "" {
		env = "development"
	}
	return env
}

func dbPool() *pgxpool.Pool {
	dbpool, err := pgxpool.New(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		slog.Error("Unable to connect to database", slog.String("error", err.Error()))
	}
	return dbpool
}

func main() {
	setupLogger()
	dbpool := dbPool()
	defer dbpool.Close()
}
