package grpcserver

import (
	"context"
	"net"

	"github.com/OneDayX/go-metrics/internal/realip"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TrustedSubnet returns an interceptor that lets through only the calls whose
// x-real-ip metadata holds an address from subnet. The rest, including calls
// without it, fail with PermissionDenied.
func TrustedSubnet(subnet *net.IPNet) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var ip net.IP
		if values := metadata.ValueFromIncomingContext(ctx, realip.MetadataKey); len(values) > 0 {
			ip = net.ParseIP(values[0])
		}

		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "the agent is not in the trusted subnet")
		}

		return handler(ctx, req)
	}
}
