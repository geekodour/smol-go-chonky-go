package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/geekodour/smol-go-chonky-go/internal/telemetry"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func cliCommand() *serverOptions {
	cliOptions := serverOptions{}
	kong.Parse(&cliOptions,
		kong.Name("smol"),
		kong.Description("smol server"),
		kong.ShortUsageOnError(),
	)
	return &cliOptions
}

func main() {
	// app context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srvOpts := cliCommand()

	// default logger
	logger := telemetry.NewSlogLogger(srvOpts.Logging.Level, srvOpts.Logging.Type)
	slog.SetDefault(logger)

	// primary prometheus registry
	promRegistry, err := telemetry.NewPrometheusRegistry()
	if err != nil {
		slog.Error("couldn't setup registry")
		return
	}

	srv, err := NewServer(srvOpts)
	if err == nil {
		slog.Error("couldn't setup server")
		return
	}
	defer srv.dbpool.Close()

	var g run.Group
	{
		g.Add(run.SignalHandler(ctx, os.Interrupt))
	}
	{

		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", srv.healthz)
		mux.Handle("/metrics", promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}))
		if srvOpts.Telemetry.Profiling {
			telemetry.AddPprof(mux)
		}
		srv := &http.Server{Handler: reqLogger(mux), Addr: ":" + srvOpts.Telemetry.Port}
		g.Add(func() error {
			fmt.Printf("Telemetry server listening on %s\n", srvOpts.Telemetry.Port)
			return srv.ListenAndServe()
		}, func(err error) {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			_ = srv.Shutdown(ctx)
		})
	}
	{
		server := &http.Server{
			Handler: commonHeaders(reqLogger(srv.mux)),
			Addr:    ":" + srvOpts.WebPort,
		}
		g.Add(func() error {
			fmt.Printf("Web server listening on %s\n", srvOpts.WebPort)
			return server.ListenAndServe()
		}, func(err error) {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			_ = server.Shutdown(ctx)
		})
	}

	slog.Error("exited", "reason", g.Run())
}
