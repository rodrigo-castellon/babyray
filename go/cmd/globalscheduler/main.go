package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "math/rand"
    "math"
    "bytes"
    "fmt"
    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)
var cfg *config.Config
func main() {
    cfg = config.LoadConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GlobalScheduler) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, grpc.WithInsecure())
    pb.RegisterGlobalSchedulerServer(s, &server{gcsClient: pb.NewGCSObjClient(conn)})
    defer conn.Close()
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

type HeartbeatEntry struct {
    numRunningTasks uint64
    numQueuedTasks uint64
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
    s.status[req.NodeId] = HeartbeatEntry{numRunningTasks: req.RunningTasks, numQueuedTasks: req.QueuedTasks, avgRunningTime: req.AvgRunningTime, avgBandwidth: req.AvgBandwidth}
    return &pb.StatusResponse{Success: true}, nil
}
func (s *server) Schedule(ctx context.Context , req *pb.GlobalScheduleRequest ) (*pb.StatusResponse, error) {
    localityFlag := false //Os.Getenv("locality_aware")
    worker_id := getBestWorker(s, localityFlag, req.Args)
    workerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, worker_id, cfg.Ports.LocalWorkerStart)

    log.Printf("the worker address is %v", workerAddress)
    conn, err := grpc.Dial(workerAddress, grpc.WithInsecure())
    if err != nil {
        log.Printf("failed to connect to %s: %v", workerAddress, err)
        return nil, err
    }
    defer conn.Close()

    workerClient := pb.NewWorkerClient(conn)

	uid := uint64(rand.Intn(100))
    output_result, err := workerClient.Run(ctx, &pb.RunRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
    if err != nil || !output_result.Success {
        log.Fatalf(fmt.Sprintf("global scheduler failed to contact worker %d. Err: %v, Response code: %d", worker_id, err, output_result.ErrorCode))
    } 
    return &pb.StatusResponse{Success: true}, nil


}

func getBestWorker(s *server, localityFlag bool, args []bytes) (uint32) {
    minId := -1; 
    minTime := math.MaxInt
    if localityFlag {
        locationsResp, err := s.gcsClient.GetObjectLocations(&pb.ObjectLocationsRequest{Args: args})
        if err != nil {
            log.Fatalf("Failed to ask gcs for object locations: %v", err)
        }
       
  
        locationToBytes = make(map[uint64]uint32)
        total := 0
        for _ , loc := range locationsResp.Locations {
            locationToBytes[loc[0]] += loc[1] 
            total += loc[1]
        }
        
        for loc, bytes := range locationToBytes {
            queueingTime := s.status[loc][numQueuedTasks] * s.status[loc][AvgRunningTime]
            transferTime := (total - bytes) * s.status[loc][AvgBandwidth]
            waitingTIme = queueingTime + transferTime
            if waitingTime < min_waiting_time {
                minTime = waitingTime
                minId = loc
            }
        }

    } else {
  
        for id, times := range s.status {
            if times[0] + times[1] < minTime {
                minId = id
                minTime = times[0] + times[1]
            }
        }
        
    }
    if minId == -1 {
        log.Fatalf("global scheduler failed to pick a worker")
    }
    return minId
}


