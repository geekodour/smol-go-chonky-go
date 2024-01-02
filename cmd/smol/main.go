package main

// TODO:
// 3. http server
// 4. sqlc + pgx
// 5. add the api endpoints
// 6. benchmark and stresstest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/run"
)

var cli struct {
	DebugLog bool   `short:"d" help:"Enable debug logging"`
	Port     string `default:"6666" short:"p" help:"Set port number"`
}

func setupLogger(debugLog bool) {
	logLevel := &slog.LevelVar{}
	logOpts := slog.HandlerOptions{Level: logLevel}
	if debugLog {
		logLevel.Set(slog.LevelDebug)
		logOpts.AddSource = true
	}

	handler := func() slog.Handler {
		if getEnv() == "production" {
			return slog.NewJSONHandler(os.Stderr, &logOpts)
		}
		return slog.NewTextHandler(os.Stderr, &logOpts)
	}()
	logger := slog.New(handler)

	slog.SetDefault(logger)
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

func dbPool(ctx context.Context) *pgxpool.Pool {
	dbpool, err := pgxpool.New(ctx, os.Getenv("DB_URL"))
	if err != nil {
		slog.Error("db connection", slog.String("error", err.Error()))
		os.Exit(1)
	}
	return dbpool
}

func reqLogger(hdlr http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { slog.Info(r.Method, slog.String("path", r.URL.Path)) }()
		hdlr.ServeHTTP(w, r)
	})
}

type app struct {
	dbpool *pgxpool.Pool
}

func (a app) healthz(w http.ResponseWriter, req *http.Request) {
	if err := a.dbpool.Ping(context.Background()); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func main() {
	kong.Parse(&cli,
		kong.Name("smol"),
		kong.Description("smol server"),
		kong.ShortUsageOnError(),
	)

	setupLogger(cli.DebugLog)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	service := app{dbpool: dbPool(ctx)}
	defer service.dbpool.Close()

	var g run.Group
	{
		g.Add(run.SignalHandler(ctx, os.Interrupt))
	}
	{
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", service.healthz)

		srv := &http.Server{Handler: reqLogger(mux), Addr: ":" + cli.Port}
		g.Add(func() error {
			fmt.Printf("Server listening on %s\n", cli.Port)
			return srv.ListenAndServe()
		}, func(err error) {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
	}

	slog.Error("exited", "reason", g.Run())
}
