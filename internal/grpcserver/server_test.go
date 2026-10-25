package grpcserver

import (
	"context"
	"testing"

	"github.com/OneDayX/go-metrics/internal/audit"
	pb "github.com/OneDayX/go-metrics/internal/proto"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/OneDayX/go-metrics/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testAuditor struct {
	events []audit.Event
}

func (a *testAuditor) Publish(event audit.Event) {
	a.events = append(a.events, event)
}

func TestUpdateMetrics(t *testing.T) {
	svc := service.NewMetricService(repository.NewMemStorage())
	auditor := &testAuditor{}
	srv := NewMetricsServer(svc, auditor)

	req := &pb.UpdateMetricsRequest{Metrics: []*pb.Metric{
		{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1.5},
		{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 5},
	}}

	_, err := srv.UpdateMetrics(context.Background(), req)
	require.NoError(t, err)

	gauge, err := svc.Fetch(context.Background(), "Alloc")
	require.NoError(t, err)
	assert.Equal(t, 1.5, *gauge.Value)

	counter, err := svc.Fetch(context.Background(), "PollCount")
	require.NoError(t, err)
	assert.Equal(t, int64(5), *counter.Delta)

	require.Len(t, auditor.events, 1)
	assert.Equal(t, []string{"Alloc", "PollCount"}, auditor.events[0].Metrics)
}

func TestUpdateMetricsUnknownType(t *testing.T) {
	srv := NewMetricsServer(service.NewMetricService(repository.NewMemStorage()), nil)

	req := &pb.UpdateMetricsRequest{Metrics: []*pb.Metric{
		{Id: "Alloc", Type: pb.Metric_MType(5), Value: 1.5},
	}}

	_, err := srv.UpdateMetrics(context.Background(), req)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
}
