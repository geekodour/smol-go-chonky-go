package main

// TODO:
// 1. benchmark and stresstest
// 2. pass logger to deps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/alecthomas/kong"
	"github.com/geekodour/smol-go-chonky-go/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/run"
)

var cli struct {
	DebugLog bool   `short:"d" help:"Enable debug logging"`
	Port     string `default:"8000" short:"p" help:"Set port number"`
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
	connConfig, err := pgxpool.ParseConfig(os.Getenv("DB_URL"))
	if err != nil {
		slog.Error("db config", slog.String("error", err.Error()))
		os.Exit(1)
	}
	// See https://pkg.go.dev/github.com/jackc/pgx/v5#hdr-PgBouncer
	connConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// TODO: Set logger and log level for pgx
	// NOTE: The same will go for other telemetry stuff
	// See https://dave.cheney.net/2017/01/23/the-package-level-logger-anti-pattern
	// See https://dave.cheney.net/2015/11/05/lets-talk-about-logging

	dbpool, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		slog.Error("db connection", slog.String("error", err.Error()))
		os.Exit(1)
	}
	return dbpool
}

func reqLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { slog.Info(r.Method, slog.String("path", r.URL.Path)) }()
		next.ServeHTTP(w, r)
	})
}

// NOTE: http.Error sets its own Content-Type headers
func commonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

type App struct {
	dbpool *pgxpool.Pool
	q      *db.Queries
}

// we're checking the connection with the db as-well in our health-check, as the
// sole purpose of our application is to get data from the db, it's fine to do so.
func (app App) healthz(w http.ResponseWriter, req *http.Request) {
	if err := app.dbpool.Ping(context.Background()); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func idFromPath(path, prefix string) (int32, error) {
	// TODO: compare with TrimPrefix at some point
	id, err := strconv.Atoi(path[len(prefix):])
	if err != nil {
		return 0, err
	}
	return int32(id), nil
}

func jsonDecode[T any](from io.Reader, to *T) error {
	dec := json.NewDecoder(from)
	dec.DisallowUnknownFields()
	err := dec.Decode(to)
	if err != nil {
		return err
	}
	return nil
}

// TODO: better error handling/messaging
// TODO: Check for correct content-type
// TODO: Limit max bytes read from body if needed
func (app App) handleCat(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		id, err := idFromPath(req.URL.Path, "/cat/")
		if err != nil {
			http.Error(w, "id", http.StatusBadRequest)
			return
		}

		cat, err := app.q.GetCat(req.Context(), id)
		if err != nil {
			http.Error(w, "missing", http.StatusNotFound)
			return
		}

		if err = json.NewEncoder(w).Encode(cat); err != nil {
			slog.Error("could not encode", "value", cat)
			return
		}
	case "POST":
		catParams := db.AddCatParams{}
		err := jsonDecode(req.Body, &catParams)
		if err != nil {
			http.Error(w, "body", http.StatusBadRequest)
			return
		}

		_, err = app.q.AddCat(req.Context(), catParams)
		if err != nil {
			http.Error(w, "no update", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case "PUT":
		id, err := idFromPath(req.URL.Path, "/cat/")
		if err != nil {
			http.Error(w, "id", http.StatusBadRequest)
			return
		}

		catParams := db.UpdateCatParams{}
		err = jsonDecode(req.Body, &catParams)
		if err != nil {
			http.Error(w, "body", http.StatusBadRequest)
			return
		}
		catParams.CatID = id

		// NOTE: This will not add new cats even if user provides a valid but
		// unused, only updates existing otherwise returns silently.
		err = app.q.UpdateCat(req.Context(), catParams)
		if err != nil {
			http.Error(w, "no update", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
	}
}

func (app App) listCats(w http.ResponseWriter, req *http.Request) {
	cats, err := app.q.ListCats(req.Context())
	if err != nil {
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	if err = json.NewEncoder(w).Encode(cats); err != nil {
		slog.Error("could not encode", "value", cats)
		return
	}
}

func NewApp() *App {
	dbpool := dbPool(context.TODO())
	q := db.New(dbpool)
	return &App{dbpool: dbpool, q: q}
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

	service := NewApp()
	defer service.dbpool.Close()

	var g run.Group
	{
		g.Add(run.SignalHandler(ctx, os.Interrupt))
	}
	{
		// Endpoints
		//  GET /healthz 			# healthcheck
		//  GET /cats 				# list cats
		//  GET /cat/:id 			# get cat
		//  PUT /cat/:id 			# update cat
		// POST /cat 				# post cat
		//
		// NOTE: Currently manually check HTTP methods/param extraction
		//       For more robust handling, check: https://github.com/go-chi/chi
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", service.healthz)
		mux.HandleFunc("/cats", service.listCats)
		mux.HandleFunc("/cat/", service.handleCat)

		srv := &http.Server{Handler: commonHeaders(reqLogger(mux)), Addr: ":" + cli.Port}
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
