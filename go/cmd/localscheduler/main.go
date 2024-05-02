package main

import (
    // "context"
    "log"
    "net"
    "strconv"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"
)



func main() {
    cfg := config.LoadConfig() // Load configuration
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
}

// server is used to implement your gRPC service.
type server struct {
   pb.UnimplementedLocalSchedulerServer
}

// Implement your service methods here.


func (s *server) Schedule(ctx context.Context, req *pb.ScheduleRequest) (*pb.ScheduleResponse, error) {
    // worker_id = check_resources()
    worker_id = nil; 
    uid = int.rand(); 
    // message RunRequest {
    //     uint32 uid = 1;
    //     string name = 2;
    //     bytes args = 3;
    //     bytes kwargs = 4;
    //   }
    if worker_id != nil {
        c := NewWorkerClient()
        r, err = c.Run(&pb.RunRequest{uid, req.Name, req.Args, req.Kwargs})
        if err != nil {
            log.Printf("cannot contact worker %d", worker_id)
        }
    } else {
        c := NewGlobalSchedulerClient()
        _, err = c.Schedule(ctx, &pb.GlobalScheduleRequest{uid, req.Name, req.Args, req.Kwargs})
        if err != nil {
            log.Printf("cannot contact global scheduler")
        }

    }
    return &pb.ScheduleResponse{uid}, nil 

}
// func (s *server) NotifyOwns(ctx context.Context, req *pb.NotifyOwnsRequest) (*pb.StatusResponse, error) {
//     s.mu.Lock()
//     defer s.mu.Unlock()

//     // Append the nodeId to the list for the given uid
//     s.objectLocations[req.Uid] = append(s.objectLocations[req.Uid], req.NodeId)
//     //log.Printf("NotifyOwns: Added Node %d to UID %d", req.NodeId, req.Uid)

//     return &pb.StatusResponse{Success: true}, nil
// }

// func (s *server) RequestLocation(ctx context.Context, req *pb.RequestLocationRequest) (*pb.StatusResponse, error) {
//     s.mu.Lock()
//     nodeIds, exists := s.objectLocations[req.Uid]
//     s.mu.Unlock()

//     if !exists || len(nodeIds) == 0 {
//         log.Printf("RequestLocation: No locations found for UID %d", req.Uid)
//         // Returning a StatusResponse indicating failure, instead of nil and a Go error
//         return &pb.StatusResponse{
//             Success:      false,
//             ErrorCode:    404, // Or another appropriate error code
//             ErrorMessage: "no locations found for given UID",
//             Details:      "The requested UID does not exist in the object locations map.",
//         }, nil // No error returned here; encoding the failure in the response message
//     }

//     // Assume successful case
//     //log.Printf("RequestLocation: Returning location for UID %d: Node %d", req.Uid, nodeIds[0])
//     return &pb.StatusResponse{
//         Success: true,
//         Details: strconv.Itoa(int(nodeIds[0])),
//     }, nil
// }
