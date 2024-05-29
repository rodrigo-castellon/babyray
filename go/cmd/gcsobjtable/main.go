package main

import (
	"context"
	"time"

	// "errors"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"fmt"
	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var cfg *config.Config

func main() {
	cfg = config.GetConfig()                                // Load configuration
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
	objectLocations map[uint64][]uint64 // object uid -> list of nodeIds as uint64
	waitlist        map[uint64][]string // object uid -> list of IP addresses as string
	mu              sync.Mutex          // lock should be used for both objectLocations and waitlist
	objectSizes     map[uint64]uint64
	lineage			map[uint64]*pb.GlobalScheduleRequest
	globalSchedulerClient pb.GlobalSchedulerClient
}

func NewGCSObjServer() *GCSObjServer {
	globalSchedulerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GlobalScheduler, cfg.Ports.GlobalScheduler)
	conn, _ := grpc.Dial(globalSchedulerAddress, grpc.WithInsecure())
	globalSchedulerClient := pb.NewGlobalSchedulerClient(conn)
	
	server := &GCSObjServer{
		objectLocations: make(map[uint64][]uint64),
		waitlist:        make(map[uint64][]string),
		mu:              sync.Mutex{},
		objectSizes:     make(map[uint64]uint64),
		lineage:         make(map[uint64]*pb.GlobalScheduleRequest),
		globalSchedulerClient: globalSchedulerClient,
		
	}
	return server
}

/*
Returns a nodeId that has object uid. If it doesn't exist anywhere,
then the second return value will be false.
Assumes that s's mutex is locked.
*/
func (s *GCSObjServer) getNodeId(uid uint64) (*uint64, bool) {
	nodeIds, exists := s.objectLocations[uid]
	if !exists {
		return nil, false
	}
	if len(nodeIds) == 0 {
		return nil, true
	}

	// Note: policy is to pick a random one; in the future it will need to be locality-based
	randomIndex := rand.Intn(len(nodeIds))
	nodeId := &nodeIds[randomIndex]
	return nodeId, true
}

// sendCallback sends a location found callback to the local object store client
// This should be used as a goroutine
func (s *GCSObjServer) sendCallback(clientAddress string, uid uint64, nodeId uint64) {
	// Set up a new gRPC connection to the client
	// TODO: Refactor to save gRPC connections rather than creating a new one each time
	// Dial is lazy-loading, but we should still save the connection for future use
	conn, err := grpc.Dial(clientAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// Log the error instead of returning it
		log.Printf("Failed to connect back to client at %s: %v", clientAddress, err)
		return
	}
	defer conn.Close() // TODO: remove in some eventual universe

	localObjStoreClient := pb.NewLocalObjStoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call LocationFound and handle any potential error
	_, err = localObjStoreClient.LocationFound(ctx, &pb.LocationFoundCallback{Uid: uid, Location: nodeId})
	if err != nil {
		log.Printf("Failed to send LocationFound callback for UID %d to client at %s: %v", uid, clientAddress, err)
	}
}

func (s *GCSObjServer) NotifyOwns(ctx context.Context, req *pb.NotifyOwnsRequest) (*pb.StatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	uid, nodeId := req.Uid, req.NodeId

	// Append the nodeId to the list for the given object uid
	if _, exists := s.objectLocations[uid]; !exists {
		s.objectLocations[uid] = []uint64{} // Initialize slice if it doesn't exist
		s.objectSizes[uid] = req.ObjectSize
	}
	s.objectLocations[uid] = append(s.objectLocations[uid], nodeId)

	// Clear waitlist for this uid, if any
	waitingIPs, exists := s.waitlist[uid]
	if exists {
		for _, clientAddress := range waitingIPs {
			go s.sendCallback(clientAddress, uid, nodeId)
		}
		// Clear the waitlist for the given uid after processing
		delete(s.waitlist, uid)
	}

	return &pb.StatusResponse{Success: true}, nil
}

func (s *GCSObjServer) RequestLocation(ctx context.Context, req *pb.RequestLocationRequest) (*pb.RequestLocationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Extract client address
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "could not get peer information; hence, failed to extract client address")
	}

	// /////
	// log.SetOutput(os.Stdout)
	// log.Println(p.Addr.String())
	// //////////

	host, _, err := net.SplitHostPort(p.Addr.String()) // Strip out the ephemeral port
	if err != nil {
		//return nil, status.Error(codes.Internal, "could not split host and port")
		// Temporary workaround: If splitting fails, use the entire address string as the host
		host = p.Addr.String()
	}
	clientPort := strconv.Itoa(cfg.Ports.LocalObjectStore) // Replace the ephemeral port with the official LocalObjectStore port
	clientAddress := net.JoinHostPort(host, clientPort)

	uid := req.Uid
	nodeId, exists := s.getNodeId(uid)
	if nodeId == nil {
		// Add client to waiting list
		if _, waiting := s.waitlist[uid]; !waiting {
			s.waitlist[uid] = []string{} // Initialize slice if it doesn't exist
		}
		s.waitlist[uid] = append(s.waitlist[uid], clientAddress)
		if exists {
			_, err := s.globalSchedulerClient.Schedule(ctx, s.lineage[uid])
			if err != nil {
				log.Fatalf("unable to contact global scheduler")
			}
		}
		// Reply to this gRPC request
		return &pb.RequestLocationResponse{
			ImmediatelyFound: false,
		}, nil
	}

	// Send immediate callback
	go s.sendCallback(clientAddress, uid, *nodeId)

	// Reply to this gRPC request
	return &pb.RequestLocationResponse{
		ImmediatelyFound: true,
	}, nil
}

func (s *GCSObjServer) GetObjectLocations(ctx context.Context, req *pb.ObjectLocationsRequest) (*pb.ObjectLocationsResponse, error) {
	locations := make(map[uint64]*pb.LocationByteTuple)
	for u := range req.Args {
		if _,ok := s.objectLocations[uint64(u)]; ok {
			locations[uint64(u)] = &pb.LocationByteTuple{Locations: s.objectLocations[uint64(u)], Bytes: s.objectSizes[uint64(u)]}
		}
		
	}
	return &pb.ObjectLocationsResponse{Locations: locations}, nil
}

func (s *GCSObjServer) RegisterLineage(ctx context.Context, req *pb.GlobalScheduleRequest) (*pb.StatusResponse, error) {
	if _, ok := s.lineage[req.Uid]; ok {
		return &pb.StatusResponse{Success: false}, status.Error(codes.Internal, "tried to register duplicate lineage")
	}
	s.lineage[req.Uid] = req
	return &pb.StatusResponse{Success: false}, nil
}
