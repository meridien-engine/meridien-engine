package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/meridien-engine/meridien-engine/internal/repository"
)

// metadataKeyBusinessID is the gRPC metadata key that carries the active
// business (tenant) identifier. Clients must set this on every RPC call.
//
// gRPC metadata keys are canonically lower-case; the metadata package
// normalises them automatically so "Business-ID" and "business-id" are
// treated as the same key.
const metadataKeyBusinessID = "business-id"

// TenantInterceptor is a gRPC unary server interceptor that enforces
// per-request tenant isolation.
//
// It reads the "business-id" value from the incoming gRPC metadata, stores it
// in the context via repository.WithBusinessID, and then forwards the enriched
// context to the actual handler. Downstream service methods can retrieve the
// business ID with repository.BusinessIDFromContext.
//
// If the "business-id" metadata key is absent or empty the interceptor aborts
// the call immediately with codes.Unauthenticated so that no handler code is
// reached without a valid tenant scope.
//
// Wire this interceptor when constructing the gRPC server:
//
//	grpc.NewServer(
//	    grpc.UnaryInterceptor(middleware.TenantInterceptor),
//	)
//
// If you need to chain multiple interceptors, use grpc.ChainUnaryInterceptor.
func TenantInterceptor(
	ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (any, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(
			codes.Unauthenticated,
			"missing gRPC metadata: business-id is required",
		)
	}

	values := md.Get(metadataKeyBusinessID)
	if len(values) == 0 || values[0] == "" {
		return nil, status.Errorf(
			codes.Unauthenticated,
			"missing required metadata key %q", metadataKeyBusinessID,
		)
	}

	businessID := values[0]
	ctx = repository.WithBusinessID(ctx, businessID)

	return handler(ctx, req)
}
