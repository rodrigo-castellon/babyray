package main

import (

	"context"
	"log"
	"testing"
	"time"
	"fmt"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
//	"github.com/rodrigo-castellon/babyray/config"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	//cfg = config.GetConfig()  
}

// type mockFuncClient struct {
//     pb.GCSFuncClient // Embedding the interface for forward compatibility
//     resp *pb.FetchResponse
//     err  error
// }

// func (m *mockFuncClient) FetchFunc(ctx context.Context, in *pb.FetchRequest, opts ...grpc.CallOption) (*pb.FetchResponse, error) {
//     return m.resp, m.err
// }

type mockGCSClient struct {
	pb.GCSObjClient
	liveNodes map[uint64]bool
}
func (m *mockGCSClient) RegisterLiveNodes(ctx context.Context, req *pb.LiveNodesRequest, opts ...grpc.CallOption) (*pb.StatusResponse, error) {
	m.liveNodes = req.LiveNodes; 
	log.Printf(fmt.Sprint(m.liveNodes))
	return &pb.StatusResponse{Success: true}, nil
}


func (m *mockGCSClient) RequestLocation(ctx context.Context, req *pb.RequestLocationRequest, opts ...grpc.CallOption) (*pb.RequestLocationResponse, error){
	return nil, nil
}
func (m *mockGCSClient) RegisterGenerating(ctx context.Context, req *pb.GeneratingRequest,  opts ...grpc.CallOption) (*pb.StatusResponse, error){
	return nil, nil
}
func (m *mockGCSClient) GetObjectLocations(ctx context.Context, req *pb.ObjectLocationsRequest,  opts ...grpc.CallOption) (*pb.ObjectLocationsResponse, error) {
	return nil, nil
}
func TestHeartbeats(t *testing.T) {
	ctx :=context.Background()

	m := mockGCSClient {
		liveNodes: make(map[uint64]bool),
	}
	s := server {
		gcsClient: &m, 
		status: make(map[uint64]HeartbeatEntry),
	}

	s.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: 200})
	s.SendLiveNodes(ctx) 
	if val, ok := m.liveNodes[200]; val != true || !ok {
		t.Errorf("Node was not correctly registered as alive with GCS")
	}

	time.Sleep(10 * LIVE_NODE_TIMEOUT)

	s.SendLiveNodes(ctx); 

	if val, _ := m.liveNodes[200]; val == true {
		t.Errorf("Node was not correctly registered as dead with GCS")
	}
}
