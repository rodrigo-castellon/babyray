package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "sync"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"

    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
)

func main() {
    cfg := config.GetConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GCSFunctionTable) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    pb.RegisterGCSFuncServer(s, NewGCSFuncServer())
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

// server is used to implement your gRPC service.
type GCSFuncServer struct {
   pb.UnimplementedGCSFuncServer
   functionStore map[string][]byte
   mu            sync.Mutex // Use a mutex to manage concurrent access
}

func NewGCSFuncServer() *GCSFuncServer {
    return &GCSFuncServer{
        functionStore: make(map[string][]byte),
    }
}

// Implement your service methods here.
func (s *GCSFuncServer) RegisterFunc(ctx context.Context, req *pb.RegisterRequest) (*pb.StatusResponse, error) {
    // Lock and unlock the mutex to handle concurrent writes safely
    s.mu.Lock()
    defer s.mu.Unlock()

    // Logic to register the function in the server's function store
    log.Printf("Registering function: %s", req.Name)
    s.functionStore[req.Name] = req.SerializedFunc
    
    return &pb.StatusResponse{Success: true}, nil
}

func (s *GCSFuncServer) FetchFunc(ctx context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {
    s.mu.Lock()
    serializedFunc, ok := s.functionStore[req.Name]
    s.mu.Unlock()
    
    if !ok {
        return nil, status.Errorf(codes.NotFound, "function not found")
    }
    
    return &pb.FetchResponse{SerializedFunc: serializedFunc}, nil
}