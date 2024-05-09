package main

import (
    "context"
    "net"
    "testing"
	"log"
	"reflect"
    // "time"
    // "sync"

    // "google.golang.org/grpc/status"
    // "google.golang.org/grpc/codes"
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    pb "github.com/rodrigo-castellon/babyray/pkg"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    lis = bufconn.Listen(bufSize)
    s := grpc.NewServer()
    pb.RegisterGCSFuncServer(s, NewGCSFuncServer())
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Fatalf("Server exited with error: %v", err)
        }
    }()
}

func bufDialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

func TestRegisterFunc(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()
    client := pb.NewGCSFuncClient(conn)

    testFunc := []byte("function data")
    resp, err := client.RegisterFunc(ctx, &pb.RegisterRequest{SerializedFunc: testFunc})
    if err != nil {
        t.Errorf("RegisterFunc failed: %v, response: %v", err, resp)
    }
}

func TestFetchFunc(t *testing.T) {
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()
    client := pb.NewGCSFuncClient(conn)

    // First, register a function to fetch
    testFunc := []byte("fetch data alskfjlaskdfnlaskfnlaskdnfl;asjdfn")
    resp, err := client.RegisterFunc(ctx, &pb.RegisterRequest{SerializedFunc: testFunc})
    if err != nil {
        t.Fatalf("Setup failure: could not register function: %v", err)
    }

    // Now, test fetching
    fetchResp, err := client.FetchFunc(ctx, &pb.FetchRequest{Name: resp.Name})
    if err != nil {
        t.Errorf("FetchFunc failed: %v", err)
    } else if fetchResp.SerializedFunc == nil {
        t.Errorf("FetchFunc returned nil for SerializedFunc")
    } else if !reflect.DeepEqual(fetchResp.SerializedFunc, testFunc) {
        t.Errorf("FetchFunc returned incorrect data: got %v, want %v", fetchResp.SerializedFunc, testFunc)
    }
}

