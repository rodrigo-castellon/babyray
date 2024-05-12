package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "math/rand"
    "fmt"
    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)


var globalSchedulerClient pb.GlobalSchedulerClient
var localNodeID uint64
var cfg *config.Config
func main() {
    cfg = config.GetConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.LocalScheduler) // Prepare the network address
    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    pb.RegisterLocalSchedulerServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }

    globalSchedulerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GlobalScheduler, cfg.Ports.GlobalScheduler)
    conn, _ := grpc.Dial(globalSchedulerAddress, grpc.WithInsecure())
    globalSchedulerClient = pb.NewGlobalSchedulerClient(conn)
    localNodeID = 0


}

// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedLocalSchedulerServer
}

// Implement your service methods here.


func (s *server) Schedule(ctx context.Context, req *pb.ScheduleRequest) (*pb.ScheduleResponse, error) {
    var worker_id int 
    // worker_id = check_resources()
    worker_id = -1
    uid := uint64(rand.Intn(100))
    if worker_id != -1 {
        workerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.Ourself, cfg.Ports.LocalWorkerStart + worker_id)
        conn, _ := grpc.Dial(workerAddress, grpc.WithInsecure())
        workerClient := pb.NewWorkerClient(conn)
        _, err := workerClient.Run(ctx, &pb.RunRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
        if err != nil {
            log.Printf("cannot contact worker %d", worker_id)
        }
    } else {
      
        _, err := globalSchedulerClient.Schedule(ctx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
        if err != nil {
            log.Printf("cannot contact global scheduler")
        }

    }
    return &pb.ScheduleResponse{Uid: uid}, nil 

}
