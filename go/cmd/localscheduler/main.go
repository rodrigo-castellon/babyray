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
	"time"

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
)


var cfg *config.Config
const HEARTBEAT_WAIT int32 = 1 
const MAX_TASKS uint32 = 10

func main() {
	cfg = config.GetConfig()                                // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.LocalScheduler) // Prepare the network address
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer()

	globalSchedulerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GlobalScheduler, cfg.Ports.GlobalScheduler)
	conn, _ := grpc.Dial(globalSchedulerAddress, grpc.WithInsecure())
	globalSchedulerClient := pb.NewGlobalSchedulerClient(conn)
	nodeId, _ := strconv.Atoi(os.Getenv("NODE_ID"))

	pb.RegisterLocalSchedulerServer(s, &server{globalSchedulerClient: globalSchedulerClient, localNodeID: uint64(nodeId)})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	ctx = context.Background()
	go SendHeartbeats(ctx, globalSchedulerClient, nodeId)



	

}

// server is used to implement your gRPC service.
type server struct {
	pb.UnimplementedLocalSchedulerServer
	globalSchedulerClient pb.GlobalSchedulerClient
	localNodeID uint64

}

// Implement your service methods here.

func (s *server) Schedule(ctx context.Context, req *pb.ScheduleRequest) (*pb.ScheduleResponse, error) {
	var worker_id int
	// worker_id = check_resources()
	worker_id, _ = strconv.Atoi(os.Getenv("NODE_ID"))

	workerAddress := fmt.Sprintf("localhost:%d", cfg.Ports.LocalWorkerStart)
        log.Printf("the worker address is %v", workerAddress)
		conn, err := grpc.Dial(workerAddress, grpc.WithInsecure())
        if err != nil {
            log.Printf("failed to connect to %s: %v", workerAddress, err)
            return nil, err
        }
        defer conn.Close()

		workerClient := pb.NewWorkerClient(conn)
	
	
	uid := uint64(rand.Intn(100))
	scheduleLocally := workerClient.WorkerStatus(&pb.StatusResponse{}).NumRunningTasks < MAX_TASKS

	if scheduleLocally {
		_, err = workerClient.Run(ctx, &pb.RunRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
		if err != nil {
            log.Printf("cannot contact worker %d: %v", worker_id, err)
            return nil, err
		}
		
	} else {
		_, err := s.globalSchedulerClient.Schedule(ctx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
		if err != nil {
            log.Printf("cannot contact global scheduler")
            return nil, err
		}

	}
	return &pb.ScheduleResponse{Uid: uid}, nil

}

func SendHeartbeats(ctx context.Context, globalSchedulerClient pb.GlobalSchedulerClient, nodeId uint64 ) {
	worker_id, _ := strconv.Atoi(os.Getenv("NODE_ID"))
	workerAddress := fmt.Sprintf("localhost:%d", cfg.Ports.LocalWorkerStart)
        log.Printf("the worker address is %v", workerAddress)
		workerConn, err := grpc.Dial(workerAddress, grpc.WithInsecure())
        if err != nil {
            log.Fatalf("failed to connect to %s: %v", workerAddress, err)
            
        }
    defer workerConn.Close()

	lobsAddress := fmt.Sprintf("localhost:%d", cfg.Ports.LocalObjectStore)
	log.Printf("the worker address is %v", lobsAddress)
	lobsConn, err := grpc.Dial(lobsAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", lobsAddress, err)
		
	}
    defer lobsConn.Close()


	workerClient := pb.NewWorkerClient(workerConn)
	lobsClient := pb.NewLocalObjStoreClient(lobsConn)
	for {
		status, _ := workerClient.WorkerStatus(ctx, &pb.StatusResponse{})
		numRunningTasks := status.NumRunningTasks
		numQueuedTasks  := status.NumQueuedTasks
		avgRunningTime  := status.AvgRunningTime
		avgBandwidth, _    := lobsClient.AvgBandwidth(ctx, &pb.StatusResponse{})
		globalSchedulerClient.Heartbeat(ctx, &pb.HeartbeatRequest{
			RunningTasks: numRunningTasks, 
			QueuedTasks: numQueuedTasks, 
		    AvgRunningTime: avgRunningTime, 
			AvgBandwidth: avgBandwidth.AvgBandwidth, 
			NodeId: nodeId })
	    time.Sleep(HEARTBEAT_WAIT * time.Second)
	}
}