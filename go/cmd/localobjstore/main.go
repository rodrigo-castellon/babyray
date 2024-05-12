package main

import (
    // "context"
    "log"
    "net"
    "strconv"
    "fmt"
    context "context"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"

)
var localObjectStore map[uint64][]byte
var localObjectChannels map[uint64]chan []byte
var gcsObjClient pb.GCSObjClient
var localNodeID uint64
var cfg *config.Config
func main() {
    cfg = config.GetConfig() // Load configuration
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

    localObjectStore = make(map[uint64][]byte)
    localObjectChannels = make(map[uint64]chan []byte)

    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
    conn, _ := grpc.Dial(gcsAddress, grpc.WithInsecure())
    gcsObjClient = pb.NewGCSObjClient(conn)
    localNodeID = 0
}

// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedLocalObjStoreServer
}

func (s *server) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StatusResponse, error) {
    localObjectStore[req.Uid] = req.ObjectBytes
    
    gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: req.Uid, NodeId: localNodeID})
    return &pb.StatusResponse{Success: true}, nil
}

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
    if val, ok := localObjectStore[req.Uid]; ok {
        return &pb.GetResponse{Uid : req.Uid, ObjectBytes : val}, nil
    }
    var nodeId uint64 = 1 
    localObjectChannels[req.Uid] = make(chan []byte)
    if req.Testing == False {
        gcsObjClient.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: req.Uid, NodeId: nodeId})
    }
    
    val := <- localObjectChannels[req.Uid]
    localObjectStore[req.Uid] = val
    return &pb.GetResponse{Uid : req.Uid, ObjectBytes : localObjectStore[req.Uid]}, nil
}

func (s* server) LocationFound(ctx context.Context, resp *pb.LocationFoundResponse) (*pb.StatusResponse, error) {
    var otherLocalAddress string
    if resp.Port != 0 {
        nodeID := resp.Location; 
        otherLocalAddress = fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, nodeID, cfg.Ports.LocalObjectStore)     
    } else {
        otherLocalAddress = fmt.Sprintf("%s:%d", resp.Address, resp.Port)
    }
   
    conn, _ := grpc.Dial(otherLocalAddress, grpc.WithInsecure())
    c := pb.NewLocalObjStoreClient(conn)
    x, _ := c.Copy(ctx, &pb.CopyRequest{Uid : resp.Uid, Requester : nodeID})
    
    gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: resp.Uid, NodeId: localNodeID})
    localObjectChannels[resp.Uid] <- x.ObjectBytes
    return &pb.StatusResponse{Success: true}, nil

}

func (s* server) Copy(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
    data, _ := localObjectStore[req.Uid];
    return &pb.CopyResponse{Uid : req.Uid, ObjectBytes : data}, nil
}


