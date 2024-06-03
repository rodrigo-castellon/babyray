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

	"github.com/rodrigo-castellon/babyray/config"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var cfg *config.Config

func main() {
	/* Set up GCS Object Table */
	cfg = config.GetConfig()                                // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.GCSObjectTable) // Prepare the network address

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer()
	pb.RegisterGCSObjServer(s, NewGCSObjServer(cfg.GCS.FlushIntervalSec))
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
	database        *sql.DB             // connection to SQLite persistent datastore
	ticker          *time.Ticker        // for GCS flushing
}

/* set flushIntervalSec to -1 to disable GCS flushing */
func NewGCSObjServer(flushIntervalSec int) *GCSObjServer {
	/* Set up SQLite */
	// Note: You don't need to call database.Close() in Golang: https://stackoverflow.com/a/50788205
	database, err := sql.Open("sqlite3", "./gcsobjtable.db") // TODO: Remove hardcode to config
	if err != nil {
		log.Fatal(err)
	}
	// Creates table if it doesn't already exist
	createObjectLocationsTable(database)

	/* Create server object */
	server := &GCSObjServer{
		objectLocations: make(map[uint64][]uint64),
		waitlist:        make(map[uint64][]string),
		mu:              sync.Mutex{},
		database:        database,
		ticker:          nil,
	}
	if flushIntervalSec != -1 {
		// Launch periodic disk flushing
		interval := time.Duration(flushIntervalSec) * time.Second
		server.ticker = time.NewTicker(interval)
		go func() {
			for range server.ticker.C {
				err := server.flushToDisk()
				if err != nil {
					log.Printf("Error flushing to disk: %v", err)
				}
			}
		}()
	}
	return server
}

func (s *GCSObjServer) flushToDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := insertOrUpdateObjectLocations(s.database, s.objectLocations)
	if err != nil {
		return err
	}
	// Completely delete the current map in memory and start blank
	s.objectLocations = make(map[uint64][]uint64) // orphaning the old map will get it garbage collected
	return nil
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
