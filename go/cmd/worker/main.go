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
    "os"
    "path/filepath"
    "encoding/base64"
    "sync"
    "time"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)

// LocalLog formats the message and logs it with a specific prefix
func LocalLog(format string, v ...interface{}) {
	var logMessage string
	if len(v) == 0 {
		logMessage = format // No arguments, use the format string as-is
	} else {
		logMessage = fmt.Sprintf(format, v...)
	}
	log.Printf("[worker] %s", logMessage)
}

// Declare the global config variable
var cfg *config.Config

const MAX_CONCURRENT_TASKS = 10
const EMA_PARAM = 0.9

var semaphore = make(chan struct{}, MAX_CONCURRENT_TASKS)
var mu sync.Mutex
var numRunningTasks uint32 = 0
var numQueuedTasks uint32 = 0
var averageRunningTime float32 = 0.1

type ClientConstructor[T any] func(grpc.ClientConnInterface) T

func createGRPCClient[T any](address string, constructor ClientConstructor[T]) T {
    conn, err := grpc.Dial(address, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("failed to connect to %s: %v", address, err)
    }
    // defer conn.Close() // f u chatgpt
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

    LocalLog("worker server listening at %v", lis.Addr())
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
    rootPath := os.Getenv("PROJECT_ROOT")
    executeFile := filepath.Join(rootPath, "go", "cmd", "worker", "execute.py")
    cmd := exec.Command("/usr/bin/python3", executeFile)

    // Create a buffer to hold the serialized data
    inputBuffer := bytes.NewBuffer(nil)

    // Assume data is the serialized data you want to encode in base64
    fB64 := []byte(base64.StdEncoding.EncodeToString(f))
    argsB64 := []byte(base64.StdEncoding.EncodeToString(args))
    kwargsB64 := []byte(base64.StdEncoding.EncodeToString(kwargs))

    // Write the function, args, and kwargs to the buffer
    inputBuffer.Write(fB64)
    inputBuffer.WriteByte('\n')
    inputBuffer.Write(argsB64)
    inputBuffer.WriteByte('\n')
    inputBuffer.Write(kwargsB64)

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
    LocalLog("in Run() rn")
    mu.Lock()
    numQueuedTasks++
    mu.Unlock()

    semaphore <- struct{}{}

    mu.Lock()
    numQueuedTasks--
    numRunningTasks++
    mu.Unlock()

    defer func() {
        mu.Lock()
        numRunningTasks--
        mu.Unlock()
        <-semaphore
    }()

    start := time.Now()

    funcResponse, err := s.funcClient.FetchFunc(ctx, &pb.FetchRequest{Name: req.Name})
    if err != nil {
        return nil, err
    }

    output, err := executeFunction(funcResponse.SerializedFunc, req.Args, req.Kwargs)
    if err != nil {
        return nil, err
    }

    _, err = s.storeClient.Store(ctx, &pb.StoreRequest{Uid: req.Uid, ObjectBytes: output})
    if err != nil {
        return nil, err
    }

    runningTime := float32(time.Since(start).Seconds())
    mu.Lock()
    averageRunningTime = EMA_PARAM*averageRunningTime + (1-EMA_PARAM)*runningTime
    mu.Unlock()

    return &pb.StatusResponse{Success: true}, nil
}

func (s *workerServer) WorkerStatus(ctx context.Context, req *pb.StatusResponse) (*pb.WorkerStatusResponse, error) {
    mu.Lock()
    defer mu.Unlock()
    // log.Printf("num queued tasks is currently: %v", numQueuedTasks)
    return &pb.WorkerStatusResponse{
        NumRunningTasks:    numRunningTasks,
        NumQueuedTasks:     numQueuedTasks,
        AverageRunningTime: averageRunningTime,
    }, nil
}