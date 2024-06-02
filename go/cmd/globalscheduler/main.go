package main

import (
    "context"
    "log"
    "net"
    "strconv"
    // "math/rand"
    "math"
   // "bytes"

   "os"
   "sync"
   "time"
    "fmt"
    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)
var cfg *config.Config
const LIVE_NODE_TIMEOUT time.Duration = 400 * time.Millisecond
const HEARTBEAT_WAIT = 100 * time.Millisecond
var mu sync.RWMutex

func main() {
    ctx := context.Background()
    //cfg = config.GetConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GlobalScheduler) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, grpc.WithInsecure())
    server := &server{gcsClient: pb.NewGCSObjClient(conn), status: make(map[uint64]HeartbeatEntry)}
    pb.RegisterGlobalSchedulerServer(s, server)
    defer conn.Close()
    log.Printf("server listening at %v", lis.Addr())
    go server.SendLiveNodes(ctx)
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

type HeartbeatEntry struct {
    timeReceived    time.Time
    numRunningTasks uint32
    numQueuedTasks uint32
    avgRunningTime float32
    avgBandwidth float32
}
// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedGlobalSchedulerServer
   gcsClient ObjClient
   status map[uint64]HeartbeatEntry
}

type ObjClient interface {
    RegisterLiveNodes(ctx context.Context, req *pb.LiveNodesRequest, opts ...grpc.CallOption) (*pb.StatusResponse, error) 
    RequestLocation(ctx context.Context, req *pb.RequestLocationRequest, opts ...grpc.CallOption) (*pb.RequestLocationResponse, error)
    RegisterGenerating(ctx context.Context, req *pb.GeneratingRequest,  opts ...grpc.CallOption) (*pb.StatusResponse, error)
    GetObjectLocations(ctx context.Context, req *pb.ObjectLocationsRequest,  opts ...grpc.CallOption) (*pb.ObjectLocationsResponse, error) 
}


func (s *server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest ) (*pb.StatusResponse, error) {
    log.Printf("heartbeat from %v", req.NodeId)
    mu.Lock()
    s.status[req.NodeId] = HeartbeatEntry{timeReceived: time.Now(), numRunningTasks: req.RunningTasks, numQueuedTasks: req.QueuedTasks, avgRunningTime: req.AvgRunningTime, avgBandwidth: req.AvgBandwidth}
    mu.Unlock()
    return &pb.StatusResponse{Success: true}, nil
}

func (s *server) LiveNodesHeartbeat(ctx context.Context) (error) {
    
    for {
        s.SendLiveNodes(ctx)
        time.Sleep(HEARTBEAT_WAIT)
    }
    return nil

}

func(s *server) SendLiveNodes(ctx context.Context) (error) {
    liveNodes := make(map[uint64]bool)
    for uid, heartbeat := range s.status {
        timeSince := time.Since(heartbeat.timeReceived)

        log.Printf("%v < %v = %b", timeSince, LIVE_NODE_TIMEOUT,  timeSince < LIVE_NODE_TIMEOUT)
        liveNodes[uid] =  timeSince < LIVE_NODE_TIMEOUT
        if _, val := liveNodes[uid]; val {
            log.Printf("%v sent as live", uid)
        } else {
            log.Printf("%v sent as dead", uid)
        }
    }
    s.gcsClient.RegisterLiveNodes(ctx, &pb.LiveNodesRequest{LiveNodes: liveNodes})
    return nil
}

func (s *server) Schedule(ctx context.Context , req *pb.GlobalScheduleRequest ) (*pb.StatusResponse, error) {
    localityFlag := false
    if os.Getenv("LOCALITY_AWARE") == "true" {
        localityFlag = true
    }

    // gives us back the node id of the worker
    node_id := getBestWorker(ctx, s, localityFlag, req.Uids)
    workerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, node_id, cfg.Ports.LocalWorkerStart)

    conn, err := grpc.Dial(workerAddress, grpc.WithInsecure())
    if err != nil {
        log.Printf("failed to connect to %s: %v", workerAddress, err)
        return nil, err
    }
    defer conn.Close()

    workerClient := pb.NewWorkerClient(conn)

    if req.NewObject {
        s.gcsClient.RegisterGenerating(ctx, &pb.GeneratingRequest{Uid: req.Uid, NodeId: node_id})
    }
    output_result, err := workerClient.Run(ctx, &pb.RunRequest{Uid: req.Uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
    if err != nil || !output_result.Success {
        log.Fatalf(fmt.Sprintf("global scheduler failed to contact node %d. Err: %v", node_id, err))
    }
    return &pb.StatusResponse{Success: true}, nil
}

func getBestWorker(ctx context.Context, s *server, localityFlag bool, uids []uint64) (uint64) {
    var minId uint64
    minId = 0
    var minTime float32
    var foundBest bool

    minTime = math.MaxFloat32
    if localityFlag {
        locationsResp, err := s.gcsClient.GetObjectLocations(ctx, &pb.ObjectLocationsRequest{Args: uids})
        if err != nil {
            log.Fatalf("Failed to ask gcs for object locations: %v", err)
        }

        locationToBytes := make(map[uint64]uint64)

        // init every node with 0
        for id, _ := range s.status {
            locationToBytes[id] = 0
        }

        var total uint64
        total = 0
        for _, val := range locationsResp.Locations {
            locs := val.Locations
            for _, loc := range locs {
                locationToBytes[uint64(loc)] += val.Bytes
                total += val.Bytes
            }
        }

        for loc, bytes := range locationToBytes {
            mu.RLock()
            queueingTime := float32(s.status[loc].numQueuedTasks) * s.status[loc].avgRunningTime
            transferTime := float32(total - bytes) * s.status[loc].avgBandwidth
            mu.RUnlock()

            waitingTime := queueingTime + transferTime
            if waitingTime < minTime {
                minTime = waitingTime
                minId = loc
                foundBest = true
            }
        }
    } else {
        // TODO: make iteration order random for maximum fairness
        mu.RLock()
        for id, heartbeat := range s.status {
            if float32(heartbeat.numQueuedTasks) * heartbeat.avgRunningTime < minTime {
                minId = id
                minTime = float32(heartbeat.numQueuedTasks) * heartbeat.avgRunningTime
                foundBest = true
            }
        }
        mu.RUnlock()
    }

    if !foundBest {
        log.Fatalf("global scheduler failed to pick a worker")
    }
    return minId
}


