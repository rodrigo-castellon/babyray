package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "math/rand"
    "math"
   // "bytes"
   "time"
//    "os"
   "sync"
    "fmt"
    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
    "github.com/rodrigo-castellon/babyray/customlog"
    "github.com/rodrigo-castellon/babyray/util"

    // "github.com/go-zookeeper/zk"
)
var cfg *config.Config

var mu sync.RWMutex

// func connectToZookeeper(servers []string) (*zk.Conn, error) {
//     var conn *zk.Conn
//     var err error
//     for i := 0; i < 10; i++ { // retry 10 times
//         conn, _, err = zk.Connect(servers, time.Second*10)
//         if err == nil {
//             return conn, nil
//         }
//         log.Printf("Failed to connect to Zookeeper, retrying in 5 seconds... (attempt %d/10)", i+1)
//         time.Sleep(5 * time.Second)
//     }
//     return nil, err
// }

func main() {
    customlog.Init()
    cfg = config.LoadConfig() // Load configuration

    // connect to zookeeper
    // servers := []string{"zookeeper:2181"}
    // // zkConn, _, err := zk.Connect(servers, time.Second)
    // zkConn, err := connectToZookeeper(servers)
    // if err != nil {
    //     log.Fatalf("Unable to connect to Zookeeper: %v", err)
    // }

    // defer zkConn.Close()

    // path := "/services/globalsched"
    // data := []byte("node1")

    // _, err = zkConn.Create(path, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
    // if err != nil {
    //     log.Fatalf("Unable to create znode: %v", err)
    // }

    // log.Println("Service registered with Zookeeper")

    address := ":" + strconv.Itoa(cfg.Ports.GlobalScheduler) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer(util.GetServerOptions()...)
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, util.GetDialOptions()...)
    pb.RegisterGlobalSchedulerServer(s, &server{gcsClient: pb.NewGCSObjClient(conn), status: make(map[uint64]HeartbeatEntry)})
    defer conn.Close()
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

type HeartbeatEntry struct {
    numRunningTasks uint32
    numQueuedTasks uint32
    avgRunningTime float32
    avgBandwidth float32
}
// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedGlobalSchedulerServer
   gcsClient pb.GCSObjClient
   status map[uint64]HeartbeatEntry
}


func (s *server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest ) (*pb.StatusResponse, error) {
    // log.Printf("heartbeat from %v", req.NodeId)
    mu.Lock()
    numQueuedTasks := req.QueuedTasks
    if (req.RunningTasks == 10) {
        numQueuedTasks = numQueuedTasks + 1 // also need to wait for something currently running to finish
    }

    s.status[req.NodeId] = HeartbeatEntry{numRunningTasks: req.RunningTasks, numQueuedTasks: numQueuedTasks, avgRunningTime: req.AvgRunningTime, avgBandwidth: req.AvgBandwidth}
    mu.Unlock()
    return &pb.StatusResponse{Success: true}, nil
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

    log.Printf("gonna call Run() now")

    // Example of creating a context with a timeout
    // For some reason this custom context is necessary... ¯\_(ツ)_/¯
    customCtx, cancel := context.WithTimeout(context.Background(), time.Hour)
    defer cancel()

    output_result, err := workerClient.Run(customCtx, &pb.RunRequest{Uid: req.Uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
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

        for loc, bytes := range locationToBytes {
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


