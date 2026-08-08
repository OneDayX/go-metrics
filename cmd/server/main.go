package main

import (
	"log"
	"net/http"

	"github.com/OneDayX/go-metrics/internal/handler"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/server"
	"github.com/OneDayX/go-metrics/internal/server/middleware"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg := server.GetConfig()

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}
	defer logger.Sync()

	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	h := handler.NewHandler(logger)

	r := chi.NewRouter()
	r.Use(middleware.Logger(logger))
	r.Post("/update/{type}/{name}/{value}", h.Update(svc)) // POST /update/gauge/Alloc/1
	r.Get("/", h.List(svc))                                // GET /
	r.Get("/value/{type}/{name}", h.Get(svc))              // GET /value/gauge/Alloc

	//JSON Routes
	r.Post("/update", h.UpdateJSON(svc)) // POST /update
	r.Post("/value", h.ValueJSON(svc))   // POST /value

	logger.Info("starting server", zap.String("addr", cfg.ServerAddr))
	logger.Debug("starting server")

	return http.ListenAndServe(cfg.ServerAddr, r)
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
