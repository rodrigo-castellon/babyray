package main

import (
    // "context"
    "log"
    "net"
    "strconv"
    "math/rand"
    
    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)

func main() {
    cfg := config.LoadConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GlobalScheduler) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, grpc.WithInsecure())
    pb.RegisterGlobalSchedulerServer(s, &server{globalSchedulerClient: pb.NewGCSClient(conn)})
    defer conn.close()
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

type HeartbeatEntry struct {
    numRunningTasks uint32
    numQueuedTasks uint32
    avgRunningTime float
    avgBandwidth float

}
// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedGlobalSchedulerServer
   gcsClient pb.GCSFuncClient
   status map[uint32]heartbeat
}


func (s *server) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest ) (*pb.StatusResponse, error) {
    s.status[req.NodeId] = HeartbeatEntry{numRunningTasks: req.RunningTasks, numQueuedTasks: req.QueuedTasks, avgRunningTime: req.AvgRunningTime, avgBandwidth: req.AvgBandwidth}
}
func (s *server) Schedule(ctx context.Context , req *pb.GlobalScheduleRequest ) (*pb.StatusResponse, error) {
    localityFlag := false //Os.Getenv("locality_aware")
    worker_id := getBestWorker(s, localityFlag, args)
    workerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, worker_id, cfg.Ports.Worker)

    log.Printf("the worker address is %v", workerAddress)
    conn, err := grpc.Dial(workerAddress, grpc.WithInsecure())
    if err != nil {
        log.Printf("failed to connect to %s: %v", workerAddress, err)
        return nil, err
    }
    defer conn.Close()

    workerClient := pb.NewWorkerClient(conn)

	uid := uint64(rand.Intn(100))
    output_result, err = workerClient.Run(&pb.RunRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
    if err != nil || !output_result.Success {
        log.Fatalf(fmt.Sprintf("global scheduler failed to contact worker %d. Err: %v, Response code: %d", worker_id, err, output_result.errorCode))
    } 
    return uid


}

func getBestWorker(s *server, localityFlag bool, args []uint64) (uint32) {
    minId = -1; 
    if localityFlag {
        locationsResp, err = s.gcsClient.GetObjectLocations(&pb.ObjectLocationRequest{Args: args})
        if err != nil {
            log.Errorf("Failed to ask gcs for object locations: %v", err)
        }
        minTime = math.MaxInt
        minId = -1
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
        minTime := math.MaxInt
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


