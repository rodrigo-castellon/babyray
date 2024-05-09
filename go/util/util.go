package util 

import (
    "context"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
)

func extractAddressFromCtx(ctx context.Context) (string, error) {
    if md, ok := metadata.FromIncomingContext(ctx); ok {
        addresses := md.Get("client-address")
        if len(addresses) > 0 {
            return addresses[0], nil
        }
        return "", status.Error(codes.InvalidArgument, "client address not provided in metadata")
    }
    return "", status.Error(codes.Internal, "failed to extract metadata")
}
