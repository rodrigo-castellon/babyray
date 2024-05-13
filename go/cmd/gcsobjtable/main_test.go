package main

import (
	"bytes"
	"context"
	"log"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"context"
	"net"

	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterGCSObjServer(s, NewGCSObjServer())
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestGetNodeId(t *testing.T) {
	// Seed the random number generator for reproducibility in tests
	rand.Seed(1)

	// Initialize the GCSObjServer
	server := &GCSObjServer{
		objectLocations: make(map[uint64][]uint64),
		waitlist:        make(map[uint64][]string),
		mu:              sync.Mutex{},
	}

	// Test case: UID exists with multiple NodeIds
	server.objectLocations[1] = []uint64{100, 101, 102}
	nodeId, exists := server.getNodeId(1)
	if !exists {
		t.Errorf("Expected UID 1 to exist")
	}
	if nodeId == nil || (*nodeId != 100 && *nodeId != 101 && *nodeId != 102) {
		t.Errorf("Expected nodeId to be one of [100, 101, 102], got %v", nodeId)
	}

	// Test case: UID exists with a single NodeId
	server.objectLocations[2] = []uint64{200}
	nodeId, exists = server.getNodeId(2)
	if !exists {
		t.Errorf("Expected UID 2 to exist")
	}
	if nodeId == nil || *nodeId != 200 {
		t.Errorf("Expected nodeId to be 200, got %v", nodeId)
	}

	// Test case: UID does not exist
	nodeId, exists = server.getNodeId(3)
	if exists {
		t.Errorf("Expected UID 3 to not exist")
	}
	if nodeId != nil {
		t.Errorf("Expected nodeId to be nil, got %v", nodeId)
	}

	// Test case: UID exists but with an empty NodeId list
	server.objectLocations[4] = []uint64{}
	nodeId, exists = server.getNodeId(4)
	if exists {
		t.Errorf("Expected UID 4 to not exist due to empty NodeId list")
	}
	if nodeId != nil {
		t.Errorf("Expected nodeId to be nil, got %v", nodeId)
	}
}

// Test the getNodeId function
func TestGetNodeId2(t *testing.T) {
	// Initialize the server
	server := NewGCSObjServer()

	// Seed the random number generator to produce consistent results
	rand.Seed(time.Now().UnixNano())

	// Define test cases
	testCases := []struct {
		uid          uint64
		nodeIds      []uint64
		expectNil    bool
		expectExists bool
	}{
		{uid: 1, nodeIds: []uint64{101, 102, 103}, expectNil: false, expectExists: true},
		{uid: 2, nodeIds: []uint64{}, expectNil: true, expectExists: false},
		{uid: 3, nodeIds: nil, expectNil: true, expectExists: false},
	}

	for _, tc := range testCases {
		// Populate the objectLocations map
		if tc.nodeIds != nil {
			server.objectLocations[tc.uid] = tc.nodeIds
		}

		// Call getNodeId
		nodeId, exists := server.getNodeId(tc.uid)

		// Check if the result is nil or not as expected
		if tc.expectNil && nodeId != nil {
			t.Errorf("Expected nil, but got %v for uid %d", *nodeId, tc.uid)
		}

		if !tc.expectNil && nodeId == nil {
			t.Errorf("Expected non-nil, but got nil for uid %d", tc.uid)
		}

		// Check if the existence flag is as expected
		if exists != tc.expectExists {
			t.Errorf("Expected exists to be %v, but got %v for uid %d", tc.expectExists, exists, tc.uid)
		}

		// If nodeId is not nil, ensure it is one of the expected nodeIds
		if nodeId != nil && !contains(tc.nodeIds, *nodeId) {
			t.Errorf("NodeId %d is not in expected nodeIds %v for uid %d", *nodeId, tc.nodeIds, tc.uid)
		}
	}
}

func TestSendCallback(t *testing.T) {
	// Setup a mock listener
	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterLocalObjStoreServer(s, &mockLocalObjStoreServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// Mock client address
	clientAddress := "bufnet"

	// Initialize the GCSObjServer
	server := &GCSObjServer{}

	// Use bufDialer in grpc.Dial call
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Run sendCallback as a goroutine
	go server.sendCallback(clientAddress, 1, 100)

	// Allow some time for the goroutine to execute
	time.Sleep(1 * time.Second)
}

// Mock implementation of the LocalObjStoreServer
type mockLocalObjStoreServer struct {
	pb.UnimplementedLocalObjStoreServer
}

func (m *mockLocalObjStoreServer) LocationFound(ctx context.Context, req *pb.LocationFoundCallback) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{Success: true}, nil
}

func TestNotifyOwns(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pb.NewGCSObjClient(conn)

	// Testing NotifyOwns
	resp, err := client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{
		Uid:    1,
		NodeId: 100,
	})
	if err != nil || !resp.Success {
		t.Errorf("NotifyOwns failed: %v, response: %v", err, resp)
	}
}

func TestRequestLocation(t *testing.T) {
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pb.NewGCSObjClient(conn)

	// Ensure the object is registered to test retrieval
	_, err = client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{
		Uid:    1,
		NodeId: 100,
	})
	if err != nil {
		t.Fatalf("Setup failure: could not register UID: %v", err)
	}

	// Test RequestLocation when the location should be found immediately
	resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 1})
	if err != nil {
		t.Errorf("RequestLocation failed: %v", err)
		return
	}
	if !resp.ImmediatelyFound {
		t.Errorf("Expected to find location immediately, but it was not found")
	}

	// Test RequestLocation for a UID that does not exist
	resp, err = client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 2})
	if err != nil {
		t.Errorf("RequestLocation failed: %v", err)
		return
	}
	if resp.ImmediatelyFound {
		t.Errorf("Expected location not to be found immediately, but it was found")
	}
}

//

// - Two goroutines will call RequestLocation for a UID that initially doesn't have any node IDs associated with it. They will block until they are notified of a change.
// - One goroutine will perform the NotifyOwns action after a short delay, adding a node ID to the UID, which should then notify the waiting goroutines.

// Node A: a LocalObjStore
// Node B: a LocalObjStore
// Node C: a GCSObjTable
//

type MockLocalObjStore struct {
	pb.UnimplementedLocalObjStoreServer
}

func (s *MockLocalObjStore) LocationFound(ctx context.Context, resp *pb.LocationFoundCallback) (*pb.StatusResponse, error) {
	return &pb.StatusResponse{Success: true}, nil
}

// HELPER UnaryInterceptor modifies the sender's IP address in the metadata
func UnaryInterceptor(mockIP string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Get the current metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}

		// Replace or add the mock IP address
		md.Set("x-forwarded-for", mockIP)

		// Create a new context with the updated metadata
		newCtx := metadata.NewIncomingContext(ctx, md)

		// Call the handler with the new context
		return handler(newCtx, req)
	}
}

func startLocalObjStoreServer(port string) (*grpc.Server, error) {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer(
		grpc.UnaryInterceptor(UnaryInterceptor("192.168.1.100")), // Mock IP address
	)
	pb.RegisterLocalObjStoreServer(s, &MockLocalObjStore{})
	go func() {
		s.Serve(lis)
	}()
	return s, nil
}

func TestStoreAndGet_External(t *testing.T) {
	// Set up GCSObjectClient
	ctx := context.Background()
	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()
	client := pb.NewGCSObjClient(conn)

	// Node A listening
	s1, err := startLocalObjStoreServer(":50051")
	if err != nil {
		t.Fatalf("Failed to start server 1: %v", err)
	}
	defer s1.Stop()

	// Node B listening
	s2, err := startLocalObjStoreServer(":50052")
	if err != nil {
		t.Fatalf("Failed to start server 2: %v", err)
	}
	defer s2.Stop()

	// Simulate Node A attempting to ask GCS for object uid 1
	resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 1})
	if err != nil {
		t.Errorf("RequestLocation failed: %v", err)
		return
	}
	if resp.ImmediatelyFound {
		t.Errorf("Expected location not to be found immediately, but it was found")
	}

	// Simulate Node B attempting to ask GCS for object uid 1
	resp, err = client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 1})
	if err != nil {
		t.Errorf("RequestLocation failed: %v", err)
		return
	}
	if resp.ImmediatelyFound {
		t.Errorf("Expected location not to be found immediately, but it was found")
	}

	data := []byte{72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100}

	sresp, err := client2.Store(ctx, &pb.StoreRequest{Uid: 3, ObjectBytes: data})
	if !sresp.Success || err != nil {
		t.Errorf("Failed to store value on LOS 2")
	}
	response2, err := client2.Get(ctx, &pb.GetRequest{Uid: 3, Testing: true})

	if err != nil || !bytes.Equal(response2.ObjectBytes, data) {
		t.Errorf("Failed to get value on LOS 2")
	}

	go func() {
		timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		response1, err := client1.Get(timeout, &pb.GetRequest{Uid: 3, Testing: true})
		if response1.Local {
			t.Errorf("found on local")
		}
		if err != nil || !bytes.Equal(response1.ObjectBytes, data) {
			if timeout.Err() == context.DeadlineExceeded {
				t.Errorf("Timeout on client1 get call")
			}
			t.Errorf("Failed to get value on LOS 1")
		}

	}()

	time.Sleep(1 * time.Second)
	timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	locStatusResp, err := client1.LocationFound(timeout, &pb.LocationFoundCallback{Uid: 3, Address: "localhost", Port: 50052})
	if err != nil || locStatusResp == nil || !locStatusResp.Success {
		if timeout.Err() == context.DeadlineExceeded {
			t.Errorf("Timeout on client1 location call")
		}
		t.Errorf("Failed to tell LOS 1 about location: %v", err)
	}

}

// Create a unit test in Go where three goroutines are involved, with the first two waiting for an object's location
// and the third notifying the server of the object's presence:
// - Two goroutines will call RequestLocation for a UID that initially doesn't have any node IDs associated with it. They will block until they are notified of a change.
// - One goroutine will perform the NotifyOwns action after a short delay, adding a node ID to the UID, which should then notify the waiting goroutines.
// func TestRequestLocationWithNotification(t *testing.T) {
// 	// Initialize the server and listener
// 	lis := bufconn.Listen(bufSize)
// 	s := grpc.NewServer()
// 	pb.RegisterGCSObjServer(s, MockNewGCSObjServer())
// 	go func() {
// 		if err := s.Serve(lis); err != nil {
// 			log.Fatalf("Server exited with error: %v", err)
// 		}
// 	}()

// 	// Setup gRPC client
// 	ctx := context.Background()
// 	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
// 	if err != nil {
// 		t.Fatalf("Failed to dial bufnet: %v", err)
// 	}
// 	defer conn.Close()
// 	client := pb.NewGCSObjClient(conn)

// 	// Use a wait group to wait for both goroutines to complete
// 	var wg sync.WaitGroup
// 	wg.Add(2)

// 	// A channel to listen for callback notifications
// 	callbackReceived := make(chan struct{}, 2)

// 	// Mock the sendCallback to capture its invocation
// 	originalSendCallback := (&GCSObjServer{}).sendCallback
// 	mockSendCallback := func(clientAddress string, uid uint64, nodeId uint64) {
// 		originalSendCallback(clientAddress, uid, nodeId)
// 		callbackReceived <- struct{}{}
// 	}

// 	server := NewGCSObjServer()
// 	server.sendCallback = mockSendCallback

// 	// First goroutine calls RequestLocation
// 	go func() {
// 		defer wg.Done()
// 		resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 1})
// 		if err != nil {
// 			t.Errorf("RequestLocation failed: %v", err)
// 			return
// 		}
// 		if resp.ImmediatelyFound {
// 			t.Errorf("Expected location not to be found immediately, but it was found")
// 		}
// 	}()

// 	// Second goroutine calls RequestLocation
// 	go func() {
// 		defer wg.Done()
// 		resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 1})
// 		if err != nil {
// 			t.Errorf("RequestLocation failed: %v", err)
// 			return
// 		}
// 		if resp.ImmediatelyFound {
// 			t.Errorf("Expected location not to be found immediately, but it was found")
// 		}
// 	}()

// 	// Third goroutine performs NotifyOwns after a short delay
// 	go func() {
// 		time.Sleep(100 * time.Millisecond)
// 		_, err := client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: 1, NodeId: 100})
// 		if err != nil {
// 			t.Errorf("NotifyOwns failed: %v", err)
// 		}
// 	}()

// 	// Wait for both callbacks to be received
// 	for i := 0; i < 2; i++ {
// 		select {
// 		case <-callbackReceived:
// 			// Callback received, continue
// 		case <-time.After(1 * time.Second):
// 			t.Errorf("Timeout waiting for callback")
// 		}
// 	}

// 	// Wait for all goroutines to finish
// 	wg.Wait()
// }

// type MockGCSObjServer struct {
// 	GCSObjServer
// }

// func (s *MockGCSObjServer) sendCallback(clientAddress string, uid uint64, nodeId uint64) {
// 	return
// }

// func MockNewGCSObjServer() *MockGCSObjServer {
// 	return NewGCSObjServer()
// }

// Helper function to check if a slice contains a specific value
func contains(slice []uint64, value uint64) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
