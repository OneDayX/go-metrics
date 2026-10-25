// Package grpcserver serves the Metrics gRPC service, so that the agents can
// send their batches over gRPC as well as over HTTP.
package grpcserver

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/OneDayX/go-metrics/internal/audit"
	"github.com/OneDayX/go-metrics/internal/models"
	pb "github.com/OneDayX/go-metrics/internal/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type batchUpdater interface {
	UpdateBatch(ctx context.Context, metrics []models.Metric) error
}

type auditor interface {
	Publish(event audit.Event)
}

// MetricsServer implements the Metrics service on top of the metric service.
type MetricsServer struct {
	pb.UnimplementedMetricsServer

	svc     batchUpdater
	auditor auditor
}

// NewMetricsServer returns a MetricsServer that saves metrics with svc. If
// auditor is nil, audit is off.
func NewMetricsServer(svc batchUpdater, auditor auditor) *MetricsServer {
	return &MetricsServer{svc: svc, auditor: auditor}
}

// New returns a gRPC server with the Metrics service. If subnet is not nil,
// the metrics are accepted only from agents in it, like on the HTTP routes.
func New(svc batchUpdater, auditor auditor, subnet *net.IPNet) *grpc.Server {
	var opts []grpc.ServerOption
	if subnet != nil {
		opts = append(opts, grpc.UnaryInterceptor(TrustedSubnet(subnet)))
	}

	srv := grpc.NewServer(opts...)
	pb.RegisterMetricsServer(srv, NewMetricsServer(svc, auditor))

	return srv
}

// UpdateMetrics saves the metrics of the request in one storage call. An
// empty request saves nothing. It fails with InvalidArgument if a metric has
// an unknown type and with Internal if the storage fails.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	metrics := make([]models.Metric, 0, len(req.GetMetrics()))
	for _, m := range req.GetMetrics() {
		metric, err := fromProto(m)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		metrics = append(metrics, metric)
	}

	if len(metrics) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	if err := s.svc.UpdateBatch(ctx, metrics); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	s.audit(ctx, metrics)

	return &pb.UpdateMetricsResponse{}, nil
}

// fromProto turns a protobuf metric into a models.Metric.
func fromProto(m *pb.Metric) (models.Metric, error) {
	switch m.GetType() {
	case pb.Metric_GAUGE:
		return models.Metric{ID: m.GetId(), MType: models.MetricTypeGauge, Value: models.Ptr(m.GetValue())}, nil
	case pb.Metric_COUNTER:
		return models.Metric{ID: m.GetId(), MType: models.MetricTypeCounter, Delta: models.Ptr(m.GetDelta())}, nil
	default:
		return models.Metric{}, fmt.Errorf("%w: %d", models.ErrUnknownMetricType, m.GetType())
	}
}

// audit sends an audit event about the saved metrics.
func (s *MetricsServer) audit(ctx context.Context, metrics []models.Metric) {
	if s.auditor == nil {
		return
	}

	names := make([]string, 0, len(metrics))
	for _, m := range metrics {
		names = append(names, m.ID)
	}

	// The peer address looks like "ip:port", we need only the ip.
	var ip string
	if p, ok := peer.FromContext(ctx); ok {
		ip = p.Addr.String()
		if host, _, err := net.SplitHostPort(ip); err == nil {
			ip = host
		}
	}

	s.auditor.Publish(audit.Event{
		TS:        time.Now().Unix(),
		Metrics:   names,
		IPAddress: ip,
	})
}
