package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
	cfg = config.GetConfig() // Load configuration
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
func TestGetNodeId_2(t *testing.T) {
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

// Mock implementation of the LocalObjStoreServer
type mockLocalObjStoreServer struct {
	pb.UnimplementedLocalObjStoreServer
	callbackReceived chan struct{}
}

func (m *mockLocalObjStoreServer) LocationFound(ctx context.Context, req *pb.LocationFoundCallback) (*pb.StatusResponse, error) {
	m.callbackReceived <- struct{}{}
	return &pb.StatusResponse{Success: true}, nil
}

func NewMockLocalObjStoreServer(callbackBufferSize int) *mockLocalObjStoreServer {
	return &mockLocalObjStoreServer{
		callbackReceived: make(chan struct{}, callbackBufferSize),
	}
}

func TestSendCallback_ErrorCase(t *testing.T) {
	// Capture log output
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	// Mock client address
	clientAddress := "localhost:689"

	// Initialize the GCSObjServer
	server := &GCSObjServer{}

	// Run sendCallback to trigger an error case
	server.sendCallback(clientAddress, 1, 100)

	// Check the log output
	logOutput := logBuffer.String()
	expectedLogSubstring1 := "Failed to send LocationFound callback for UID"
	expectedLogSubstring2 := "connection refused"

	if !bytes.Contains([]byte(logOutput), []byte(expectedLogSubstring1)) {
		t.Errorf("Expected log message containing '%s' not found. Actual log: %s", expectedLogSubstring1, logOutput)
	}

	if !bytes.Contains([]byte(logOutput), []byte(expectedLogSubstring2)) {
		t.Errorf("Expected log message containing '%s' not found. Actual log: %s", expectedLogSubstring2, logOutput)
	}
}

// Checks for a callback hit using go channel
func TestSendCallback_Hit(t *testing.T) {
	// Create a context with a timeout to manage server and test lifecycle
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Prepare the network address
	address := ":" + strconv.Itoa(cfg.Ports.LocalObjectStore)
	lis, err := net.Listen("tcp", address)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	// Create and start the gRPC server
	s := grpc.NewServer()
	mock := NewMockLocalObjStoreServer(1)
	pb.RegisterLocalObjStoreServer(s, mock)
	log.Printf("server listening at %v", lis.Addr())

	// Run the server in a goroutine
	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Ensure the server is shut down cleanly at the end of the test
	defer func() {
		s.GracefulStop()
		lis.Close()
	}()

	// Allow some time for the server to start
	time.Sleep(200 * time.Millisecond)

	// Mock client address - ephemeral outbound
	host := "localhost"
	clientPort := strconv.Itoa(cfg.Ports.LocalObjectStore) // Replace the ephemeral port with the official LocalObjectStore port
	clientAddress := net.JoinHostPort(host, clientPort)

	// Initialize the GCSObjServer
	server := &GCSObjServer{}

	// Use a WaitGroup to wait for the callback goroutine
	var wg sync.WaitGroup
	wg.Add(1)

	// Run sendCallback as a goroutine
	go func() {
		defer wg.Done()
		server.sendCallback(clientAddress, 1, 100)
	}()

	// Wait for the callback or timeout
	select {
	case <-mock.callbackReceived:
		// Callback received, continue
	case <-ctx.Done():
		t.Errorf("Timeout waiting for callback")
	}

	// Wait for all goroutines to finish
	wg.Wait()
}

// func TestSendCallback_Hit(t *testing.T) {

// 	address := ":" + strconv.Itoa(cfg.Ports.LocalObjectStore) // Prepare the network address
// 	lis, err := net.Listen("tcp", address)
// 	if err != nil {
// 		log.Fatalf("failed to listen: %v", err)
// 	}
// 	s := grpc.NewServer()
// 	mock := NewMockLocalObjStoreServer(1)
// 	pb.RegisterLocalObjStoreServer(s, mock)
// 	log.Printf("server listening at %v", lis.Addr())
// 	if err := s.Serve(lis); err != nil {
// 		log.Fatalf("failed to serve: %v", err)
// 	}

// 	// Allow some time for the server to start
// 	time.Sleep(200 * time.Millisecond)

// 	// Mock client address - ephemeral outbound
// 	clientAddress := "localhost:777"

// 	// Initialize the GCSObjServer
// 	server := &GCSObjServer{}

// 	// Run sendCallback as a goroutine
// 	go server.sendCallback(clientAddress, 1, 100)

// 	// Catch it
// 	select {
// 	case <-mock.callbackReceived:
// 		// Callback received, continue
// 	case <-time.After(1 * time.Second):
// 		t.Errorf("Timeout waiting for callback")
// 	}
// }

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

// We don't listen for a callback in this test
func TestRequestLocation(t *testing.T) {
	clientAddress := "127.0.0.1:8080"
	addr, err := net.ResolveTCPAddr("tcp", clientAddress)
	if err != nil {
		fmt.Printf("Error resolving address: %v\n", err)
		return
	}
	p := &peer.Peer{
		Addr: addr,
	}

	// Create a new context with the peer information
	ctx := peer.NewContext(context.Background(), p)

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

type mockSchedulerClient struct {
	pb.GlobalSchedulerClient
	requestsReceived map[uint64]*pb.GlobalScheduleRequest
}

func (m *mockSchedulerClient) Schedule(ctx context.Context , req *pb.GlobalScheduleRequest, opts ...grpc.CallOption ) (*pb.StatusResponse, error) {
	m.requestsReceived[req.Uid] = req
	return nil, nil
}
func (m *mockSchedulerClient) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest, opts ...grpc.CallOption ) (*pb.StatusResponse, error) {
	return nil, nil
}
func (m *mockSchedulerClient) LiveNodesHeartbeat(ctx context.Context) (error)  {
	return nil
}
func (m *mockSchedulerClient) SendLiveNodes(ctx context.Context) {
	return nil
}
func TestNodeDiesWhileGenerating(t *testing.T) {
	/*
		-Worker tells GCS that its generating
		-Global Scheduler tells it that that node is dead
		-another node asks for object
		-global scheduler should receive a schedule request for that object
	*/
	ctx = context.Background()
	req := &pb.GlobalScheduleRequest{Uid: 200, Name: "func_name"}
	m := mockSchedulerClient {
		requestsReceived: make(map[uint64]bool), 
	}

	s := GCSObjServer {
		generating: make(map[uint64]uint64),
		globalSchedulerClient: m, 
		lineage: make(map[uint64]*pb.GlobalScheduleRequest),
		liveNodes: make(map[uint64]bool), 
	}

	s.RegisterLineage(ctx, req)

	s.RegisterGenerating(ctx, &pb.GeneratingRequest{Uid: 200, NodeId: 5})

	//Node 5 is assumed to be dead b/c we're not sending heartbeats. 

	s.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 200})

	if val, ok := m.requestsReceived[200]; !ok || val.Name != "func_name"{
		t.Errorf("Global scheduler never received new schedule request")
	}






	
}

// // MockGCSObjServer inherits GCSObjServer and overrides sendCallback
// type MockGCSObjServer struct {
// 	*GCSObjServer
// 	callbackReceived chan struct{}
// }

// func NewMockGCSObjServer(callbackBufferSize int) *MockGCSObjServer {
// 	return &MockGCSObjServer{
// 		GCSObjServer:     NewGCSObjServer(),
// 		callbackReceived: make(chan struct{}, callbackBufferSize),
// 	}
// }

// // sendCallback is the dummy method for testing
// func (s *MockGCSObjServer) sendCallback(clientAddress string, uid uint64, nodeId uint64) {
// 	//log.Printf("Mock sendCallback called with clientAddress: %s, uid: %d, nodeId: %d", clientAddress, uid, nodeId)
// 	s.callbackReceived <- struct{}{}
// }

// var mockLis *bufconn.Listener

// func mockBufDialer(context.Context, string) (net.Conn, error) {
// 	return mockLis.Dial()
// }

// A test which simulates the following:
// - Two LocalObjStore nodes will call RequestLocation for a UID that initially doesn't have any node IDs associated with it. They will not be happy until they are notified of a change.
// - One LocalObjStore node will perform the NotifyOwns action after a short delay, adding a node ID to the UID, which should then notify the waiting nodes.
// - One GCSObjTable node will be tested, but its sendCallback will be simulated

// Node A: a LocalObjStore requesting location
// Node B: a LocalObjStore requesting location
// Node C: a LocalObjStore notifying ownership
// Node D: a GCSObjTable being tested
// func TestRequestLocationNotifyOwnsHitsCallback(t *testing.T) {
// 	NUM_CALLBACKS_EXPECTED := 2

// 	// Initialize the server and listener
// 	mockLis = bufconn.Listen(bufSize)
// 	s := grpc.NewServer()
// 	mock := NewMockGCSObjServer(NUM_CALLBACKS_EXPECTED) // USE OF MOCK HERE
// 	pb.RegisterGCSObjServer(s, mock)
// 	go func() {
// 		if err := s.Serve(mockLis); err != nil {
// 			log.Fatalf("Server exited with error: %v", err)
// 		}
// 	}()

// 	// Setup gRPC client
// 	ctx := context.Background()
// 	conn, err := grpc.DialContext(ctx, "junk", grpc.WithContextDialer(mockBufDialer), grpc.WithInsecure())
// 	if err != nil {
// 		t.Fatalf("Failed to dial bufnet: %v", err)
// 	}
// 	defer conn.Close()
// 	client := pb.NewGCSObjClient(conn)

// 	// Simulate Node A calling RequestLocation
// 	resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 555})
// 	if err != nil {
// 		t.Errorf("RequestLocation failed: %v", err)
// 		return
// 	}
// 	if resp.ImmediatelyFound {
// 		t.Errorf("Expected location not to be found immediately, but it was found")
// 	}

// 	// Simulate Node B calling RequestLocation
// 	resp, err = client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 555})
// 	if err != nil {
// 		t.Errorf("RequestLocation failed: %v", err)
// 		return
// 	}
// 	if resp.ImmediatelyFound {
// 		t.Errorf("Expected location not to be found immediately, but it was found")
// 	}

// 	// Simulate Node C calling NotifyOwns
// 	_, err = client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: 555, NodeId: 100})
// 	if err != nil {
// 		t.Errorf("NotifyOwns failed: %v", err)
// 	}

// 	// Wait for both callbacks to be received
// 	for i := 0; i < 2; i++ {
// 		select {
// 		case <-mock.callbackReceived:
// 			// Callback received, continue
// 		case <-time.After(1 * time.Second):
// 			t.Errorf("Timeout waiting for callback")
// 		}
// 	}
// }

// ==============

// Unit test in Go

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
