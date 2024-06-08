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
    pb.RegisterWorkerServer(s, &workerServer{})
    go func() {
        if err := s.Serve(lis); err != nil {
            log.Fatalf("Server exited with error: %v", err)
        }
    }()
}

type mockFuncClient struct {
    pb.GCSFuncClient // Embedding the interface for forward compatibility
    resp *pb.FetchResponse
    err  error
}

func (m *mockFuncClient) FetchFunc(ctx context.Context, in *pb.FetchRequest, opts ...grpc.CallOption) (*pb.FetchResponse, error) {
    return m.resp, m.err
}

type mockStoreClient struct {
    pb.LocalObjStoreClient // Embedding the interface for forward compatibility
    statusResp *pb.StatusResponse
    resp *pb.GetResponse
    err  error
}

func (m *mockStoreClient) Store(ctx context.Context, in *pb.StoreRequest, opts ...grpc.CallOption) (*pb.StatusResponse, error) {
    return m.statusResp, m.err
}

func bufDialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}

func TestExecuteFunction(t *testing.T) {
    // Expected output from the Python script
    // These bytes are a result of running:
    // >> import cloudpickle as pickle; print([x for x in pickle.dumps(8)])
    // in Python.

    // expectedOutput = 8
    expectedOutput := []byte{128, 5, 75, 8, 46}

    // Create the test input. All created by doing:
    // python3 -c "import cloudpickle as pickle; print([x for x in pickle.dumps(python_obj)])"
    // where python_obj is either the function, args, or kwargs

    // f = lambda x,y: x + y
    f := []byte{128, 5, 149, 162, 1, 0, 0, 0, 0, 0, 0, 140, 23, 99, 108, 111, 117, 100, 112, 105, 99, 107, 108, 101, 46, 99, 108, 111, 117, 100, 112, 105, 99, 107, 108, 101, 148, 140, 14, 95, 109, 97, 107, 101, 95, 102, 117, 110, 99, 116, 105, 111, 110, 148, 147, 148, 40, 104, 0, 140, 13, 95, 98, 117, 105, 108, 116, 105, 110, 95, 116, 121, 112, 101, 148, 147, 148, 140, 8, 67, 111, 100, 101, 84, 121, 112, 101, 148, 133, 148, 82, 148, 40, 75, 2, 75, 0, 75, 0, 75, 2, 75, 2, 75, 3, 67, 12, 151, 0, 124, 0, 124, 1, 122, 0, 0, 0, 83, 0, 148, 78, 133, 148, 41, 140, 1, 120, 148, 140, 1, 121, 148, 134, 148, 140, 8, 60, 115, 116, 114, 105, 110, 103, 62, 148, 140, 8, 60, 108, 97, 109, 98, 100, 97, 62, 148, 104, 14, 75, 1, 67, 10, 128, 0, 200, 17, 200, 49, 201, 19, 128, 0, 148, 67, 0, 148, 41, 41, 116, 148, 82, 148, 125, 148, 40, 140, 11, 95, 95, 112, 97, 99, 107, 97, 103, 101, 95, 95, 148, 78, 140, 8, 95, 95, 110, 97, 109, 101, 95, 95, 148, 140, 8, 95, 95, 109, 97, 105, 110, 95, 95, 148, 117, 78, 78, 78, 116, 148, 82, 148, 104, 0, 140, 18, 95, 102, 117, 110, 99, 116, 105, 111, 110, 95, 115, 101, 116, 115, 116, 97, 116, 101, 148, 147, 148, 104, 24, 125, 148, 125, 148, 40, 104, 21, 104, 14, 140, 12, 95, 95, 113, 117, 97, 108, 110, 97, 109, 101, 95, 95, 148, 104, 14, 140, 15, 95, 95, 97, 110, 110, 111, 116, 97, 116, 105, 111, 110, 115, 95, 95, 148, 125, 148, 140, 14, 95, 95, 107, 119, 100, 101, 102, 97, 117, 108, 116, 115, 95, 95, 148, 78, 140, 12, 95, 95, 100, 101, 102, 97, 117, 108, 116, 115, 95, 95, 148, 78, 140, 10, 95, 95, 109, 111, 100, 117, 108, 101, 95, 95, 148, 104, 22, 140, 7, 95, 95, 100, 111, 99, 95, 95, 148, 78, 140, 11, 95, 95, 99, 108, 111, 115, 117, 114, 101, 95, 95, 148, 78, 140, 23, 95, 99, 108, 111, 117, 100, 112, 105, 99, 107, 108, 101, 95, 115, 117, 98, 109, 111, 100, 117, 108, 101, 115, 148, 93, 148, 140, 11, 95, 95, 103, 108, 111, 98, 97, 108, 115, 95, 95, 148, 125, 148, 117, 134, 148, 134, 82, 48, 46}
    // args = [5,3]
    args := []byte{128, 5, 149, 9, 0, 0, 0, 0, 0, 0, 0, 93, 148, 40, 75, 5, 75, 3, 101, 46}
    // kwargs = dict()
    kwargs := []byte{128, 5, 125, 148, 46}

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

    // Set up mocks
    mockFunc := &mockFuncClient{
        resp: &pb.FetchResponse{
            // lambda x,y: x + y
            SerializedFunc: []byte{128, 5, 149, 162, 1, 0, 0, 0, 0, 0, 0, 140, 23, 99, 108, 111, 117, 100, 112, 105, 99, 107, 108, 101, 46, 99, 108, 111, 117, 100, 112, 105, 99, 107, 108, 101, 148, 140, 14, 95, 109, 97, 107, 101, 95, 102, 117, 110, 99, 116, 105, 111, 110, 148, 147, 148, 40, 104, 0, 140, 13, 95, 98, 117, 105, 108, 116, 105, 110, 95, 116, 121, 112, 101, 148, 147, 148, 140, 8, 67, 111, 100, 101, 84, 121, 112, 101, 148, 133, 148, 82, 148, 40, 75, 2, 75, 0, 75, 0, 75, 2, 75, 2, 75, 3, 67, 12, 151, 0, 124, 0, 124, 1, 122, 0, 0, 0, 83, 0, 148, 78, 133, 148, 41, 140, 1, 120, 148, 140, 1, 121, 148, 134, 148, 140, 8, 60, 115, 116, 114, 105, 110, 103, 62, 148, 140, 8, 60, 108, 97, 109, 98, 100, 97, 62, 148, 104, 14, 75, 1, 67, 10, 128, 0, 200, 17, 200, 49, 201, 19, 128, 0, 148, 67, 0, 148, 41, 41, 116, 148, 82, 148, 125, 148, 40, 140, 11, 95, 95, 112, 97, 99, 107, 97, 103, 101, 95, 95, 148, 78, 140, 8, 95, 95, 110, 97, 109, 101, 95, 95, 148, 140, 8, 95, 95, 109, 97, 105, 110, 95, 95, 148, 117, 78, 78, 78, 116, 148, 82, 148, 104, 0, 140, 18, 95, 102, 117, 110, 99, 116, 105, 111, 110, 95, 115, 101, 116, 115, 116, 97, 116, 101, 148, 147, 148, 104, 24, 125, 148, 125, 148, 40, 104, 21, 104, 14, 140, 12, 95, 95, 113, 117, 97, 108, 110, 97, 109, 101, 95, 95, 148, 104, 14, 140, 15, 95, 95, 97, 110, 110, 111, 116, 97, 116, 105, 111, 110, 115, 95, 95, 148, 125, 148, 140, 14, 95, 95, 107, 119, 100, 101, 102, 97, 117, 108, 116, 115, 95, 95, 148, 78, 140, 12, 95, 95, 100, 101, 102, 97, 117, 108, 116, 115, 95, 95, 148, 78, 140, 10, 95, 95, 109, 111, 100, 117, 108, 101, 95, 95, 148, 104, 22, 140, 7, 95, 95, 100, 111, 99, 95, 95, 148, 78, 140, 11, 95, 95, 99, 108, 111, 115, 117, 114, 101, 95, 95, 148, 78, 140, 23, 95, 99, 108, 111, 117, 100, 112, 105, 99, 107, 108, 101, 95, 115, 117, 98, 109, 111, 100, 117, 108, 101, 115, 148, 93, 148, 140, 11, 95, 95, 103, 108, 111, 98, 97, 108, 115, 95, 95, 148, 125, 148, 117, 134, 148, 134, 82, 48, 46},//("gASVBgEAAAAAAACMCmRpbGwuX2RpbGyUjBBfY3JlYXRlX2Z1bmN0aW9ulJOUKGgAjAxfY3JlYXRlX2NvZGWUk5QoSwJLAEsASwJLAktTQwh8AHwBFwBTAJROhZQpjAF4lIwBeZSGlIwtL1VzZXJzL3JvZHJpZ28tY2FzdGVsbG9uL2JhYnlyYXkvZ28vc2NyaXB0LnB5lIwIPGxhbWJkYT6USwZDAJQpKXSUUpRjX19idWlsdGluX18KX19tYWluX18KaAtOTnSUUpR9lH2UKIwPX19hbm5vdGF0aW9uc19flH2UjAxfX3F1YWxuYW1lX1+UjBZtYWluLjxsb2NhbHM+LjxsYW1iZGE+lHWGlGIu")
        },
        err:  nil,
    }
    mockStore := &mockStoreClient{
        statusResp: &pb.StatusResponse{},
        resp: &pb.GetResponse{},
        err: nil,
    }

    s := workerServer{
        funcClient: mockFunc,
        storeClient: mockStore,
    }

    // Setup test data
    uid := uint64(383838)
    funcName := uint64(557379)
    // [1,1]
    args := []byte{128, 5, 149, 9, 0, 0, 0, 0, 0, 0, 0, 93, 148, 40, 75, 1, 75, 1, 101, 46}//("gASVBwAAAAAAAABLBUsDhpQu")
    // dict()
    kwargs := []byte{128, 5, 125, 148, 46}//("gAR9lC4=")

    // Mock setup: Register the function and prepare the expected output
    // expectedOutput := []byte("function output")
    // funcServer.RegisterFunc(ctx, &funcpb.RegisterRequest{Name: funcName, SerializedFunc: args})
    // storeServer.Store(ctx, &storepb.StoreRequest{Uid: uid, Output: expectedOutput})

    // Run the worker service
    resp, err := s.Run(ctx, &pb.RunRequest{Uid: uid, Name: funcName, Args: args, Kwargs: kwargs})
    _ = resp
    _ = err

    // log.Printf("%v", resp)
    // if err != nil {
    //     t.Errorf("Run failed: %v", err)
    // } else if !resp.Success {
    //     t.Errorf("Run returned unsuccessful")
    // }

    // Validate the stored output
    // Add additional checks to verify that the function was executed and the output was stored correctly
    // This can include fetching data from the mock store service and comparing it to `expectedOutput`
}

