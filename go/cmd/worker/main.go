package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "fmt"
    // "encoding/json"
    // "bytes"
    "os/exec"
    "encoding/base64"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)

// Declare the global config variable
var cfg *config.Config

func main() {
    cfg := config.GetConfig()
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

func executeFunction(f []byte, args []byte, kwargs []byte) ([]byte, error) {
    // Prepare the command to run the Python script
    // cmd := exec.Command("python3", "execute.py")
    // cmd := exec.Command("which", "python3")
    // cmd := exec.Command("command", "-v", "python3", ">/dev/null", "&&", "echo", "$(which", "python3)", "||", "echo", `"not found"`)
    // cmd := exec.Command("which", "python3", ">/dev/null", "&&", "echo", "$(which", "python3)", "||", "echo", `"not found"`)
    cmd := exec.Command("pwd")

    // Create a buffer to hold the serialized data
    // inputBuffer := bytes.NewBuffer(nil)

    // // Write the function, args, and kwargs to the buffer
    // inputBuffer.Write(f)
    // inputBuffer.WriteByte('\n')
    // inputBuffer.Write(args)
    // inputBuffer.WriteByte('\n')
    // inputBuffer.Write(kwargs)

    // // Set the stdin to our input buffer
    // cmd.Stdin = inputBuffer

    // Capture the output
    output, err := cmd.Output()
    if err != nil {
        log.Fatalf("Error executing function: %v", err)
    }

    log.Printf("output: %v", output)
    log.Printf("output: %s", output)

    // Decode the Base64 output to get the original pickled data
    data, err := base64.StdEncoding.DecodeString(string(output))
    if err != nil {
        log.Fatalf("Error decoding Base64: %v", err)
    }

    // Return the output from the Python script
    return data, nil
}

// run executes the function by fetching it, running it, and storing the result.
func (s *server) Run(ctx context.Context, req *pb.RunRequest) (*pb.StatusResponse, error) {
    // Assuming RunRequest contains uid, name, args, and kwargs

    // Connect to the gcs_func_gRPC service
    cfg := config.GetConfig()
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

    output, _ := executeFunction(funcResponse.SerializedFunc, req.Args, req.Kwargs)

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