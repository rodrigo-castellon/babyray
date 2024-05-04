package main

import (
    "context"
    "net"
    "log"
    // "reflect"
    "testing"
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

func TestExecuteFunction(t *testing.T) {
    // Expected output from the Python script
    // These bytes are a result of running:
    // >> import dill as pickle; print([x for x in pickle.dumps(8)])
    // in Python.

    // expectedOutput = 8
    expectedOutput := []byte{128, 4, 75, 8, 46}

    // Create the test input. All created by doing:
    // base64.b64encode(pickle.dumps(python_obj)).decode('ascii')
    // where python_obj is either the function, args, or kwargs

    // f = lambda x,y: x + y
    f := []byte("gASVBgEAAAAAAACMCmRpbGwuX2RpbGyUjBBfY3JlYXRlX2Z1bmN0aW9ulJOUKGgAjAxfY3JlYXRlX2NvZGWUk5QoSwJLAEsASwJLAktTQwh8AHwBFwBTAJROhZQpjAF4lIwBeZSGlIwtL1VzZXJzL3JvZHJpZ28tY2FzdGVsbG9uL2JhYnlyYXkvZ28vc2NyaXB0LnB5lIwIPGxhbWJkYT6USwZDAJQpKXSUUpRjX19idWlsdGluX18KX19tYWluX18KaAtOTnSUUpR9lH2UKIwPX19hbm5vdGF0aW9uc19flH2UjAxfX3F1YWxuYW1lX1+UjBZtYWluLjxsb2NhbHM+LjxsYW1iZGE+lHWGlGIu") // Serialized Python function
    // args = [5,3]
    args := []byte("gASVBwAAAAAAAABLBUsDhpQu")
    // kwargs = dict()
    kwargs := []byte("gAR9lC4=")

    // Call executeFunction
    output, err := executeFunction(f, args, kwargs)
    if err != nil {
        t.Fatalf("executeFunction returned an error: %v", err)
    }

    if !bytes.Equal(output, expectedOutput) {
        // Just print out the bytes
        t.Errorf("Expected %v, got %v", expectedOutput, output)
    }
}

func TestWorkerRun(t *testing.T) {

    // scaffold for unit test.

    ctx := context.Background()

    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    if err != nil {
        t.Fatalf("Failed to dial bufnet: %v", err)
    }
    defer conn.Close()
    client := pb.NewWorkerClient(conn)


    // Setup test data
    uid := uint32(383838)
    funcName := "testFunction"
    args := []byte("args data")
    kwargs := []byte("kwargs data")

    // Mock setup: Register the function and prepare the expected output
    // expectedOutput := []byte("function output")
    // funcServer.RegisterFunc(ctx, &funcpb.RegisterRequest{Name: funcName, SerializedFunc: args})
    // storeServer.Store(ctx, &storepb.StoreRequest{Uid: uid, Output: expectedOutput})

    // Run the worker service
    resp, err := client.Run(ctx, &pb.RunRequest{Uid: uid, Name: funcName, Args: args, Kwargs: kwargs})
    _ = resp
    // if err != nil {
    //     t.Errorf("Run failed: %v", err)
    // } else if !resp.Success {
    //     t.Errorf("Run returned unsuccessful")
    // }

    // Validate the stored output
    // Add additional checks to verify that the function was executed and the output was stored correctly
    // This can include fetching data from the mock store service and comparing it to `expectedOutput`
}

