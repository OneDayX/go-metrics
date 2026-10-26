// Package agent collects metrics and reports them to the server.
package agent

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/OneDayX/go-metrics/internal/models"
	pb "github.com/OneDayX/go-metrics/internal/proto"
	"github.com/OneDayX/go-metrics/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// shutdownTimeout is how long the last batches get to reach the server after
// the agent is told to stop.
const shutdownTimeout = 10 * time.Second

// Agent runs the polling and the reporting apart from each other: two
// goroutines fill the storage, Run builds batches out of it, and a pool of
// workers sends them over HTTP or gRPC. The pool size is the limit of requests
// in flight.
type Agent struct {
	cfg Config
	svc *service.MetricService
}

// New returns an Agent that collects metrics with svc and sends them as cfg
// says.
func New(cfg Config, svc *service.MetricService) *Agent {
	return &Agent{cfg: cfg, svc: svc}
}

// Run collects and sends metrics until ctx is cancelled. Then it sends what
// was collected since the last report and waits for the requests in flight,
// giving them shutdownTimeout. It returns an error only if it cannot start:
// the very first poll fails, the gRPC client cannot be created or the address
// of the host cannot be found.
func (a *Agent) Run(ctx context.Context) error {
	// Poll once right away, so the first report has something to send.
	if err := a.svc.Collect(ctx); err != nil {
		return err
	}

	// With a gRPC address the workers share one connection; it is closed
	// after they are done. A nil client means HTTP.
	serverAddr := a.cfg.ServerAddr
	var client pb.MetricsClient
	if a.cfg.GRPCAddr != "" {
		conn, err := grpc.NewClient(a.cfg.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()

		serverAddr = a.cfg.GRPCAddr
		client = pb.NewMetricsClient(conn)
	}

	// Every batch carries the agent's address, so that the server can check
	// its subnet.
	realIP, err := hostIP(serverAddr)
	if err != nil {
		return fmt.Errorf("find the host address: %w", err)
	}

	var pollers sync.WaitGroup

	collectors := []func(context.Context) error{a.svc.Collect, a.svc.CollectSystem}
	for _, collect := range collectors {
		pollers.Add(1)
		go func() {
			defer pollers.Done()
			a.poll(ctx, collect)
		}()
	}

	// The workers must outlive ctx to deliver the last batches, so they get
	// their own context, cancelled shutdownTimeout after ctx.
	sendCtx, cancelSend := context.WithCancel(context.WithoutCancel(ctx))
	defer cancelSend()

	stopTimer := context.AfterFunc(ctx, func() {
		time.AfterFunc(shutdownTimeout, cancelSend)
	})
	defer stopTimer()

	batches := make(chan []models.Metric)

	var senders sync.WaitGroup

	for range a.cfg.RateLimit {
		senders.Add(1)
		go func() {
			defer senders.Done()
			a.send(sendCtx, client, realIP, batches)
		}()
	}

	a.report(ctx, batches)

	// Shutdown: let the polls in progress finish, send what they collected,
	// then close the channel to tell the workers to stop.
	pollers.Wait()
	a.handOff(sendCtx, batches)
	close(batches)

	senders.Wait()
	return nil
}

// poll collects metrics into the storage on every tick.
func (a *Agent) poll(ctx context.Context, collect func(context.Context) error) {
	ticker := time.NewTicker(a.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := collect(ctx); err != nil {
				log.Printf("error collecting metrics: %v", err)
			}
		}
	}
}

// report hands a batch to the workers on every tick until ctx is cancelled.
func (a *Agent) report(ctx context.Context, batches chan<- []models.Metric) {
	ticker := time.NewTicker(a.cfg.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.handOff(ctx, batches)
		}
	}
}

// handOff takes everything collected so far and waits for a free worker to
// send it. It does not give up on ctx: the counters of the batch already count
// as reported, so dropping it would lose them.
func (a *Agent) handOff(ctx context.Context, batches chan<- []models.Metric) {
	if batch := a.svc.Batch(ctx); len(batch) > 0 {
		batches <- batch
	}
}

// send is one worker of the pool: it sends the batches it is given, one at a
// time, until Run closes the channel. They go over gRPC if client is not nil.
func (a *Agent) send(ctx context.Context, client pb.MetricsClient, realIP string, batches <-chan []models.Metric) {
	for batch := range batches {
		var err error
		if client != nil {
			err = a.svc.SendGRPC(ctx, client, realIP, batch)
		} else {
			err = a.svc.SendHTTP(ctx, a.cfg.ServerAddr, realIP, a.cfg.Key, a.cfg.PublicKey, batch)
		}

		if err != nil {
			log.Printf("error sending metrics: %v", err)
		}
	}
}

// hostIP returns the address of the interface the agent reaches the server
// through. Dialing UDP sends nothing, it only picks the route.
func hostIP(serverAddr string) (string, error) {
	conn, err := net.Dial("udp", serverAddr)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}
