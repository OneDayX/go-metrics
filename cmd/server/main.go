package main

import (
	"net/http"

	"github.com/OneDayX/go-metrics/internal/handler"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/server"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	cfg := server.DefaultConfig()
	cfg.ParseFlags()

	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", handler.UpdateHandler(svc)) // POST /update/gauge/Alloc/1
	r.Get("/", handler.ListHandler(svc))                                // GET /
	r.Get("/value/{type}/{name}", handler.GetHandler(svc))              // GET /value/gauge/Alloc

	return http.ListenAndServe(cfg.ServerAddr, r)
}
