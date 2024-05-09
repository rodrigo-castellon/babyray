package main

import (
    "context"
    "net"
    "testing"
	"log"
	"bytes"
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    pb "github.com/rodrigo-castellon/babyray/pkg"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    lis = bufconn.Listen(bufSize)
    s := grpc.NewServer()
    pb.RegisterLocalObjStoreServer(s, &server{})
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Fatalf("Server exited with error: %v", err)
        }
    }()
}

func bufDialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

func TestStoreAndGet(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())

    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()
	data := []byte{72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100}
    client := pb.NewLocalObjStoreClient(conn)

    // Test Store
    resp, err := client.Store(ctx, &pb.StoreRequest{
        Uid:    1,
        ObjectBytes: []byte{72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100},
    })
    if err != nil || !resp.Success {
        t.Errorf("Store failed: %v, response: %v", err, resp)
    }

	// Test Get
	resp2, err2 := client.Get(ctx, &pb.GetRequest{
		Uid: 1
	})
	if err2 != nil || !bytes.Equals(data, resp.ObjectBytes) {
		t.Errorf("Get failed: %v, response: %v", err2, resp2)
	}

}

func TestLocationFound(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }g
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
