package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/geekodour/smol-go-chonky-go/internal/db"
	"github.com/geekodour/smol-go-chonky-go/internal/telemetry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type serverOptions struct {
	WebPort   string `default:"8000" help:"web server port"`
	Telemetry struct {
		Port      string `default:"8001" help:"telemetry server port"`
		Profiling bool   `help:"whether to expose pprof endpoints"`
	} `embed:"" prefix:"telemetry."`
	Logging struct {
		Level string `enum:"debug,info,warn,error" default:"info"`
		Type  string `enum:"json,console" default:"console"`
	} `embed:"" prefix:"logging."`
}
type Server struct {
	mux    http.Handler
	dbpool *pgxpool.Pool // pgx
	q      *db.Queries   // sqlc
	logger telemetry.Logger
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

func dbPool(ctx context.Context) (*pgxpool.Pool, error) {
	connConfig, err := pgxpool.ParseConfig(os.Getenv("DB_URL"))
	if err != nil {
		return nil, err
	}
	// See https://pkg.go.dev/github.com/jackc/pgx/v5#hdr-PgBouncer
	connConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	dbpool, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		return nil, err
	}
	return dbpool, nil
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

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// TODO: better error handling/messaging
// TODO: Check for correct content-type
// TODO: Limit max bytes read from body if needed
func (s *Server) handleCat(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "GET":
		id, err := idFromPath(req.URL.Path, "/cat/")
		if err != nil {
			http.Error(w, "id", http.StatusBadRequest)
			return
		}

		cat, err := s.q.GetCat(req.Context(), id)
		if err != nil {
			http.Error(w, "missing", http.StatusNotFound)
			return
		}

		if err = json.NewEncoder(w).Encode(cat); err != nil {
			s.logger.Error("encode", "value", cat)
			return
		}
	case "POST":
		catParams := db.AddCatParams{}
		err := jsonDecode(req.Body, &catParams)
		if err != nil {
			http.Error(w, "body", http.StatusBadRequest)
			return
		}

		_, err = s.q.AddCat(req.Context(), catParams)
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
		err = s.q.UpdateCat(req.Context(), catParams)
		if err != nil {
			http.Error(w, "no update", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
	}
}

func (s *Server) listCats(w http.ResponseWriter, req *http.Request) {
	cats, err := s.q.ListCats(req.Context())
	if err != nil {
		s.logger.Error("application", "err", err)
		http.Error(w, "internal", http.StatusInternalServerError)
		return
	}
	if err = json.NewEncoder(w).Encode(cats); err != nil {
		s.logger.Error("could not encode", "value", cats)
		return
	}
}

// we're checking the connection with the db as-well in our health-check, as the
// sole purpose of our application is to get data from the db, it's fine to do so.
func (s *Server) healthz(w http.ResponseWriter, req *http.Request) {
	if err := s.dbpool.Ping(context.Background()); err == nil {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// Web endpoints
//
//	GET /cats 				# list cats
//	GET /cat/:id 			# get cat
//	PUT /cat/:id 			# update cat
//
// POST /cat 				# post cat
func addRoutes(server *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/cats", server.listCats)
	mux.HandleFunc("/cat/", server.handleCat)
	return mux
}

func NewServer(config *serverOptions) (*Server, error) {
	dbpool, err := dbPool(context.TODO())
	if err != nil {
		return nil, err
	}
	logger := telemetry.NewSlogLogger(config.Logging.Level, config.Logging.Type)
	server := &Server{
		dbpool: dbpool,
		q:      db.New(dbpool),
		logger: &telemetry.SlogLogger{Logger: logger},
	}
	server.mux = addRoutes(server)
	return server, nil
}
