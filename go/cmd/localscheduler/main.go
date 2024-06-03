package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
    "os"
	"time"

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
)




var cfg *config.Config
const HEARTBEAT_WAIT = 100 * time.Millisecond
const MAX_TASKS uint32 = 10

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
	cfg = config.GetConfig()                                // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.LocalScheduler) // Prepare the network address
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer()

	// set up worker connection early
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
	server := &server{globalSchedulerClient: globalSchedulerClient, workerClient: workerClient, globalCtx: context.Background(), localNodeID: uint64(nodeId), gcsClient: gcsObjClient, alive: true }
	pb.RegisterLocalSchedulerServer(s, server)
	ctx := context.Background()
	go server.SendHeartbeats(ctx, globalSchedulerClient, uint64(nodeId))

	LocalLog("localsched server listening at %v", lis.Addr())
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
	alive bool
}

// Implement your service methods here.

func (s *server) Schedule(ctx context.Context, req *pb.ScheduleRequest) (*pb.ScheduleResponse, error) {
	var worker_id int
	worker_id, _ = strconv.Atoi(os.Getenv("NODE_ID"))
    uid := rand.Uint64()

	scheduleLocally, err := s.workerClient.WorkerStatus(ctx, &pb.StatusResponse{})
	if err != nil {
		LocalLog("got error from WorkerStatus() in Sched: %v", err)
	}
	_ , err = s.gcsClient.RegisterLineage(ctx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs, Uids: req.Uids})
	if err != nil {
		LocalLog("cant hit gcs: %v", err)
	}

	// custom behavior if the client itself specifies where we should send this
	// computation
	if (req.NodeId != 0) {
		LocalLog("Doing something special with req.NodeId = %v", req.NodeId)
		go func() {
            _, err := s.globalSchedulerClient.Schedule(s.globalCtx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs, Uids: req.Uids, NodeId: req.NodeId})
            if err != nil {
                LocalLog("cannot contact global scheduler")
            } else {
				// LocalLog("Just ran it on global!")
			}
        }()
		return &pb.ScheduleResponse{Uid: uid}, nil
	}

	if scheduleLocally.NumRunningTasks < MAX_TASKS {
		// LocalLog("Just running locally")
		go func() {
			s.gcsClient.RegisterGenerating(ctx, &pb.GeneratingRequest{Uid: uid, NodeId: s.localNodeID})
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
            _, err := s.globalSchedulerClient.Schedule(s.globalCtx, &pb.GlobalScheduleRequest{Uid: uid, Name: req.Name, Args: req.Args, Kwargs: req.Kwargs, NewObject: true, Uids: req.Uids})
            if err != nil {
                LocalLog("cannot contact global scheduler")
            } else {
				// LocalLog("Just ran it on global!")
			}
        }()

	}
	return &pb.ScheduleResponse{Uid: uid}, nil

}

func (s *server) KillServer(ctx context.Context, req *pb.StatusResponse) (*pb.StatusResponse, error) {
	s.alive = false
	return &pb.StatusResponse{Success: true}, nil
}

func (s *server) ReviveServer(ctx context.Context, req *pb.StatusResponse) (*pb.StatusResponse, error) {
	s.alive = true
	return &pb.StatusResponse{Success: true}, nil
}

func (s *server) SendHeartbeats(ctx context.Context, globalSchedulerClient pb.GlobalSchedulerClient, nodeId uint64 ) {

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
		if s.alive {
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
				LocalLog("got error from AvgBand(): %v", err)
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
		}
	    time.Sleep(HEARTBEAT_WAIT)
	}
}

