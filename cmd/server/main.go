package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/OneDayX/go-metrics/internal/audit"
	"github.com/OneDayX/go-metrics/internal/database"
	"github.com/OneDayX/go-metrics/internal/handler"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/server"
	"github.com/OneDayX/go-metrics/internal/server/middleware"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// shutdownTimeout is how long the requests in progress get to finish.
const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// ctx is cancelled on SIGINT/SIGTERM so we can flush metrics before exiting.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := server.GetConfig()

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}
	defer logger.Sync()

	// Storage is picked in order: database, then file, then memory only.
	// An interface, not *database.DB: a nil pointer inside it would not be nil.
	var db handler.Pinger
	var persister *repository.Persister
	var svc *service.MetricService

	switch {
	case cfg.DatabaseDSN != "":
		pg, err := database.New(ctx, cfg.DatabaseDSN)
		if err != nil {
			return err
		}
		defer pg.Close()
		db = pg

		svc = service.NewMetricService(repository.NewDBStorage(pg.Pool()))
		logger.Info("storing metrics in the database")

	case cfg.FileStoragePath != "":
		storage := repository.NewMemStorage()
		persister = repository.NewPersister(storage, cfg.FileStoragePath, cfg.StoreInterval)

		if cfg.Restore {
			if err := persister.LoadMetrics(); err != nil {
				logger.Error("failed to load metrics", zap.Error(err))
			}
		}

		if cfg.StoreInterval == 0 {
			svc = service.NewMetricService(repository.NewPersistentMemStorage(storage, persister))
		} else {
			svc = service.NewMetricService(storage)
			persister.Start()
		}
		logger.Info("storing metrics in a file", zap.String("path", cfg.FileStoragePath))

	default:
		svc = service.NewMetricService(repository.NewMemStorage())
		logger.Info("storing metrics in memory only")
	}

	auditor := newAuditor(cfg, logger)
	h := handler.NewHandler(logger, auditor)

	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))
	// After gzip, so the signatures cover the plain body; after the logger, so
	// rejected requests are logged too.
	if cfg.Key != "" {
		r.Use(middleware.Hash(cfg.Key))
	}

	// Metrictest sending weird requests with trailing slashes, so we add StripSlashes middleware
	r.Use(chimw.StripSlashes)
	r.Post("/update/{type}/{name}/{value}", h.Update(svc)) // POST /update/gauge/Alloc/1
	r.Get("/", h.List(svc))                                // GET /
	r.Get("/value/{type}/{name}", h.Get(svc))              // GET /value/gauge/Alloc
	r.Get("/ping", h.Ping(db))                             // GET /ping

	//JSON Routes
	r.Post("/update", h.UpdateJSON(svc))   // POST /update and /update/
	r.Post("/updates", h.UpdatesJSON(svc)) // POST /updates and /updates/
	r.Post("/value", h.ValueJSON(svc))     // POST /value and /value/

	srv := &http.Server{Addr: cfg.ServerAddr, Handler: r}

	// Buffered, so the goroutine can report the error and exit even when
	// nobody is waiting for it anymore.
	serveErr := make(chan error, 1)

	logger.Info("starting server", zap.String("addr", cfg.ServerAddr))
	go func() {
		// ErrServerClosed only means Shutdown was called, which is not a failure.
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	// Stop on SIGINT/SIGTERM, or when the listener fails; the metrics are saved
	// either way.
	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serveErr:
		runErr = fmt.Errorf("server error: %w", err)
		logger.Error("server error", zap.Error(err))
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shut down the server", zap.Error(err))
	}

	// The server is stopped, so we can send the rest of the audit events.
	if err := auditor.Close(shutdownCtx); err != nil {
		logger.Error("failed to deliver the remaining audit events", zap.Error(err))
	}

	if persister == nil {
		return runErr
	}

	persister.Stop()
	return errors.Join(runErr, persister.SaveMetrics())
}

// newAuditor creates the audit publisher. If both audit file and audit URL
// are empty, it has no observers and audit is off.
func newAuditor(cfg server.Config, logger *zap.Logger) *audit.Publisher {
	publisher := audit.NewPublisher(logger)

	if cfg.AuditFile != "" {
		publisher.Subscribe(audit.NewFileObserver(cfg.AuditFile))
	}
	if cfg.AuditURL != "" {
		publisher.Subscribe(audit.NewHTTPObserver(cfg.AuditURL))
	}

	return publisher
}

// newLogger creates a zap logger. By default it writes JSON logs to stdout;
// if cfg.LogFile is set, logs are written to that file instead.
func newLogger(cfg server.Config) (*zap.Logger, error) {
	zcfg := zap.NewProductionConfig()
	if cfg.LogFile != "" {
		zcfg.OutputPaths = []string{cfg.LogFile}
	}
	return zcfg.Build()
}
