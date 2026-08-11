package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/OneDayX/go-metrics/internal/handler"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/server"
	"github.com/OneDayX/go-metrics/internal/server/middleware"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

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

	storage := repository.NewMemStorage()
	persister := repository.NewPersister(storage, cfg.FileStoragePath, cfg.StoreInterval)

	if cfg.Restore {
		if err := persister.LoadMetrics(); err != nil {
			logger.Error("failed to load metrics", zap.Error(err))
		}
	}

	var svc *service.MetricService
	if cfg.StoreInterval == 0 {
		svc = service.NewMetricService(repository.NewPersistentMemStorage(storage, persister))
	} else {
		svc = service.NewMetricService(storage)
		persister.Start()
	}

	h := handler.NewHandler(logger)

	r := chi.NewRouter()
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))

	// Metrictest sending weird requests with trailing slashes, so we add StripSlashes middleware
	r.Use(chimw.StripSlashes)
	r.Post("/update/{type}/{name}/{value}", h.Update(svc)) // POST /update/gauge/Alloc/1
	r.Get("/", h.List(svc))                                // GET /
	r.Get("/value/{type}/{name}", h.Get(svc))              // GET /value/gauge/Alloc

	//JSON Routes
	r.Post("/update", h.UpdateJSON(svc)) // POST /update and /update/
	r.Post("/value", h.ValueJSON(svc))   // POST /value and /value/

	logger.Info("starting server", zap.String("addr", cfg.ServerAddr))
	go func() {
		if err := http.ListenAndServe(cfg.ServerAddr, r); err != nil {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	// Wait for SIGINT/SIGTERM, then flush metrics to disk before exiting.
	<-ctx.Done()
	logger.Info("shutdown signal received, saving metrics")

	persister.Stop()
	return persister.SaveMetrics()
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
