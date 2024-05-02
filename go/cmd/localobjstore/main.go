package main

import (
    // "context"
    "log"
    "net"
    "strconv"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)
var localObjectStore map[uint32]bytes
var localObjectChannels map[uint32]chan uint32
func main() {
    cfg := config.LoadConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.LocalObjectStore) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    pb.RegisterLocalObjStoreServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
    localObjectStore = make(map[uint32]bytes)
    localObjectChannels = make(map[uint32]chan uint32)
}

// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedLocalObjStoreServer
}

func (s *server) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StatusResponse, error) {
    localObjectStore[req.uid] = req.objectBytes
    c := NewGCSObjClient(); 
    c.NotifyOwns(ctx, &pb.NotifyOwnsRequest{req.uid})

}

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
    if val, ok = localObjectStore[req.uid]; ok {
        return val
    }
    nodeId := 1
    localObjectChannels[req.uid] = make(chan uint32)
    c := NewGCSObjClient(); 
    c.RequestLocation(&pb.RequestLocationRequest{req.uid, nodeId})
    localObjectStore[req.uid] <- localObjectChannels[req.uid]
    return localObjectStore[req.uid]

}

func (s* server) LocationFound(ctx context.Context, resp *pb.LocationFoundResponse) (*pb.StatusResponse, error) {
    nodeID := resp.NodeId; 
    otherLocalAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.LocalScheduler)
    conn, err := grpc.Dial(otherLocalAddress, grpc.WithInsecure())
    x := conn.CopyRequest(ctx, &pb.CopyRequest{uid = resp.uid, requester = localNodeId})
    
    c := NewGCSObjClient(); 
    c.NotifyOwns(ctx, &pb.NotifyOwnsRequest{req.uid})
    localObjectChannels[resp.uid] <- x.objectBytes
    return &pb.StatusResponse{Success: true}
    

}

func (s* server) CopyRequest(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
    data, ok = localObjectStore[req.uid]; ok {
        return &pb.CopyResponse{uid = req.uid, objectBytes = data}
    }

}

// Implement your service methods here.

// func (s *server) NotifyOwns(ctx context.Context, req *pb.NotifyOwnsRequest) (*pb.StatusResponse, error) {
//     s.mu.Lock()
//     defer s.mu.Unlock()

//     // Append the nodeId to the list for the given uid
//     s.objectLocations[req.Uid] = append(s.objectLocations[req.Uid], req.NodeId)
//     //log.Printf("NotifyOwns: Added Node %d to UID %d", req.NodeId, req.Uid)

//     return &pb.StatusResponse{Success: true}, nil
// }

// func (s *server) RequestLocation(ctx context.Context, req *pb.RequestLocationRequest) (*pb.StatusResponse, error) {
//     s.mu.Lock()
//     nodeIds, exists := s.objectLocations[req.Uid]
//     s.mu.Unlock()

//     if !exists || len(nodeIds) == 0 {
//         log.Printf("RequestLocation: No locations found for UID %d", req.Uid)
//         // Returning a StatusResponse indicating failure, instead of nil and a Go error
//         return &pb.StatusResponse{
//             Success:      false,
//             ErrorCode:    404, // Or another appropriate error code
//             ErrorMessage: "no locations found for given UID",
//             Details:      "The requested UID does not exist in the object locations map.",
//         }, nil // No error returned here; encoding the failure in the response message
//     }

//     // Assume successful case
//     //log.Printf("RequestLocation: Returning location for UID %d: Node %d", req.Uid, nodeIds[0])
//     return &pb.StatusResponse{
//         Success: true,
//         Details: strconv.Itoa(int(nodeIds[0])),
//     }, nil
// }

