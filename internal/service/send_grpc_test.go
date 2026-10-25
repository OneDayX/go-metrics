package service

import (
	"context"
	"testing"

	"github.com/OneDayX/go-metrics/internal/models"
	pb "github.com/OneDayX/go-metrics/internal/proto"
	"github.com/OneDayX/go-metrics/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// testMetricsClient saves the last request and answers with err.
type testMetricsClient struct {
	req    *pb.UpdateMetricsRequest
	realIP []string
	err    error
}

func (c *testMetricsClient) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest, _ ...grpc.CallOption) (*pb.UpdateMetricsResponse, error) {
	c.req = in
	md, _ := metadata.FromOutgoingContext(ctx)
	c.realIP = md.Get("x-real-ip")

	return &pb.UpdateMetricsResponse{}, c.err
}

func TestMetricService_SendGRPC(t *testing.T) {
	svc := NewMetricService(repository.NewMemStorage())
	require.NoError(t, svc.Update(context.Background(), models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(3))}))

	client := &testMetricsClient{}

	err := svc.SendGRPC(context.Background(), client, "192.168.1.10", svc.Batch(context.Background()))
	require.NoError(t, err)

	require.Len(t, client.req.GetMetrics(), 1)
	assert.Equal(t, "PollCount", client.req.GetMetrics()[0].GetId())
	assert.Equal(t, pb.Metric_COUNTER, client.req.GetMetrics()[0].GetType())
	assert.Equal(t, int64(3), client.req.GetMetrics()[0].GetDelta())
	assert.Equal(t, []string{"192.168.1.10"}, client.realIP)
}

func TestMetricService_SendGRPC_ReturnsIncrementsOfAFailedBatch(t *testing.T) {
	svc := NewMetricService(repository.NewMemStorage())
	require.NoError(t, svc.Update(context.Background(), models.Metric{ID: "PollCount", MType: models.MetricTypeCounter, Delta: models.Ptr(int64(1))}))

	client := &testMetricsClient{err: status.Error(codes.PermissionDenied, "denied")}

	err := svc.SendGRPC(context.Background(), client, "", svc.Batch(context.Background()))
	require.Error(t, err)

	batch := svc.Batch(context.Background())
	require.Len(t, batch, 1)
	assert.Equal(t, int64(1), *batch[0].Delta, "the failed increment is counted again")
}
