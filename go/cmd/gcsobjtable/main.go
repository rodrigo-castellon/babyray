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
	"github.com/rodrigo-castellon/babyray/util"
	"github.com/rodrigo-castellon/babyray/customlog"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var cfg *config.Config

// LocalLog formats the message and logs it with a specific prefix
func LocalLog(format string, v ...interface{}) {
	var logMessage string
	if len(v) == 0 {
		logMessage = format // No arguments, use the format string as-is
	} else {
		logMessage = fmt.Sprintf(format, v...)
	}
	log.Printf("[lobs] %s", logMessage)
}

func main() {
	customlog.Init()
	cfg = config.GetConfig()                                // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.GCSObjectTable) // Prepare the network address

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer(util.GetServerOptions()...)
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
}

func NewGCSObjServer() *GCSObjServer {
	server := &GCSObjServer{
		objectLocations: make(map[uint64][]uint64),
		waitlist:        make(map[uint64][]string),
		mu:              sync.Mutex{},
		objectSizes:     make(map[uint64]uint64),
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
	if !exists || len(nodeIds) == 0 {
		return nil, false
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
	conn, err := grpc.Dial(clientAddress, util.GetDialOptions()...)
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

	log.Printf("WAS JUST NOTIFYOWNS()ED")

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
	if !exists {
		// Add client to waiting list
		if _, exists := s.waitlist[uid]; !exists {
			s.waitlist[uid] = []string{} // Initialize slice if it doesn't exist
		}
		s.waitlist[uid] = append(s.waitlist[uid], clientAddress)

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
	log.Printf("DEEP PRINT!")
	log.Printf("length = %v", len(s.objectLocations))
	for k, v := range s.objectLocations {
		log.Printf("s.objectLocations[%v] = %v", k, v)
	}

	for _, u := range req.Args {
		if _,ok := s.objectLocations[uint64(u)]; ok {
			locations[uint64(u)] = &pb.LocationByteTuple{Locations: s.objectLocations[uint64(u)], Bytes: s.objectSizes[uint64(u)]}
		}
	}
	return &pb.ObjectLocationsResponse{Locations: locations}, nil
}