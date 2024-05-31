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
	// "time"
	"sync"

	"github.com/rodrigo-castellon/babyray/config"
	"github.com/rodrigo-castellon/babyray/customlog"
	"github.com/rodrigo-castellon/babyray/util"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
)

const DEFAULT_AVG_BANDWIDTH = 0.1 // to start with

// LocalLog formats the message and logs it with a specific prefix
func LocalLog(format string, v ...interface{}) {
	var logMessage string
	if len(v) == 0 {
		logMessage = format // No arguments, use the format string as-is
	} else {
		logMessage = fmt.Sprintf(format, v...)
	}
	log.Printf("[lobs] %s", logMessage)
}

var cfg *config.Config
const EMA_PARAM float32 = .9
var mu sync.RWMutex

func main() {
	customlog.Init()
	cfg = config.GetConfig()                                  // Load configuration
    startServer(":" + strconv.Itoa(cfg.Ports.LocalObjectStore))
	// Create a channel and block on it to prevent the main function from exiting
	block := make(chan struct{})
	<-block
}

func startServer(port string) (*grpc.Server, error) {

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(util.GetServerOptions()...)
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, util.GetDialOptions()...)
	nodeId, _ := strconv.Atoi(os.Getenv("NODE_ID"))
	pb.RegisterLocalObjStoreServer(s, &server{localObjectStore: make(map[uint64][]byte), localObjectChannels: make(map[uint64]chan *pb.LocationFoundCallback), gcsObjClient: pb.NewGCSObjClient(conn), localNodeID: uint64(nodeId), avgBandwidth: DEFAULT_AVG_BANDWIDTH})


	LocalLog("lobs server listening at %v", lis.Addr())
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
	localObjectChannels map[uint64]chan *pb.LocationFoundCallback
	gcsObjClient        pb.GCSObjClient
	localNodeID         uint64
	avgBandwidth        float32
}

func (s *server) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StatusResponse, error) {
	mu.Lock()
	s.localObjectStore[req.Uid] = req.ObjectBytes
	mu.Unlock()

	s.gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: req.Uid, NodeId: s.localNodeID, ObjectSize: uint64(len(req.ObjectBytes))})
	return &pb.StatusResponse{Success: true}, nil
}

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	mu.RLock()
	if val, ok := s.localObjectStore[req.Uid]; ok {
		mu.RUnlock()
		return &pb.GetResponse{Uid: req.Uid, ObjectBytes: val, Local: true}, nil
	}
	mu.RUnlock()

	LocalLog("IN GET() RN!")

	s.localObjectChannels[req.Uid] = make(chan *pb.LocationFoundCallback)
	if req.Testing == false {
		s.gcsObjClient.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: req.Uid, Requester: s.localNodeID})
	}

	resp := <-s.localObjectChannels[req.Uid]

	LocalLog("GOT THE RESPONSE!")

	// handle the response accordingly
	if (!req.Copy) {
		return &pb.GetResponse{Uid: req.Uid}, nil
	}
	var otherLocalAddress string

	if resp.Port == 0 {
		nodeID := resp.Location
		otherLocalAddress = fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, nodeID, cfg.Ports.LocalObjectStore)
	} else {
		otherLocalAddress = fmt.Sprintf("%s:%d", resp.Address, resp.Port)
	}

	conn, err := grpc.Dial(otherLocalAddress, util.GetDialOptions()...)

	if err != nil {
		return &pb.GetResponse{Uid: req.Uid}, errors.New(fmt.Sprintf("failed to dial other LOS @:%s ", otherLocalAddress))
		// return &pb.StatusResponse{Success: false}, errors.New(fmt.Sprintf("failed to dial other LOS @:%s ", otherLocalAddress))
	}

	c := pb.NewLocalObjStoreClient(conn)

	// start := time.Now()

	LocalLog("CALLING COPY() ON THIS NODE...")
	x, err := c.Copy(ctx, &pb.CopyRequest{Uid: resp.Uid, Requester: s.localNodeID})

	LocalLog("the err was: %v", err)
	LocalLog("GETTING THE TOTAL BANDWIDTH FROM THIS...")
	// bandwidth := float32(len(x.ObjectBytes)) / float32((time.Now().Sub(start).Seconds()))

	// s.avgBandwidth = EMA_PARAM * s.avgBandwidth + (1 - EMA_PARAM) * bandwidth 

	if x == nil || err != nil {
		return &pb.GetResponse{Uid: req.Uid}, errors.New(fmt.Sprintf("failed to copy from other LOS @:%s ", otherLocalAddress))
	}

	if resp.Port == 0 {
			s.gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: resp.Uid, NodeId: s.localNodeID})
	}

	// val := <-s.localObjectChannels[req.Uid]
	mu.Lock()
	defer mu.Unlock()
	s.localObjectStore[req.Uid] = x.ObjectBytes
	return &pb.GetResponse{Uid: req.Uid, ObjectBytes: s.localObjectStore[req.Uid], Local: false}, nil
}

func (s *server) LocationFound(ctx context.Context, resp *pb.LocationFoundCallback) (*pb.StatusResponse, error) {

	channel, _ := s.localObjectChannels[resp.Uid]
	channel <- resp

	return &pb.StatusResponse{Success: true}, nil

}

func (s *server) Copy(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
	mu.RLock()
	data, ok := s.localObjectStore[req.Uid]
	mu.RUnlock()
	if !ok {
		return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: nil}, errors.New("object was not in LOS")
	}
	return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: data}, nil
}

func(s *server) AvgBandwidth(ctx context.Context, req *pb.StatusResponse) (*pb.BandwidthResponse, error) {
	return &pb.BandwidthResponse{AvgBandwidth: s.avgBandwidth}, nil
}