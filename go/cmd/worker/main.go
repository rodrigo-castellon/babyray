package main

import (
    "context"
    "log"
    "net"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
)

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterYourServiceServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

// server is used to implement your gRPC service.
type server struct {
    pb.UnimplementedYourServiceServer
}

// Implement your service methods here.

