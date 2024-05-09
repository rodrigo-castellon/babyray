package main

import (
    "context"
    // "errors"
    "log"
    "net"
    "strconv"
    "sync"
    "math/rand"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"

    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
)

func main() {
    cfg := config.GetConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GCSObjectTable) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    pb.RegisterGCSObjServer(s, NewGCSObjServer()) 
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

// server is used to implement your gRPC service.
type GCSObjServer struct {
    pb.UnimplementedGCSObjServer
    objectLocations map[uint64][]uint64
    mu              sync.Mutex
    cond            *sync.Cond
}

func NewGCSObjServer() *GCSObjServer {
    server := &GCSObjServer{
        objectLocations: make(map[uint64][]uint64),
        mu:              sync.Mutex{}, // Mutex initialized here.
    }
    server.cond = sync.NewCond(&server.mu) // Properly pass the address of the struct's mutex.
    return server
}

// Implement your service methods here.
func (s *GCSObjServer) NotifyOwns(ctx context.Context, req *pb.NotifyOwnsRequest) (*pb.StatusResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Append the nodeId to the list for the given uid
    s.objectLocations[req.Uid] = append(s.objectLocations[req.Uid], req.NodeId)
    s.cond.Broadcast()

    return &pb.StatusResponse{Success: true}, nil
}

func (s *GCSObjServer) RequestLocation(ctx context.Context, req *pb.RequestLocationRequest) (*pb.RequestLocationResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    nodeIds, exists := s.objectLocations[req.Uid]
    
    for !exists || len(nodeIds) == 0 {
        s.cond.Wait()
        nodeIds, exists = s.objectLocations[req.Uid]
        if ctx.Err() != nil {
            return nil, status.Error(codes.Canceled, "request cancelled or context deadline exceeded")
        }
    }

    // Assume successful case
    randomIndex := rand.Intn(len(nodeIds))
    return &pb.RequestLocationResponse{
        NodeId : nodeIds[randomIndex], 
    }, nil
}

// NOTE: We only use one cv for now, which may cause performance issues in the future
// However, there is significant overhead if we want one cv per object, as then we would have to manage
// the cleanup through reference counting 