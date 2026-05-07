package grpcserver

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestTrustedSubnetInterceptorEmptySubnetAllowsRequest(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("")
	called := false

	_, err := interceptor(context.Background(), nil, nil, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	if !called {
		t.Fatalf("handler was not called")
	}
}

func TestTrustedSubnetInterceptorPanicsOnInvalidCIDR(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()

	_ = TrustedSubnetInterceptor("not-a-cidr")
}

func TestTrustedSubnetInterceptorRejectsMissingRealIP(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		t.Fatalf("handler must not be called")
		return nil, nil
	})

	if got, want := status.Code(err), codes.PermissionDenied; got != want {
		t.Fatalf("status code = %v, want %v", got, want)
	}
}

func TestTrustedSubnetInterceptorAllowsTrustedIP(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(metadataXRealIP, "192.168.1.42"))
	called := false

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor() error = %v", err)
	}
	if !called {
		t.Fatalf("handler was not called")
	}
}

func TestTrustedSubnetInterceptorRejectsUntrustedIP(t *testing.T) {
	interceptor := TrustedSubnetInterceptor("192.168.1.0/24")
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(metadataXRealIP, "10.0.0.1"))

	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(context.Context, any) (any, error) {
		t.Fatalf("handler must not be called")
		return nil, nil
	})

	if got, want := status.Code(err), codes.PermissionDenied; got != want {
		t.Fatalf("status code = %v, want %v", got, want)
	}
}
