package main

import (
    "context"
    "log"
    "net"

    "google.golang.org/grpc"
    pb "path/to/your/generated/grpc/code"
)

type server struct {
    pb.UnimplementedRayDriverServer
}

func (s *server) ExecuteCommand(ctx context.Context, in *pb.CommandRequest) (*pb.CommandResponse, error) {
    log.Printf("Received: %v", in.GetCommand())
    // Implement your command execution logic here
    return &pb.CommandResponse{Output: "Command executed: " + in.GetCommand()}, nil
}

func main() {
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    s := grpc.NewServer()
    pb.RegisterRayDriverServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

