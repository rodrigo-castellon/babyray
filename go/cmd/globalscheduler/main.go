package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "math/rand"
    "math"
   // "bytes"

//    "os"
//    "time"
//    "os"
   "sync"
   "time"
    "fmt"
    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
    "github.com/rodrigo-castellon/babyray/customlog"
    "github.com/rodrigo-castellon/babyray/util"

    // "github.com/go-zookeeper/zk"
)
var cfg *config.Config
const LIVE_NODE_TIMEOUT time.Duration = 400 * time.Millisecond
const HEARTBEAT_WAIT = 100 * time.Millisecond
var mu sync.RWMutex

const MAX_CONCURRENT_TASKS = 10

// LocalLog formats the message and logs it with a specific prefix
func LocalLog(format string, v ...interface{}) {
	var logMessage string
	if len(v) == 0 {
		logMessage = format // No arguments, use the format string as-is
	} else {
		logMessage = fmt.Sprintf(format, v...)
	}
	log.Printf("[localscheduler] %s", logMessage)
}

func main() {
    customlog.Init()
    ctx := context.Background()
    cfg = config.GetConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GlobalScheduler) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer(util.GetServerOptions()...)
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, util.GetDialOptions()...)
    serverInstance := &server{gcsClient: pb.NewGCSObjClient(conn), status: make(map[uint64]HeartbeatEntry)}
    pb.RegisterGlobalSchedulerServer(s, serverInstance)
    defer conn.Close()
    log.Printf("server listening at %v", lis.Addr())
    go serverInstance.LiveNodesHeartbeat(ctx)
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
    //log.Printf("heartbeat from %v", req.NodeId)
    mu.Lock()
    numQueuedTasks := req.QueuedTasks
    if (req.RunningTasks == MAX_CONCURRENT_TASKS) {
        numQueuedTasks = numQueuedTasks + 1 // also need to wait for something currently running to finish
    }

    s.status[req.NodeId] = HeartbeatEntry{timeReceived: time.Now(), numRunningTasks: req.RunningTasks, numQueuedTasks: numQueuedTasks, avgRunningTime: req.AvgRunningTime, avgBandwidth: req.AvgBandwidth}
    mu.Unlock()
    return &pb.StatusResponse{Success: true}, nil
}

func (s *server) LiveNodesHeartbeat(ctx context.Context) (error) {
    ctx = context.Background()
    for {
        s.SendLiveNodes(ctx)
        time.Sleep(HEARTBEAT_WAIT)
    }
    return nil

}

func(s *server) SendLiveNodes(ctx context.Context) (error) {
    liveNodes := make(map[uint64]bool)
    mu.RLock()
    for uid, heartbeat := range s.status {
        liveNodes[uid] =  time.Since(heartbeat.timeReceived) < LIVE_NODE_TIMEOUT
    }
    mu.RUnlock()
    // LocalLog("sending RegisterLiveNodes() call now")
    // go func() {
    //     if _, err := s.gcsClient.RegisterLiveNodes(ctx, &pb.LiveNodesRequest{LiveNodes: liveNodes}); err != nil {
    //         LocalLog("Error registering live nodes: %v", err)
    //     }
    // }()
    s.gcsClient.RegisterLiveNodes(ctx, &pb.LiveNodesRequest{LiveNodes: liveNodes})
    return nil
}

func (s *server) Schedule(ctx context.Context , req *pb.GlobalScheduleRequest ) (*pb.StatusResponse, error) {
    localityFlag := req.LocalityFlag
    // localityFlag := false
    // if os.Getenv("LOCALITY_AWARE") == "true" {
    //     localityFlag = true
    // }

    log.Printf("locality aware? it's: %v", localityFlag)

    log.Printf("THE REQ UIDS ARE = %v", req.Uids)

    // gives us back the node id of the worker
    node_id := req.NodeId
    if (req.NodeId == 0) {
        node_id = getBestWorker(ctx, s, localityFlag, req.Uids)
    }

    log.Printf("best worker was node_id = %v", node_id)
    workerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, node_id, cfg.Ports.LocalWorkerStart)

    conn, err := grpc.Dial(workerAddress, util.GetDialOptions()...)
    if err != nil {
        log.Printf("failed to connect to %s: %v", workerAddress, err)
        return nil, err
    }
    defer conn.Close()

    workerClient := pb.NewWorkerClient(conn)
    LocalLog("Contacted the worker")
    // if req.NewObject {
    LocalLog("registering this as a generating node now.")
    s.gcsClient.RegisterGenerating(ctx, &pb.GeneratingRequest{Uid: req.Uid, NodeId: node_id})
    // }
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
    if (localityFlag && len(uids) > 0) {
        log.Printf("LOCALITY FLAG IS ON!")
        log.Printf("ASKING THE GCS FOR THESE OBJECTS: %v", uids)
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
            log.Printf("the locations resp val = %v", val)
            for _, loc := range locs {
                locationToBytes[uint64(loc)] += val.Bytes
                total += val.Bytes
            }
        }

        // Collect keys from the map
        keys := make([]uint64, 0, len(s.status))
        for id := range s.status {
            keys = append(keys, id)
        }

        // Shuffle the keys
        rand.Shuffle(len(keys), func(i, j int) {
            keys[i], keys[j] = keys[j], keys[i]
        })

        for _, loc := range keys {

        // for loc, bytes := range locationToBytes {
            bytes := locationToBytes[loc]
            // skip dead nodes
            if heartbeat, _ := s.status[loc]; !(time.Since(heartbeat.timeReceived) < LIVE_NODE_TIMEOUT) {
                continue
            }
            mu.RLock()
            queueingTime := float32(s.status[loc].numQueuedTasks) * s.status[loc].avgRunningTime
            transferTime := float32(total - bytes) / s.status[loc].avgBandwidth
            mu.RUnlock()

            log.Printf("worker = %v; queueing and transfer = %v, %v", loc, queueingTime, transferTime)

            waitingTime := queueingTime + transferTime
            if waitingTime < minTime {
                minTime = waitingTime
                minId = loc
                foundBest = true
            }
        }
    } else {
        log.Printf("doing the statuses rn")
        // TODO: make iteration order random for maximum fairness
        mu.RLock()

        // Collect keys from the map
        keys := make([]uint64, 0, len(s.status))
        for id := range s.status {
            keys = append(keys, id)
        }

        // Shuffle the keys
        rand.Shuffle(len(keys), func(i, j int) {
            keys[i], keys[j] = keys[j], keys[i]
        })

        for _, id := range keys {
            heartbeat := s.status[id]
            if !(time.Since(heartbeat.timeReceived) < LIVE_NODE_TIMEOUT) {
                continue
            }

            log.Printf("worker = %v; queued time = %v", id, float32(heartbeat.numQueuedTasks)*heartbeat.avgRunningTime)
            if float32(heartbeat.numQueuedTasks)*heartbeat.avgRunningTime < minTime {
                minId = id
                minTime = float32(heartbeat.numQueuedTasks) * heartbeat.avgRunningTime
                foundBest = true
            }
        }
        mu.RUnlock()

        // mu.RLock()
        // for id, heartbeat := range s.status {
        //     log.Printf("worker = %v; queued time = %v", id, float32(heartbeat.numQueuedTasks) * heartbeat.avgRunningTime)
        //     if float32(heartbeat.numQueuedTasks) * heartbeat.avgRunningTime < minTime {
        //         minId = id
        //         minTime = float32(heartbeat.numQueuedTasks) * heartbeat.avgRunningTime
        //         foundBest = true
        //     }
        // }
        // mu.RUnlock()
    }

    if !foundBest {
        log.Fatalf("global scheduler failed to pick a worker")
    }
    return minId
}


