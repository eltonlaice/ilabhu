package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eltonlaice/ilabhu/control-plane/internal/api"
	"github.com/eltonlaice/ilabhu/control-plane/internal/catalog"
	"github.com/eltonlaice/ilabhu/control-plane/internal/provisioner"
	"github.com/eltonlaice/ilabhu/control-plane/internal/session"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	labsDir := flag.String("labs-dir", "../labs", "directory containing lab manifests")
	stateDir := flag.String("state-dir", defaultStateDir(), "directory for per-session terraform state")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cat, err := catalog.Load(*labsDir)
	if err != nil {
		logger.Error("load catalog", "err", err)
		os.Exit(1)
	}
	logger.Info("catalog loaded", "labs", len(cat.List()))

	if err := os.MkdirAll(*stateDir, 0o700); err != nil {
		logger.Error("create state dir", "err", err)
		os.Exit(1)
	}

	mgr := &session.Manager{
		Store:    session.NewStore(),
		Catalog:  cat,
		StateDir: *stateDir,
		TF:       &provisioner.Terraform{Stdout: os.Stdout, Stderr: os.Stderr},
		Logger:   logger,
	}

	srv := &api.Server{Catalog: cat, Manager: mgr, Logger: logger}
	httpSrv := &http.Server{
		Addr:              *addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("listening", "addr", *addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server", "err", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	logger.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}

func defaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".ilabhu"
	}
	return home + "/.ilabhu"
}
