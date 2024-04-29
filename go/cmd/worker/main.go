package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "fmt"
    "encoding/json"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)

// Declare the global config variable
var cfg *config.Config

func main() {
    cfg := config.LoadConfig()
    address := ":" + strconv.Itoa(cfg.Ports.LocalWorkerStart)

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    s := grpc.NewServer()
    pb.RegisterWorkerServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

type server struct {
    pb.UnimplementedWorkerServer
}

// Implement your service methods here.

// executeFunction is a stub that simulates the execution of a Python function and returns a byte slice.
func executeFunction(f []byte, args []byte, kwargs []byte) []byte {
    // Placeholder result as a simple string, to be converted into bytes.
    result := "Placeholder result from Python function execution"
    
    // Convert the string result to a byte slice to satisfy the gRPC byte type requirement.
    resultBytes, _ := json.Marshal(result)
    return resultBytes
}


// run executes the function by fetching it, running it, and storing the result.
func (s *server) Run(ctx context.Context, req *pb.RunRequest) (*pb.StatusResponse, error) {
    // Assuming RunRequest contains uid, name, args, and kwargs

    // Connect to the gcs_func_gRPC service
    funcServiceAddr := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSFunctionTable)

    funcConn, err := grpc.Dial(funcServiceAddr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("failed to connect to func service: %v", err)
    }
    defer funcConn.Close()
    funcClient := pb.NewGCSFuncClient(funcConn)

    // Fetch function using gRPC call
    funcResponse, err := funcClient.FetchFunc(ctx, &pb.FetchRequest{Name: req.Name})
    if err != nil {
        return nil, err
    }
    output := executeFunction(funcResponse.SerializedFunc, req.Args, req.Kwargs) // executeFunction is a placeholder for actual function execution

    // Connect to the local_object_store_gRPC service
    // localObjStoreAddr := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSFunctionTable)
    storeConn, err := grpc.Dial("localhost:50000", grpc.WithInsecure())
    if err != nil {
        log.Fatalf("failed to connect to store service: %v", err)
    }
    defer storeConn.Close()
    storeClient := pb.NewLocalObjStoreClient(storeConn)

    // Store output using gRPC call
    _, err = storeClient.Store(ctx, &pb.StoreRequest{Uid: req.Uid, ObjectBytes: output})
    if err != nil {
        return nil, err
    }

    return &pb.StatusResponse{Success: true}, nil
}