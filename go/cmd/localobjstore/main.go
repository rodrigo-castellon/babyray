package main

import (
	// "context"
	context "context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"os"

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
)

// var localObjectStore map[uint64][]byte
// var localObjectChannels map[uint64]chan []byte
// var gcsObjClient pb.GCSObjClient
// var localNodeID uint64
var cfg *config.Config

func main() {
	cfg = config.GetConfig()                                  // Load configuration
    startServer(":" + strconv.Itoa(cfg.Ports.LocalObjectStore))
	// address := ":" + strconv.Itoa(cfg.Ports.LocalObjectStore) // Prepare the network address
	// if address == "" {
	// 	lis, err := net.Listen("tcp", address)
	// 	if err != nil {
	// 		log.Fatalf("failed to listen: %v", err)
	// 	}
	// 	_ = lis
	// 	s := grpc.NewServer()
	// 	pb.RegisterLocalObjStoreServer(s, &server{})
	// 	log.Printf("server listening at %v", lis.Addr())
	// 	if err := s.Serve(lis); err != nil {
	// 		log.Fatalf("failed to serve: %v", err)
	// 	}
	// }

	// localObjectStore = make(map[uint64][]byte)
	// localObjectChannels = make(map[uint64]chan []byte)

	// gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	// conn, _ := grpc.Dial(gcsAddress, grpc.WithInsecure())
	// gcsObjClient = pb.NewGCSObjClient(conn)
	// localNodeID = 0
}

func startServer(port string) (*grpc.Server, error) {

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, grpc.WithInsecure())
	nodeId, _ := strconv.Atoi(os.Getenv("NODE_ID"))
	pb.RegisterLocalObjStoreServer(s, &server{localObjectStore: make(map[uint64][]byte), localObjectChannels: make(map[uint64]chan []byte), gcsObjClient: pb.NewGCSObjClient(conn), localNodeID: uint64(nodeId)})
	

	//log.Printf("server listening at %v", lis.Addr())
	go func() {
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	}()


	return s, nil
}

// server is used to implement your gRPC service.
type server struct {
	pb.UnimplementedLocalObjStoreServer
	localObjectStore    map[uint64][]byte
	localObjectChannels map[uint64]chan []byte
	gcsObjClient        pb.GCSObjClient
	localNodeID         uint64
}

func (s *server) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StatusResponse, error) {
	s.localObjectStore[req.Uid] = req.ObjectBytes

	s.gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: req.Uid, NodeId: s.localNodeID})
	return &pb.StatusResponse{Success: true}, nil
}

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if val, ok := s.localObjectStore[req.Uid]; ok {
		return &pb.GetResponse{Uid: req.Uid, ObjectBytes: val, Local: true}, nil
	}

	s.localObjectChannels[req.Uid] = make(chan []byte)
	if req.Testing == false {
		s.gcsObjClient.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: req.Uid, Requester: s.localNodeID})
	}

	val := <-s.localObjectChannels[req.Uid]
	s.localObjectStore[req.Uid] = val
	return &pb.GetResponse{Uid: req.Uid, ObjectBytes: s.localObjectStore[req.Uid], Local: false}, nil
}

func (s *server) LocationFound(ctx context.Context, resp *pb.LocationFoundCallback) (*pb.StatusResponse, error) {
	var otherLocalAddress string

	if resp.Port == 0 {
		nodeID := resp.Location
		otherLocalAddress = fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, nodeID, cfg.Ports.LocalObjectStore)
	} else {
		otherLocalAddress = fmt.Sprintf("%s:%d", resp.Address, resp.Port)
	}

	conn, err := grpc.Dial(otherLocalAddress, grpc.WithInsecure())

	if err != nil {
		return &pb.StatusResponse{Success: false}, errors.New(fmt.Sprintf("failed to dial other LOS @:%s ", otherLocalAddress))
	}

	c := pb.NewLocalObjStoreClient(conn)

	x, err := c.Copy(ctx, &pb.CopyRequest{Uid: resp.Uid, Requester: s.localNodeID})

	if x == nil || err != nil {
		return &pb.StatusResponse{Success: false}, errors.New(fmt.Sprintf("failed to copy from other LOS @:%s ", otherLocalAddress))
	}
	if resp.Port == 0 {

	     s.gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: resp.Uid, NodeId: s.localNodeID})
	}

	channel, ok := s.localObjectChannels[resp.Uid]
	if !ok {
		return &pb.StatusResponse{Success: false}, errors.New("channel DNE")
	}
	channel <- x.ObjectBytes

	return &pb.StatusResponse{Success: true}, nil

}

func (s *server) Copy(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
	data, ok := s.localObjectStore[req.Uid]
	if !ok {
		return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: nil}, errors.New("object was not in LOS")
	}
	return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: data}, nil
}
