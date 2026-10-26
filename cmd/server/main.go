// Server receives metrics from agents over HTTP and gRPC and stores them in
// memory, in a file or in PostgreSQL. The flags, environment variables and
// config file keys are described in internal/server.Config.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof" // registers the /debug/pprof handlers in http.DefaultServeMux
	"os/signal"
	"syscall"
	"time"

	"github.com/OneDayX/go-metrics/internal/audit"
	"github.com/OneDayX/go-metrics/internal/database"
	"github.com/OneDayX/go-metrics/internal/grpcserver"
	"github.com/OneDayX/go-metrics/internal/handler"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/server"
	"github.com/OneDayX/go-metrics/internal/server/middleware"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// shutdownTimeout is how long the requests in progress get to finish.
const shutdownTimeout = 10 * time.Second

// Build info, set at build time with
// -ldflags "-X main.buildVersion=... -X main.buildDate=... -X main.buildCommit=...".
var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	printBuildInfo()

	// ErrHelp means -h printed the usage, which is not a failure.
	if err := run(); err != nil && !errors.Is(err, flag.ErrHelp) {
		log.Fatal(err)
	}
}

// printBuildInfo prints the build info to stdout.
func printBuildInfo() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)
}

func run() error {
	// ctx is cancelled on SIGINT/SIGTERM/SIGQUIT so we can flush metrics before exiting.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	cfg, err := server.GetConfig()
	if err != nil {
		return err
	}

	logger, err := newLogger(cfg)
	if err != nil {
		return err
	}
	// Sync often fails on stdout, and there is nowhere to report it anyway.
	defer func() { _ = logger.Sync() }()

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

	auditor, err := newAuditor(cfg, logger)
	if err != nil {
		return err
	}

	startPprof(cfg.PprofAddr, logger)

	// Buffered with a slot per server, so the goroutines can report the error
	// and exit even when nobody is waiting for it anymore.
	serveErr := make(chan error, 2)

	httpSrv := startHTTP(cfg.ServerAddr, newRouter(cfg, logger, auditor, svc, db), logger, serveErr)
	grpcSrv := startGRPC(cfg, svc, auditor, logger, serveErr)

	// Stop on a signal, or when the listener fails; the metrics are saved
	// either way.
	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serveErr:
		runErr = fmt.Errorf("server error: %w", err)
		logger.Error("server error", zap.Error(err))
	}

	shutdown(httpSrv, grpcSrv, auditor, logger)

	if persister == nil {
		return runErr
	}

	persister.Stop()
	return errors.Join(runErr, persister.SaveMetrics())
}

// newRouter builds the HTTP API: the middlewares and the routes of the
// handlers.
func newRouter(cfg server.Config, logger *zap.Logger, auditor *audit.Publisher, svc *service.MetricService, db handler.Pinger) http.Handler {
	h := handler.NewHandler(logger, auditor)

	r := chi.NewRouter()
	// Before gzip: the agent compresses the body first and then encrypts it.
	if cfg.PrivateKey != nil {
		r.Use(middleware.Decrypt(cfg.PrivateKey))
	}
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.Logger(logger))
	// After gzip, so the signatures cover the plain body; after the logger, so
	// rejected requests are logged too.
	if cfg.Key != "" {
		r.Use(middleware.Hash(cfg.Key))
	}

	// Metrictest sending weird requests with trailing slashes, so we add StripSlashes middleware
	r.Use(chimw.StripSlashes)
	r.Get("/", h.List(svc))                   // GET /
	r.Get("/value/{type}/{name}", h.Get(svc)) // GET /value/gauge/Alloc
	r.Get("/ping", h.Ping(db))                // GET /ping
	r.Post("/value", h.ValueJSON(svc))        // POST /value and /value/

	// The routes that take metrics accept them only from the trusted subnet.
	r.Group(func(r chi.Router) {
		if cfg.TrustedNet != nil {
			r.Use(middleware.TrustedSubnet(cfg.TrustedNet))
		}
		r.Post("/update/{type}/{name}/{value}", h.Update(svc)) // POST /update/gauge/Alloc/1
		r.Post("/update", h.UpdateJSON(svc))                   // POST /update and /update/
		r.Post("/updates", h.UpdatesJSON(svc))                 // POST /updates and /updates/
	})

	return r
}

// startPprof serves net/http/pprof on its own address, so the profiles are
// not part of the API. Profiling is off if addr is empty.
func startPprof(addr string, logger *zap.Logger) {
	if addr == "" {
		return
	}

	logger.Info("starting pprof server", zap.String("addr", addr))
	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			logger.Error("pprof server error", zap.Error(err))
		}
	}()
}

// startHTTP starts the HTTP server on addr; a failure to serve goes to
// serveErr.
func startHTTP(addr string, router http.Handler, logger *zap.Logger, serveErr chan<- error) *http.Server {
	srv := &http.Server{Addr: addr, Handler: router}

	logger.Info("starting server", zap.String("addr", addr))
	go func() {
		// ErrServerClosed only means Shutdown was called, which is not a failure.
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	return srv
}

// startGRPC starts the gRPC server next to HTTP, with the same storage and
// audit; a failure to serve goes to serveErr. It returns nil if gRPC is off.
func startGRPC(cfg server.Config, svc *service.MetricService, auditor *audit.Publisher, logger *zap.Logger, serveErr chan<- error) *grpc.Server {
	if cfg.GRPCAddr == "" {
		return nil
	}

	srv := grpcserver.New(svc, auditor, cfg.TrustedNet)

	logger.Info("starting gRPC server", zap.String("addr", cfg.GRPCAddr))
	go func() {
		if err := serveGRPC(srv, cfg.GRPCAddr); err != nil {
			serveErr <- fmt.Errorf("gRPC: %w", err)
		}
	}()

	return srv
}

// serveGRPC listens on addr and serves gRPC calls until the server is stopped.
// Serve returns nil after a stop, so only a real failure is an error.
func serveGRPC(srv *grpc.Server, addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	return srv.Serve(listener)
}

// shutdown stops the servers, giving the requests in progress shutdownTimeout
// to finish, and then delivers the rest of the audit events. grpcSrv is nil if
// gRPC is off.
func shutdown(httpSrv *http.Server, grpcSrv *grpc.Server, auditor *audit.Publisher, logger *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Error("failed to shut down the server", zap.Error(err))
	}

	if grpcSrv != nil {
		stopGRPC(ctx, grpcSrv)
	}

	// The servers are stopped, so we can send the rest of the audit events and
	// close the audit file.
	if err := auditor.Close(ctx); err != nil {
		logger.Error("failed to deliver the remaining audit events", zap.Error(err))
	}
}

// stopGRPC lets the calls in progress finish, but cuts them off when ctx is
// done: GracefulStop alone would wait for them forever.
func stopGRPC(ctx context.Context, srv *grpc.Server) {
	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-ctx.Done():
		srv.Stop()
	}
}

// newAuditor creates the audit publisher. If both audit file and audit URL
// are empty, it has no observers and audit is off.
func newAuditor(cfg server.Config, logger *zap.Logger) (*audit.Publisher, error) {
	publisher := audit.NewPublisher(logger)

	if cfg.AuditFile != "" {
		file, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			return nil, fmt.Errorf("open audit file: %w", err)
		}
		publisher.Subscribe(file)
	}
	if cfg.AuditURL != "" {
		publisher.Subscribe(audit.NewHTTPObserver(cfg.AuditURL))
	}

	return publisher, nil
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
