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
    return &pb.GetResponse{uid = req.uid, objectBytes = localObjectStore[req.uid]}
}

func (s* server) LocationFound(ctx context.Context, resp *pb.LocationFoundResponse) (*pb.StatusResponse, error) {
    nodeID := resp.NodeId; 
    otherLocalAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.LocalScheduler)
    conn, _ := grpc.Dial(otherLocalAddress, grpc.WithInsecure())
    x := conn.Copy(ctx, &pb.CopyRequest{uid = resp.uid, requester = nodeID})
    c := NewGCSObjClient(); 
    c.NotifyOwns(ctx, &pb.NotifyOwnsRequest{req.uid})
    localObjectChannels[resp.uid] <- x.objectBytes
    return &pb.StatusResponse{Success: true}

}

func (s* server) Copy(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
    data, _ = localObjectStore[req.uid];
    return &pb.CopyResponse{uid = req.uid, objectBytes = data}
}


