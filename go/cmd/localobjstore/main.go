package main

import (
    // "context"
    "log"
    "net"
    "strconv"
    "fmt"
    context "context"
    "errors"
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
    if address == "" {
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
    localObjectChannels[req.Uid] = make(chan []byte)
    if req.Testing == false {
        gcsObjClient.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: req.Uid, Requester: localNodeID})
    }
    
    val := <- localObjectChannels[req.Uid]
    localObjectStore[req.Uid] = val
    return &pb.GetResponse{Uid : req.Uid, ObjectBytes : localObjectStore[req.Uid]}, nil
}

func (s* server) LocationFound(ctx context.Context, resp *pb.LocationFoundResponse) (*pb.StatusResponse, error) {
    var otherLocalAddress string
  
    if resp.Port == 0 {
        nodeID := resp.Location; 
        otherLocalAddress = fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, nodeID, cfg.Ports.LocalObjectStore)     
    } else {
        otherLocalAddress = fmt.Sprintf("%s:%d", resp.Address, resp.Port)
    }
    log.Println("starting dial")
    conn, err := grpc.Dial(otherLocalAddress, grpc.WithInsecure())
    log.Println("finished dial")
    if err != nil {
        return &pb.StatusResponse{Success: false}, errors.New(fmt.Sprintf("failed to dial other LOS @:%s ", otherLocalAddress))
    }

    c := pb.NewLocalObjStoreClient(conn)
    log.Println("starting copy")
    x, err := c.Copy(ctx, &pb.CopyRequest{Uid : resp.Uid, Requester : localNodeID})
    log.Println("finished copy")
    if x == nil || err != nil {
        return &pb.StatusResponse{Success: false}, errors.New(fmt.Sprintf("failed to copy from other LOS @:%s ", otherLocalAddress))
    }
    // if resp.Port == "" {

    //     gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: resp.Uid, NodeId: localNodeID})
    // }
    
    localObjectChannels[resp.Uid] <- x.ObjectBytes
    log.Println("wrote to channel")
    return &pb.StatusResponse{Success: true}, nil

}

func (s* server) Copy(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
    data, ok:= localObjectStore[req.Uid];
    if !ok {
        return &pb.CopyResponse{Uid: req.Uid, ObjectBytes : nil}, errors.New("object was not in LOS")
    }
    return &pb.CopyResponse{Uid : req.Uid, ObjectBytes : data}, nil
}


