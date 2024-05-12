package main

import (
    "context"
    "net"
    "testing"
	"bytes"
    "time"
    "google.golang.org/grpc"
    "google.golang.org/grpc/test/bufconn"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func init() {
    startServer("50051")
    // lis = bufconn.Listen(bufSize)
    // s := grpc.NewServer()
    // pb.RegisterLocalObjStoreServer(s, &server{})
    // go func() {
    //     if err := s.Serve(lis); err != nil {
    //         log.Fatalf("Server exited with error: %v", err)
    //     }
    // }()
}

func startServer(port string) (*grpc.Server, error) {

    lis, err := net.Listen("tcp", port)
    if err != nil {
        return nil, err
    }
    s := grpc.NewServer()
    pb.RegisterLocalObjStoreServer(s, &server{})
    go func() {
        s.Serve(lis)
    }()
    return s, nil
}

func bufDialer(context.Context, string) (net.Conn, error) {
    return lis.Dial()
}
type mockGCSClient struct {
    pb.GCSObjClient
    statusResp *pb.StatusResponse
    resp *pb.RequestLocationResponse
    err error
}

func(m *mockGCSClient) RequestLocation(ctx context.Context, in *pb.RequestLocationRequest, opts ...grpc.CallOption) (*pb.RequestLocationResponse, error) {
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

func (m *mockStoreClient) Get(ctx context.Context, in *pb.GetRequest, opts ...grpc.CallOption) (*pb.GetResponse, error) {
    return m.resp, m.err
}

func TestStoreAndGet_Local(t *testing.T) {
     ctx := context.Background()
    // conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())

    // if err != nil {
    //     t.Fatalf("Failed to dial bufnet: %v", err)
    // }
    _, err := startServer("50051")
    if err != nil {
        t.Fatalf("failed to start server: %v", err)
    }
    conn, err := grpc.DialContext(ctx, "localhost:50051", grpc.WithInsecure())
    defer conn.Close()
	data := []byte{72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100}
    client := pb.NewLocalObjStoreClient(conn)

    // Test Store
    resp, err := client.Store(ctx, &pb.StoreRequest{
        Uid:    1,
        ObjectBytes: data,
    })
    if err != nil || !resp.Success {
        t.Errorf("Store failed: %v, response: %v", err, resp)
    }

	// Test Get
	resp2, err2 := client.Get(ctx, &pb.GetRequest{
		Uid: 1, 
        Testing: true,
	})
	if err2 != nil || !bytes.Equal(data, resp2.ObjectBytes) {
		t.Errorf("Get failed: %v, response: %v", err2, resp2)
	}

}
func TestStoreAndGet_External(t *testing.T) {
	ctx := context.Background()
    s1, err := startServer("50051")
    if err != nil {
        t.Fatalf("Failed to start server 1: %v", err)
    }
    defer s1.Stop()

    s2, err := startServer("50052")
    if err != nil {
        t.Fatalf("Failed to start server 2: %v", err)
    }
    defer s2.Stop()


    if err != nil {
        t.Fatalf("Failed to dial server 1: %v", err)
    }
    conn1, err1 := grpc.DialContext(ctx, "localhost:50051", grpc.WithInsecure())
    if err1 != nil {
        t.Fatalf("Failed to dial LOC1")

    }

    conn2, err2 := grpc.DialContext(ctx, "localhost:50052", grpc.WithInsecure())
    if err2 != nil {
        t.Fatalf("failed to dial LOC2")
    }
    defer conn1.Close()
    defer conn2.Close()
    client1 := pb.NewLocalObjStoreClient(conn1)
    client2 := pb.NewLocalObjStoreClient(conn2)

	data := []byte{72, 101, 108, 108, 111, 32, 87, 111, 114, 108, 100}
    
    // mockStore := &mockStoreClient{
    //     statusResp: &pb.StatusResponse{},
    //     resp: &pb.GetResponse{Uid: 1, ObjectBytes: data},
    //     err: nil,
    // }

    // mockGCSClient := &mockGCSClient{
    //     statusResp: &pb.StatusResponse{},
    //     resp: &pb.RequestLocationResponse{NodeId: 2}, 
    //     err: nil
    // }   
    
    sresp, err := client2.Store(ctx, &pb.StoreRequest{Uid: 1, ObjectBytes: data})
    if !sresp.Success || err != nil {
        t.Errorf("Failed to store value on LOS 2")
    } 
    response2, err := client2.Get(ctx, &pb.GetRequest{Uid: 1, Testing: true})

    if err != nil || !bytes.Equal(response2.ObjectBytes, data) {
        t.Errorf("Failed to get value on LOS 2")
    }

    
    go func(){ 
        response1, err := client1.Get(ctx, &pb.GetRequest{Uid: 1, Testing: true})
        if err != nil || !bytes.Equal(response1.ObjectBytes, data) {
            t.Errorf("Failed to get value on LOS 1")

        }
    
    }()


    time.Sleep(1 * time.Second)  

    locStatusResp, err := client1.LocationFound(ctx, &pb.LocationFoundResponse{Uid: 1, Address: "localhost", Port: 50052})

    if !locStatusResp.Success || err != nil {
        t.Errorf("Failed to tell LOS 1 about location")
    }


	
}

// func TestLocationFound(t *testing.T) {
//     ctx := context.Background()
//     conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
//     if err != nil {
//         t.Fatalf("Failed to dial bufnet: %v", err)
//     }
//     defer conn.Close()
//     client := pb.NewLocalObjStoreClient(conn)
// 	uid := 100
// 	go {
// 		client.Get(pb.GetRequest{Uid: 100})
// 	}

// }

// func TestLocationFound(t *testing.T) {
// 	mockGCSClient := newMockLocalObjStoreClient()

// 	// Initialize a mock server and required dependencies
// 	s := &server{
// 		gcsObjClient:       mockGCSClient,
// 		localObjectChannels: map[string]chan []byte{"testUID": make(chan []byte, 1)},
// 	}

// 	ctx := context.Background()
// 	resp := &pb.LocationFoundResponse{
// 		Uid:      "testUID",
// 		Location: 1,
// 	}

// 	statusResp, err := s.LocationFound(ctx, resp)
// 	require.NoError(t, err)
// 	require.NotNil(t, statusResp)
// 	require.True(t, statusResp.Success)

// 	select {
// 	case data := <-s.localObjectChannels["testUID"]:
// 		require.Equal(t, []byte("mockObjectData"), data)
// 	default:
// 		t.Fatal("Expected data not found in the channel")
// 	}
// }
