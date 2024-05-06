package main

import (
    // "context"
    "log"
    "net"
    "strconv"
    "bytes"
    "fmt"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"

)
var localObjectStore map[uint32][]byte
var localObjectChannels map[uint32]chan uint32
var gcsObjClient GCSObjClient
var localNodeID
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
    localObjectStore = make(map[uint32][]byte)
    localObjectChannels = make(map[uint32]chan uint32)

    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
    conn, _ := grpc.Dial(gcsAddress, grpc.WithInsecure())
    gcsObjClient = NewGCSObjClient(conn)
    localNodeID = 0
}

// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedLocalObjStoreServer
}

func (s *server) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StatusResponse, error) {
    localObjectStore[req.Uid] = req.ObjectBytes
    
    gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{req.Uid, localNodeID})

}

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
    if val, ok = localObjectStore[req.Uid]; ok {
        return val
    }
    nodeId := 1
    localObjectChannels[req.Uid] = make(chan uint32)
    gcsObjClient.RequestLocation(&pb.RequestLocationRequest{uid: req.Uid, nodeId: nodeId})
    localObjectStore[req.Uid] <- localObjectChannels[req.Uid]
    return &pb.GetResponse{uid : req.Uid, ObjectBytes : localObjectStore[req.Uid]}
}

func (s* server) LocationFound(ctx context.Context, resp *pb.LocationFoundResponse) (*pb.StatusResponse, error) {
    nodeID := resp.NodeId; 
    otherLocalAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, nodeID, cfg.Ports.LocalScheduler)
    conn, _ := grpc.Dial(otherLocalAddress, grpc.WithInsecure())
    x := conn.Copy(ctx, &pb.CopyRequest{uid : resp.uid, requester : nodeID})
    
    gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{req.Uid, localNodeID})
    localObjectChannels[resp.uid] <- x.ObjectBytes
    return &pb.StatusResponse{Success: true}

}

func (s* server) Copy(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
    data, _ = localObjectStore[req.Uid];
    return &pb.CopyResponse{uid : req.Uid, ObjectBytes : data}
}


