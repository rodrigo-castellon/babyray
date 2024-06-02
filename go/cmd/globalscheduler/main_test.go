package main

import (

	"context"

	"testing"
	"time"

	pb "github.com/rodrigo-castellon/babyray/pkg"

	"google.golang.org/grpc/test/bufconn"
	"github.com/rodrigo-castellon/babyray/config"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	cfg = config.GetConfig()  
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
	pb.GCSObjServer
	liveNodes map[uint64]bool
}
func (m *mockGCSClient) RegisterLiveNodes(ctx context.Context, req *pb.LiveNodesRequest) (*pb.StatusResponse, error) {
	m.liveNodes = req.LiveNodes; 
	return &pb.StatusResponse{Success: true}, nil
}
func TestHeartbeats(t *testing.T) {
	ctx :=context.Background()

	m := mockGCSClient {
		liveNodes: make(map[uint64]bool),
	}
	s := server {
		gcsClient: m, 
		status: make(map[uint64]HeartbeatEntry),
	}

	s.Heartbeat(ctx, &pb.HeartbeatRequest{NodeId: 200})
	s.SendLiveNodes(ctx) 
	if val, ok := m.liveNodes[200]; val != true || !ok {
		t.Errorf("Node was not correctly registered as alive with GCS")
	}

	time.Sleep(2 * LIVE_NODE_TIMEOUT)

	s.SendLiveNodes(ctx); 

	if val, ok := m.liveNodes[200]; val == true || ok {
		t.Errorf("Node was not correctly registered as dead with GCS")
	}
}
