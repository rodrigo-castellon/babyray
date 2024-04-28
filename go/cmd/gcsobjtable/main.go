package main

import (
    "context"
    // "errors"
    "log"
    "net"
    "strconv"
    "sync"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)

func main() {
    cfg := config.LoadConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GCSObjectTable) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    pb.RegisterGCSObjServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

// server is used to implement your gRPC service.
type server struct {
    pb.UnimplementedGCSObjServer
    objectLocations map[uint32][]uint32
    mu              sync.Mutex
}

// Implement your service methods here.
func (s *server) NotifyOwns(ctx context.Context, req *pb.NotifyOwnsRequest) (*pb.StatusResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Append the nodeId to the list for the given uid
    s.objectLocations[req.Uid] = append(s.objectLocations[req.Uid], req.NodeId)
    //log.Printf("NotifyOwns: Added Node %d to UID %d", req.NodeId, req.Uid)

    return &pb.StatusResponse{Success: true}, nil
}

func (s *server) RequestLocation(ctx context.Context, req *pb.RequestLocationRequest) (*pb.StatusResponse, error) {
    s.mu.Lock()
    nodeIds, exists := s.objectLocations[req.Uid]
    s.mu.Unlock()

    if !exists || len(nodeIds) == 0 {
        log.Printf("RequestLocation: No locations found for UID %d", req.Uid)
        // Returning a StatusResponse indicating failure, instead of nil and a Go error
        return &pb.StatusResponse{
            Success:      false,
            ErrorCode:    404, // Or another appropriate error code
            ErrorMessage: "no locations found for given UID",
            Details:      "The requested UID does not exist in the object locations map.",
        }, nil // No error returned here; encoding the failure in the response message
    }

    // Assume successful case
    //log.Printf("RequestLocation: Returning location for UID %d: Node %d", req.Uid, nodeIds[0])
    return &pb.StatusResponse{
        Success: true,
        Details: strconv.Itoa(int(nodeIds[0])),
    }, nil
}
