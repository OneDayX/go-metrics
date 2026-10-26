package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/OneDayX/go-metrics/internal/realip"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTrustedSubnet(t *testing.T) {
	_, subnet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	tests := []struct {
		name     string
		realIP   string
		wantCode codes.Code
	}{
		{name: "in the subnet", realIP: "192.168.1.10", wantCode: codes.OK},
		{name: "out of the subnet", realIP: "10.0.0.1", wantCode: codes.PermissionDenied},
		{name: "no metadata", realIP: "", wantCode: codes.PermissionDenied},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.realIP != "" {
				ctx = metadata.NewIncomingContext(ctx, metadata.Pairs(realip.MetadataKey, tt.realIP))
			}

			_, err := TrustedSubnet(subnet)(ctx, nil, nil, handler)

			assert.Equal(t, tt.wantCode, status.Code(err))
		})
	}
}
