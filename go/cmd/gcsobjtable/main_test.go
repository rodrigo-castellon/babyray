package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"testing"
	"time"

	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
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

// func TestRequestLocation(t *testing.T) {
// 	ctx := context.Background()
// 	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
// 	if err != nil {
// 		t.Fatalf("Failed to dial bufnet: %v", err)
// 	}
// 	defer conn.Close()
// 	client := pb.NewGCSObjClient(conn)

// 	// First, ensure the object is registered to test retrieval
// 	_, err = client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{
// 		Uid:    1,
// 		NodeId: 100,
// 	})
// 	if err != nil {
// 		t.Fatalf("Setup failure: could not register UID: %v", err)
// 	}

// 	// Test RequestLocation
// 	resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 1})
// 	if err != nil {
// 		t.Errorf("RequestLocation failed: %v", err)
// 		return
// 	}
// 	if resp.Location != 100 {
// 		t.Errorf("RequestLocation returned incorrect node ID: got %d, want %d", resp.Location, 100)
// 	}
// }

// Create a unit test in Go where three goroutines are involved, with the first two waiting for an object's location
// and the third notifying the server of the object's presence:
// - Two goroutines will call RequestLocation for a UID that initially doesn't have any node IDs associated with it. They will block until they are notified of a change.
// - One goroutine will perform the NotifyOwns action after a short delay, adding a node ID to the UID, which should then notify the waiting goroutines.
// func TestRequestLocationWithNotification(t *testing.T) {
// 	ctx := context.Background()
// 	conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
// 	if err != nil {
// 		t.Fatalf("Failed to dial bufnet: %v", err)
// 	}
// 	defer conn.Close()
// 	client := pb.NewGCSObjClient(conn)

// 	var wg sync.WaitGroup
// 	uid := uint64(1) // Example UID for testing

// 	// Start two goroutines that are trying to fetch the object location
// 	for i := 0; i < 2; i++ {
// 		wg.Add(1)
// 		go func(index int) {
// 			defer wg.Done()
// 			resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: uid})
// 			if err != nil {
// 				t.Errorf("Goroutine %d: RequestLocation failed: %v", index, err)
// 				return
// 			}
// 			if resp.Location != 100 {
// 				t.Errorf("Goroutine %d: RequestLocation returned incorrect node ID: got %d, want %d", index, resp.Location, 100)
// 			}
// 		}(i)
// 	}

// 	// Goroutine to notify
// 	wg.Add(1)
// 	go func() {
// 		defer wg.Done()
// 		// Let the requests initiate first
// 		time.Sleep(100 * time.Millisecond)
// 		_, err := client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{
// 			Uid:    uid,
// 			NodeId: 100,
// 		})
// 		if err != nil {
// 			t.Fatalf("NotifyOwns failed: %v", err)
// 		}
// 	}()

// 	wg.Wait() // Wait for all goroutines to complete
// }

// Test the getNodeId function
func TestGetNodeId(t *testing.T) {
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

// Helper function to check if a slice contains a specific value
func contains(slice []uint64, value uint64) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}
