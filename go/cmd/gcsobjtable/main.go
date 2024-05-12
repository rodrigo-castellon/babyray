package main

import (
	"context"
	"fmt"
	"time"

	// "errors"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func main() {
	cfg := config.GetConfig()                               // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.GCSObjectTable) // Prepare the network address

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer()
	pb.RegisterGCSObjServer(s, NewGCSObjServer())
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// server is used to implement your gRPC service.
type GCSObjServer struct {
	pb.UnimplementedGCSObjServer
	objectLocations map[uint64][]uint64
	mu              sync.Mutex
	cond            *sync.Cond
}

func NewGCSObjServer() *GCSObjServer {
	server := &GCSObjServer{
		objectLocations: make(map[uint64][]uint64),
		mu:              sync.Mutex{}, // Mutex initialized here.
	}
	server.cond = sync.NewCond(&server.mu) // Properly pass the address of the struct's mutex.
	return server
}

// Implement your service methods here.
func (s *GCSObjServer) NotifyOwns(ctx context.Context, req *pb.NotifyOwnsRequest) (*pb.StatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Append the nodeId to the list for the given uid
	s.objectLocations[req.Uid] = append(s.objectLocations[req.Uid], req.NodeId)
	s.cond.Broadcast()

	return &pb.StatusResponse{Success: true}, nil
}

func (s *GCSObjServer) getNodeId(uid uint64) (*uint64, bool) {
	nodeIds, exists := s.objectLocations[uid]
	if !exists || len(nodeIds) == 0 {
		return nil, false
	}

	// Note: policy is to pick a random one; in the future it will need to be locality-based
	randomIndex := rand.Intn(len(nodeIds))
	nodeId := &nodeIds[randomIndex]
	return nodeId, true
}

func (s *GCSObjServer) RequestLocation(ctx context.Context, req *pb.RequestLocationRequest) (*pb.RequestLocationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract client address
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "could not get peer information; hence, failed to extract client address")
	}
	clientAddress := p.Addr.String()

	// Set up a new gRPC connection to the client
	// TODO: Refactor to save gRPC connections rather than creating a new one each time
	// Dial is lazy-loading, but we should still save the connection for future use
	conn, err := grpc.Dial(clientAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		errorMessage := fmt.Sprintf("Failed to connect back to client: %v", err)
		return nil, status.Error(codes.Internal, errorMessage)
	}
	//defer conn.Close() // TODO: remove

	gcsObjClient := pb.NewLocalObjStoreClient(conn)

	nodeId, exists := s.getNodeId(req.Uid)
	if !exists {
		// TODO: Add client to waiting list - also should be lock-based waiting list

		return &pb.RequestLocationResponse{
			ImmediatelyFound: false,
		}, nil
	}

	// Send immediate callback
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		gcsObjClient.LocationFound(ctx, &pb.RequestLocationCallback{NodeId: *nodeId})
	}()
	return &pb.RequestLocationResponse{
		ImmediatelyFound: true,
	}, nil
}

/* NOTE: We only use one cv for now, which may cause performance issues in the future
However, there is significant overhead if we want one cv per object, as then we would have to manage
the cleanup through reference counting  */
