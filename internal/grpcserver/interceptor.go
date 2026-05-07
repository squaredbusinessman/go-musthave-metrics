package grpcserver

import (
	"context"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const metadataXRealIP = "x-real-ip"

// TrustedSubnetInterceptor проверяет, что IP агента входит в доверенную подсеть.
func TrustedSubnetInterceptor(trustedSubnet string) grpc.UnaryServerInterceptor {
	if trustedSubnet == "" {
		return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
			return handler(ctx, req)
		}
	}

	_, subnet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		panic(fmt.Sprintf("invalid trusted subnet %q: %v", trustedSubnet, err))
	}

	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		values := md.Get(metadataXRealIP)
		if len(values) == 0 {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		ip := net.ParseIP(strings.TrimSpace(values[0]))
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, "forbidden")
		}

		return handler(ctx, req)
	}
}
