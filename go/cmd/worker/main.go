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
    "runtime"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
    "github.com/rodrigo-castellon/babyray/customlog"
    "github.com/rodrigo-castellon/babyray/util"
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
    conn, err := grpc.Dial(address, util.GetDialOptions()...)
    if err != nil {
        log.Fatalf("failed to connect to %s: %v", address, err)
    }
    // defer conn.Close() // f u chatgpt
    return constructor(conn)
}

func main() {
    customlog.Init()
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

    s := grpc.NewServer(util.GetServerOptions()...)
    pb.RegisterWorkerServer(s, &workerServer{
        funcClient: funcClient,
        storeClient: storeClient,
        alive: true,
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
    alive bool
}

type FuncClient interface {
    FetchFunc(ctx context.Context, in *pb.FetchRequest, opts ...grpc.CallOption) (*pb.FetchResponse, error)
}

type StoreClient interface {
    Store(ctx context.Context, in *pb.StoreRequest, opts ...grpc.CallOption) (*pb.StatusResponse, error)
}

func executeFunction(f []byte, args []byte, kwargs []byte) ([]byte, error) {
    startTime := time.Now()

    // Prepare the command to run the Python script
    rootPath := os.Getenv("PROJECT_ROOT")
    executeFile := filepath.Join(rootPath, "go", "cmd", "worker", "execute.py")
    cmd := exec.Command("/usr/bin/python3", executeFile)

    LocalLog("Time to prepare command: %v\n", time.Since(startTime))

    // Create a buffer to hold the serialized data
    inputBuffer := bytes.NewBuffer(nil)

    // Assume data is the serialized data you want to encode in base64
    encodeStartTime := time.Now()
    fB64 := []byte(base64.StdEncoding.EncodeToString(f))
    argsB64 := []byte(base64.StdEncoding.EncodeToString(args))
    kwargsB64 := []byte(base64.StdEncoding.EncodeToString(kwargs))

    LocalLog("Time to encode inputs: %v\n", time.Since(encodeStartTime))

    // Write the function, args, and kwargs to the buffer
    bufferWriteStartTime := time.Now()
    inputBuffer.Write(fB64)
    inputBuffer.WriteByte('\n')
    inputBuffer.Write(argsB64)
    inputBuffer.WriteByte('\n')
    inputBuffer.Write(kwargsB64)

    LocalLog("Time to write to buffer: %v\n", time.Since(bufferWriteStartTime))

    // Set the stdin to our input buffer
    cmd.Stdin = inputBuffer

    // Capture the output
    LocalLog("Running the function here yo")
    cmdExecStartTime := time.Now()
    output, err := cmd.Output()
    if err != nil {
        log.Fatalf("Error executing function: %v", err)
    }

    LocalLog("Time to execute command: %v\n", time.Since(cmdExecStartTime))

    // Decode the Base64 output to get the original pickled data
    // LocalLog("the output from this was: %s", string(output)[:min(len(string(output)), 282)])
    // LocalLog("the length of the overall string was: %v", len(string(output)))
    decodeStartTime := time.Now()
    data, err := base64.StdEncoding.DecodeString(string(output))
    if err != nil {
        log.Fatalf("Error decoding Base64: %v", err)
    }
    LocalLog("Time to decode output: %v\n", time.Since(decodeStartTime))

    totalTime := time.Since(startTime)
    log.Printf("Total execution time: %v\n", totalTime)

    // Return the output from the Python script
    return data, nil
}

// run executes the function by fetching it, running it, and storing the result.
func (s *workerServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.StatusResponse, error) {
    LocalLog("in Run() rn")
    defer LocalLog("Finished running")
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

    LocalLog("....fetching?")
    funcResponse, err := s.funcClient.FetchFunc(ctx, &pb.FetchRequest{Name: req.Name})
    LocalLog("the err was %v", err)
    if err != nil {
        LocalLog("Failed to hit func table: %v", err)
        return nil, err
    }

    LocalLog("gonna execute now")
    output, err := executeFunction(funcResponse.SerializedFunc, req.Args, req.Kwargs)
    LocalLog("Executed!")
    if err != nil {
        LocalLog("failed to exec func %v", err)
        return nil, err
    }

    LocalLog("finished executing function")

    LocalLog("s.alive is %v", s.alive)

    if (!s.alive) {
        LocalLog("Worker is not alive, exiting...")
        runtime.Goexit()
    }

    _, err = s.storeClient.Store(ctx, &pb.StoreRequest{Uid: req.Uid, ObjectBytes: output})
    if err != nil {
        LocalLog("failed to hit gcs %v", err)
        return nil, err
    }

    runningTime := float32(time.Since(start).Seconds())
    mu.Lock()
    averageRunningTime = EMA_PARAM*averageRunningTime + (1-EMA_PARAM)*runningTime
    mu.Unlock()

    return &pb.StatusResponse{Success: true}, nil
}

func (s *workerServer) KillServer(ctx context.Context, req *pb.StatusResponse) (*pb.StatusResponse, error) {
	LocalLog("GOT KILLED!")
	s.alive = false
    LocalLog("s.alive is now: %v", s.alive)
	return &pb.StatusResponse{Success: true}, nil
}

func (s *workerServer) ReviveServer(ctx context.Context, req *pb.StatusResponse) (*pb.StatusResponse, error) {
	s.alive = true
	return &pb.StatusResponse{Success: true}, nil
}

func (s *workerServer) WorkerStatus(ctx context.Context, req *pb.StatusResponse) (*pb.WorkerStatusResponse, error) {
    mu.Lock()
    defer mu.Unlock()
    //log.Printf("num queued tasks is currently: %v", numQueuedTasks)
    return &pb.WorkerStatusResponse{
        NumRunningTasks:    numRunningTasks,
        NumQueuedTasks:     numQueuedTasks,
        AverageRunningTime: averageRunningTime,
    }, nil
}