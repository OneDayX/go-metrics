package service

import (
	"context"

	"github.com/OneDayX/go-metrics/internal/models"
	pb "github.com/OneDayX/go-metrics/internal/proto"
	"github.com/OneDayX/go-metrics/internal/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// SendGRPC reports one batch to the server in a single UpdateMetrics call. A
// non-empty realIP goes in the x-real-ip metadata. Like Send, it repeats the
// call while the server cannot be reached and gives the increments back if
// the batch does not make it.
func (s *MetricService) SendGRPC(ctx context.Context, client pb.MetricsClient, realIP string, metrics []models.Metric) error {
	if len(metrics) == 0 {
		return nil
	}

	if realIP != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-real-ip", realIP)
	}

	req := &pb.UpdateMetricsRequest{Metrics: toProto(metrics)}

	err := retry.Do(func() error {
		_, err := client.UpdateMetrics(ctx, req)
		return err
	}, isUnavailable)
	if err != nil {
		s.unreport(metrics)
		return err
	}

	return nil
}

// isUnavailable reports whether the call failed to reach the server.
func isUnavailable(err error) bool {
	return status.Code(err) == codes.Unavailable
}

// toProto turns metrics into their protobuf form.
func toProto(metrics []models.Metric) []*pb.Metric {
	result := make([]*pb.Metric, 0, len(metrics))

	for _, metric := range metrics {
		m := &pb.Metric{Id: metric.ID}

		switch metric.MType {
		case models.MetricTypeCounter:
			m.Type = pb.Metric_COUNTER
			if metric.Delta != nil {
				m.Delta = *metric.Delta
			}
		case models.MetricTypeGauge:
			m.Type = pb.Metric_GAUGE
			if metric.Value != nil {
				m.Value = *metric.Value
			}
		}

		result = append(result, m)
	}

	return result
}
