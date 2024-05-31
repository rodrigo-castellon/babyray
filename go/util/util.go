package util

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/keepalive"
)

// GetDialOptions returns the gRPC dial options with max message size set.
func GetDialOptions() []grpc.DialOption {
	// Set the max size options
	return []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(1024 * 1024 * 1024), // 1GB
			grpc.MaxCallSendMsgSize(1024 * 1024 * 1024), // 1GB
		),
	}
}

// GetServerOptions returns the gRPC server options with max message size set.
func GetServerOptions() []grpc.ServerOption {
	const MAX_MSG_SIZE = 200 * 1024 * 1024 // 200MB

	return []grpc.ServerOption{
		grpc.MaxRecvMsgSize(MAX_MSG_SIZE),
		grpc.MaxSendMsgSize(MAX_MSG_SIZE),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
		}),
	}
}


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
