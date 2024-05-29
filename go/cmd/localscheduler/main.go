package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
    "os"
    // "math"

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
)

var globalSchedulerClient pb.GlobalSchedulerClient
var localNodeID uint64
var cfg *config.Config

func main() {
	cfg = config.GetConfig()                                // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.LocalScheduler) // Prepare the network address
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer()
	
	

	workerAddress := fmt.Sprintf("localhost:%d", cfg.Ports.LocalWorkerStart)
	workerConn, _ := grpc.Dial(workerAddress, grpc.WithInsecure())
	workerClient := pb.NewWorkerClient(workerConn)

	globalSchedulerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GlobalScheduler, cfg.Ports.GlobalScheduler)
	scheduleConn, _ := grpc.Dial(globalSchedulerAddress, grpc.WithInsecure())
	globalSchedulerClient := pb.NewGlobalSchedulerClient(scheduleConn)

	gcsObjAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	gcsConn, _ := grpc.Dial(gcsObjAddress, grpc.WithInsecure())
	gcsObjClient := pb.NewGCSObjClient(gcsConn)
	nodeId, _ := strconv.Atoi(os.Getenv("NODE_ID"))
	pb.RegisterLocalSchedulerServer(s, &server{globalSchedulerClient: globalSchedulerClient, workerClient: workerClient, globalCtx: context.Background(), localNodeID: uint64(nodeId), gcsClient: gcsObjClient })

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

// server is used to implement your gRPC service.
type server struct {
	pb.UnimplementedLocalSchedulerServer
	globalSchedulerClient pb.GlobalSchedulerClient
	workerClient pb.WorkerClient
	gcsClient pb.GCSObjClient
	globalCtx context.Context
	localNodeID uint64
}

// Implement your service methods here.

func (s *server) Schedule(ctx context.Context, req *pb.ScheduleRequest) (*pb.ScheduleResponse, error) {
	var worker_id int
	worker_id, _ = strconv.Atoi(os.Getenv("NODE_ID"))
    uid := rand.Uint64()

	

	_, err := s.gcsClient.RegisterLineage(ctx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
	if scheduleLocally.NumRunningTasks < MAX_TASKS {
		// LocalLog("Just running locally")

		go func() {
            _, err := s.workerClient.Run(s.globalCtx, &pb.RunRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
            if err != nil {
                LocalLog("cannot contact worker %d: %v", worker_id, err)
            } else {
                // LocalLog("Just ran it!")
            }
        }()
		
	} else {
		// LocalLog("contacting global scheduler")
		go func() {
            _, err := s.globalSchedulerClient.Schedule(s.globalCtx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
            if err != nil {
                LocalLog("cannot contact global scheduler")
            } else {
				// LocalLog("Just ran it on global!")
			}
        }()
	}
	return &pb.ScheduleResponse{Uid: uid}, nil

}
