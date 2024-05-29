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
	pb.RegisterLocalSchedulerServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

	globalSchedulerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GlobalScheduler, cfg.Ports.GlobalScheduler)
	scheduleConn, _ := grpc.Dial(globalSchedulerAddress, grpc.WithInsecure())
	globalSchedulerClient := pb.NewGlobalSchedulerClient(scheduleConn)

	gcsObjAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	gcsConn, _ := grpc.Dial(gcsObjAddress, grpc.WithInsecure())
	gcsObjClient := pb.NewGCSObjClient(gcsConn)

	nodeId, _ := strconv.Atoi(os.Getenv("NODE_ID"))

	pb.RegisterLocalSchedulerServer(s, &server{globalSchedulerClient: globalSchedulerClient, workerClient: workerClient, globalCtx: context.Background(), localNodeID: uint64(nodeId), gcsClient: gcsObjClient })

	ctx := context.Background()
	go SendHeartbeats(ctx, globalSchedulerClient, uint64(nodeId))

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
	// worker_id = check_resources()
	worker_id, _ = strconv.Atoi(os.Getenv("NODE_ID"))
    uid := rand.Uint64()

	scheduleLocally, _ := s.workerClient.WorkerStatus(ctx, &pb.StatusResponse{})

	_, err := s.gcsClient.RegisterLineage(ctx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
	if err != nil {
		LocalLoc("failed to register lineage %v", err)
	}
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

		_, err := globalSchedulerClient.Schedule(ctx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs})
		if err != nil {
            log.Printf("cannot contact global scheduler")
            return nil, err
		}

	}
	return &pb.ScheduleResponse{Uid: uid}, nil

}

func SendHeartbeats(ctx context.Context, globalSchedulerClient pb.GlobalSchedulerClient, nodeId uint64 ) {
	workerAddress := fmt.Sprintf("localhost:%d", cfg.Ports.LocalWorkerStart)
	workerConn, err := grpc.Dial(workerAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", workerAddress, err)
	}
    defer workerConn.Close()

	lobsAddress := fmt.Sprintf("localhost:%d", cfg.Ports.LocalObjectStore)
	lobsConn, err := grpc.Dial(lobsAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to %s: %v", lobsAddress, err)
		
	}
    defer lobsConn.Close()


	workerClient := pb.NewWorkerClient(workerConn)
	lobsClient := pb.NewLocalObjStoreClient(lobsConn)
	for {
		backoff := 1
		var status *pb.WorkerStatusResponse

		for {
			status, err = workerClient.WorkerStatus(ctx, &pb.StatusResponse{})
			if err != nil {
				LocalLog("got error from WorkerStatus(): %v", err)
				LocalLog("retrying in %v seconds", backoff)
				time.Sleep(time.Duration(backoff) * time.Second)
				backoff *= 2
				if backoff > 32 {
					backoff = 32 // Cap the backoff to 32 seconds
				}
				continue
			}
			break
		}

		numRunningTasks := status.NumRunningTasks
		numQueuedTasks  := status.NumQueuedTasks
		avgRunningTime  := status.AverageRunningTime
		var avgBandwidth *pb.BandwidthResponse
		backoff = 1
		for {
			avgBandwidth, err  = lobsClient.AvgBandwidth(ctx, &pb.StatusResponse{})
			if err != nil {
				LocalLog("got error from WorkerStatus(): %v", err)
				LocalLog("retrying in %v seconds", backoff)
				time.Sleep(time.Duration(backoff) * time.Second)
				backoff *= 2
				if backoff > 32 {
					backoff = 32 // Cap the backoff to 32 seconds
				}
				continue
			}
			break
		}

		heartbeatRequest := &pb.HeartbeatRequest{
			RunningTasks: numRunningTasks, 
			QueuedTasks: numQueuedTasks, 
		    AvgRunningTime: avgRunningTime, 
			AvgBandwidth: avgBandwidth.AvgBandwidth, 
			NodeId: nodeId }

		// LocalLog("HeartbeatRequest: RunningTasks=%d, QueuedTasks=%d, AvgRunningTime=%.2f, AvgBandwidth=%.2f, NodeId=%d",
		// 	heartbeatRequest.RunningTasks,
		// 	heartbeatRequest.QueuedTasks,
		// 	heartbeatRequest.AvgRunningTime,
		// 	heartbeatRequest.AvgBandwidth,
		// 	heartbeatRequest.NodeId)

		globalSchedulerClient.Heartbeat(ctx, heartbeatRequest)
	    time.Sleep(HEARTBEAT_WAIT)
	}
}
