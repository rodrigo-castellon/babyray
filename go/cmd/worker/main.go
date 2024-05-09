package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "fmt"
    // "encoding/json"
    "bytes"
    "os/exec"
    "encoding/base64"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)

// Declare the global config variable
var cfg *config.Config

type ClientConstructor[T any] func(grpc.ClientConnInterface) T

func createGRPCClient[T any](address string, constructor ClientConstructor[T]) T {
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("failed to connect to %s: %v", address, err)
    }
    defer conn.Close()
    return constructor(conn)
}

func main() {
    cfg := config.GetConfig()
    address := ":" + strconv.Itoa(cfg.Ports.LocalWorkerStart)

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }

    // Set up the clients
    funcServiceAddr := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSFunctionTable)
    funcClient := createGRPCClient[pb.GCSFuncClient](funcServiceAddr, pb.NewGCSFuncClient)
    storeClient := createGRPCClient[pb.LocalObjStoreClient]("localhost:50000", pb.NewLocalObjStoreClient)

    s := grpc.NewServer()
    pb.RegisterWorkerServer(s, &workerServer{
        funcClient: funcClient,
        storeClient: storeClient,
    })

    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
}

type workerServer struct {
    pb.UnimplementedWorkerServer
    funcClient FuncClient
    storeClient StoreClient
}

type FuncClient interface {
    FetchFunc(ctx context.Context, in *pb.FetchRequest, opts ...grpc.CallOption) (*pb.FetchResponse, error)
}

type StoreClient interface {
    Store(ctx context.Context, in *pb.StoreRequest, opts ...grpc.CallOption) (*pb.StatusResponse, error)
}

func executeFunction(f []byte, args []byte, kwargs []byte) ([]byte, error) {
    // Prepare the command to run the Python script
    cmd := exec.Command("sh", "-c", "python3 execute.py")

    // Create a buffer to hold the serialized data
    inputBuffer := bytes.NewBuffer(nil)

    // Assume data is the serialized data you want to encode in base64
    // fB64 := base64.StdEncoding.EncodeToString(f)
    // argsB64 := base64.StdEncoding.EncodeToString(args)
    // kwargsB64 := base64.StdEncoding.EncodeToString(kwargs)

    // Write the function, args, and kwargs to the buffer
    inputBuffer.Write(f)
    // inputBuffer.writes(fB64)
    inputBuffer.WriteByte('\n')
    // inputBuffer.Write(argsB64)
    inputBuffer.Write(args)
    inputBuffer.WriteByte('\n')
    // inputBuffer.Write(kwargsB64)
    inputBuffer.Write(kwargs)

    // Set the stdin to our input buffer
    cmd.Stdin = inputBuffer

    // Capture the output
    output, err := cmd.Output()
    if err != nil {
        log.Fatalf("Error executing function: %v", err)
    }

    // Decode the Base64 output to get the original pickled data
    data, err := base64.StdEncoding.DecodeString(string(output))
    if err != nil {
        log.Fatalf("Error decoding Base64: %v", err)
    }

    // Return the output from the Python script
    return data, nil
}

// run executes the function by fetching it, running it, and storing the result.
func (s *workerServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.StatusResponse, error) {
    // Assuming RunRequest contains uid, name, args, and kwargs

    // Connect to the gcs_func_gRPC service

    // Fetch function using gRPC call
    funcResponse, err := s.funcClient.FetchFunc(ctx, &pb.FetchRequest{Name: req.Name})
    if err != nil {
        return nil, err
    }

    output, _ := executeFunction(funcResponse.SerializedFunc, req.Args, req.Kwargs)

    // Store output using gRPC call
    _, err = s.storeClient.Store(ctx, &pb.StoreRequest{Uid: req.Uid, ObjectBytes: output})
    if err != nil {
        return nil, err
    }

    return &pb.StatusResponse{Success: true}, nil
}