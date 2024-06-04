/*
Some TODOs:
- GCS chain replication hasn't been done for GCS Object Table yet


*/

package main

import (
	"context"
	"log"
	"net"
	"strconv"

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"github.com/rodrigo-castellon/babyray/util"
	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	cfg := config.GetConfig()                                 // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.GCSFunctionTable) // Prepare the network address

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer(util.GetServerOptions()...)
	pb.RegisterGCSFuncServer(s, NewGCSFuncServer())
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

type GCSFuncServerProxy struct {
	pb.UnimplementedGCSFuncServer
}

func NewGCSFuncServer() *GCSFuncServerProxy {
	return &GCSFuncServerProxy{}
}

func (s *GCSFuncServerProxy) RegisterFunc(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	// Lock and unlock the mutex to handle concurrent writes safely
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate UID for this newly registered function
	uid, err := generateUID()
	if err != nil {
		log.Println("Failed to generate UID:", err)
		return nil, status.Errorf(codes.Internal, "Internal failed to generate UID")
	}

	// Logic to register the function in the server's function store
	s.functionStore[uid] = req.SerializedFunc

	return &pb.RegisterResponse{Name: uid}, nil
}

func (s *GCSFuncServerProxy) FetchFunc(ctx context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {
	s.mu.Lock()
	serializedFunc, ok := s.functionStore[req.Name]
	s.mu.Unlock()

	if !ok {
		return nil, status.Errorf(codes.NotFound, "function not found")
	}

	return &pb.FetchResponse{SerializedFunc: serializedFunc}, nil
}
