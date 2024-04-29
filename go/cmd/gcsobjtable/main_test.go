package main

import (
    "context"
    "net"
    "testing"
	"log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    pb "github.com/rodrigo-castellon/babyray/pkg"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    lis = bufconn.Listen(bufSize)
    s := grpc.NewServer()
    pb.RegisterGCSObjServer(s, &GCSObjServer{objectLocations: make(map[uint32][]uint32)})
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

    // Test NotifyOwns
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
    if err != nil || !resp.Success {
        t.Errorf("RequestLocation failed: %v, response: %v", err, resp)
    }
    if resp.Details != "100" {
        t.Errorf("RequestLocation returned incorrect node ID: got %s, want %s", resp.Details, "100")
    }
}
