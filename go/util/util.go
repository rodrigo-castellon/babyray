package util

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// DEPRECATED -- DO NOT USE
func ExtractAddressFromCtx(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// If no metadata is present at all, return an InvalidArgument error
		return "", status.Error(codes.InvalidArgument, "no metadata available in context")
	}

	addresses := md.Get("client-address")
	if len(addresses) == 0 {
		// Metadata is there but does not have the expected 'client-address' key
		return "", status.Error(codes.InvalidArgument, "client address not provided in metadata")
	}

	return addresses[0], nil
}
