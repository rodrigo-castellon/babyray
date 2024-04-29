package main

import (
    "context"
    "net"
    "log"
    // "reflect"
    "testing"

    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    pb "github.com/rodrigo-castellon/babyray/pkg"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    lis = bufconn.Listen(bufSize)
    s := grpc.NewServer()
    pb.RegisterWorkerServer(s, &server{
    })
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Fatalf("Server exited with error: %v", err)
        }
    }()
}

func bufDialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

func TestWorkerRun(t *testing.T) {

    // scaffold for unit test.

    // ctx := context.Background()
    // conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    // if err != nil {
    //     t.Fatalf("Failed to dial bufnet: %v", err)
    // }
    // defer conn.Close()
    // client := pb.NewWorkerClient(conn)

    // Setup test data
    // uid := "unique_id_123"
    // funcName := "testFunction"
    // args := []byte("args data")
    // kwargs := []byte("kwargs data")

    // Mock setup: Register the function and prepare the expected output
    // expectedOutput := []byte("function output")
    // funcServer.RegisterFunc(ctx, &funcpb.RegisterRequest{Name: funcName, SerializedFunc: args})
    // storeServer.Store(ctx, &storepb.StoreRequest{Uid: uid, Output: expectedOutput})

    // Run the worker service
    // resp, err := client.Run(ctx, &pb.RunRequest{Uid: uid, Name: funcName, Args: args, Kwargs: kwargs})
    // if err != nil {
    //     t.Errorf("Run failed: %v", err)
    // } else if !resp.Success {
    //     t.Errorf("Run returned unsuccessful")
    // }

    // Validate the stored output
    // Add additional checks to verify that the function was executed and the output was stored correctly
    // This can include fetching data from the mock store service and comparing it to `expectedOutput`
}

