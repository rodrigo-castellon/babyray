package main

import (
    "context"
    "net"
    "testing"
	"log"
    "sync"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    pb "github.com/rodrigo-castellon/babyray/pkg"
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

func TestRequestLocation(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()
    client := pb.NewGCSObjClient(conn)

    // First, ensure the object is registered to test retrieval
    _, err = client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{
        Uid:    1,
        NodeId: 100,
    })
    if err != nil {
        t.Fatalf("Setup failure: could not register UID: %v", err)
    }

    // Test RequestLocation
    resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: 1})
    if err != nil {
        t.Errorf("RequestLocation failed: %v", err)
        return
    }
    if resp.NodeId != 100 {
        t.Errorf("RequestLocation returned incorrect node ID: got %d, want %d", resp.NodeId, 100)
    }
}

// Create a unit test in Go where three goroutines are involved, with the first two waiting for an object's location 
// and the third notifying the server of the object's presence:
// - Two goroutines will call RequestLocation for a UID that initially doesn't have any node IDs associated with it. They will block until they are notified of a change.
// - One goroutine will perform the NotifyOwns action after a short delay, adding a node ID to the UID, which should then notify the waiting goroutines.
func TestRequestLocationWithNotification(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()
    client := pb.NewGCSObjClient(conn)

    var wg sync.WaitGroup
    uid := uint64(1)  // Example UID for testing

    // Start two goroutines that are trying to fetch the object location
    for i := 0; i < 2; i++ {
        wg.Add(1)
        go func(index int) {
            defer wg.Done()
            resp, err := client.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: uid})
            if err != nil {
                t.Errorf("Goroutine %d: RequestLocation failed: %v", index, err)
                return
            }
            if resp.NodeId != 100 {
                t.Errorf("Goroutine %d: RequestLocation returned incorrect node ID: got %d, want %d", index, resp.NodeId, 100)
            }
        }(i)
    }

    // Goroutine to notify
    wg.Add(1)
    go func() {
        defer wg.Done()
        // Let the requests initiate first
        time.Sleep(100 * time.Millisecond)
        _, err := client.NotifyOwns(ctx, &pb.NotifyOwnsRequest{
            Uid:    uid,
            NodeId: 100,
        })
        if err != nil {
            t.Fatalf("NotifyOwns failed: %v", err)
        }
    }()

    wg.Wait() // Wait for all goroutines to complete
}