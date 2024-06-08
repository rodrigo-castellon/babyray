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
)

func main() {
	cfg := config.GetConfig() // Load configuration
	if cfg.GCS.EnableChainReplication {
		address := ":" + strconv.Itoa(cfg.Ports.GCSFunctionTable) // Prepare the network address

		lis, err := net.Listen("tcp", address)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		log.Printf("server listening at %v", lis.Addr())

		// Register with gRPC
		s := grpc.NewServer(util.GetServerOptions()...)
		pb.RegisterGCSFuncServer(s, NewGCSFuncServerProxy())

		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}
}

type GCSFuncServerProxy struct {
	pb.UnimplementedGCSFuncServer
}

func NewGCSFuncServerProxy() *GCSFuncServerProxy {
	return &GCSFuncServerProxy{}
}

func (s *GCSFuncServerProxy) RegisterFunc(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {

}

func (s *GCSFuncServerProxy) FetchFunc(ctx context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {

}
