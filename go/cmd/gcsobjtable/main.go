package main

import (
	"bufio"
	"context"
	"os"
	"time"

	// "errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"runtime"
	"strconv"
	"sync"

	"github.com/rodrigo-castellon/babyray/config"
	"github.com/rodrigo-castellon/babyray/customlog"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"github.com/rodrigo-castellon/babyray/util"
	"google.golang.org/grpc"

	// "google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/peer"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
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
	/* Set up GCS Object Table */
	customlog.Init()
	cfg = config.GetConfig()                                // Load configuration
	address := ":" + strconv.Itoa(cfg.Ports.GCSObjectTable) // Prepare the network address

	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	_ = lis
	s := grpc.NewServer(util.GetServerOptions()...)
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
	objectSizes     map[uint64]uint64
}

/* set flushIntervalSec to -1 to disable GCS flushing */
func NewGCSObjServer(flushIntervalSec int) *GCSObjServer {
	/* Set up SQLite */
	// Note: You don't need to call database.Close() in Golang: https://stackoverflow.com/a/50788205
	database, err := sql.Open("sqlite3", "./gcsobjtable.db") // TODO: Remove hardcode to config
	if err != nil {
		log.Fatal(err)
	}
	setCacheSizeToZero(database)
	// Creates table if it doesn't already exist
	createObjectLocationsTable(database)

	/* Create server object */
	server := &GCSObjServer{
		objectLocations: make(map[uint64][]uint64),
		waitlist:        make(map[uint64][]string),
		mu:              sync.Mutex{},
		database:        database,
		ticker:          nil,
		objectSizes:     make(map[uint64]uint64),
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
				} else {
					log.Printf("Successfully flushed to disk!")
				}
			}
		}()
	}
	return server
}

func (s *GCSObjServer) flushToDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	garbage_collect := false     // TODO: REMOVE HARDCODED
	flush_to_AOF_instead := true // TODO: REMOVE HARDCODED
	aof_filename := "./aof.txt"  // TODO: REMOVE HARDCODED

	if flush_to_AOF_instead {
		WriteObjectLocationsToAOF(aof_filename, s.objectLocations)
	} else {
		// Flush to SQLite3 Disk Database
		err := insertOrUpdateObjectLocations(s.database, s.objectLocations)
		if err != nil {
			return err
		}
	}

	// Completely delete the current map in memory and start blank
	s.objectLocations = make(map[uint64][]uint64) // orphaning the old map will get it garbage collected
	// Manually trigger garbage collection if desired
	if garbage_collect {
		runtime.GC()
		fmt.Println("Garbage collection triggered")
	}
	return nil
}

// WriteObjectLocationsToAOF appends the contents of the objectLocations map to a file
func WriteObjectLocationsToAOF(filename string, objectLocations map[uint64][]uint64) error {
	// Open file in append mode, create if it doesn't exist
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Create a buffered writer
	writer := bufio.NewWriter(file)

	// Iterate over the map and write to file
	for key, values := range objectLocations {
		// Convert key to string
		keyStr := strconv.FormatUint(key, 10)

		// Convert values slice to a comma-separated string
		var valuesStr string
		for i, val := range values {
			if i > 0 {
				valuesStr += ","
			}
			valuesStr += strconv.FormatUint(val, 10)
		}

		// Write key and values to file
		_, err := writer.WriteString(fmt.Sprintf("%s: [%s]\n", keyStr, valuesStr))
		if err != nil {
			return fmt.Errorf("error writing to file: %w", err)
		}
	}

	// Flush the buffered writer
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("error flushing buffer: %w", err)
	}

	return nil
}

/*
Returns a nodeId that has object uid. If it doesn't exist anywhere,
then the second return value will be false.
Assumes that s's mutex is locked.
The return values are the location, boolean if it exists, and error
*/
func (s *GCSObjServer) getNodeId(uid uint64) (*uint64, bool, error) {
	nodeIds, exists := s.objectLocations[uid]
	if !exists || len(nodeIds) == 0 {
		// Not found in memory; it's a cache miss so let's try disk
		var err error
		nodeIds, err = getObjectLocations(s.database, uid)
		if err != nil {
			return nil, false, err
		}
		// Move state into cache, even if it's empty so we know how to answer in the future
		s.objectLocations[uid] = nodeIds
		if len(nodeIds) == 0 {
			// Not found
			return nil, false, nil
		}
		// Proceed below
	}

	// Note: policy is to pick a random one; in the future it will need to be locality-based
	randomIndex := rand.Intn(len(nodeIds))
	nodeId := &nodeIds[randomIndex]
	return nodeId, true, nil
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

	host, _, err := net.SplitHostPort(p.Addr.String()) // Strip out the ephemeral port
	if err != nil {
		//return nil, status.Error(codes.Internal, "could not split host and port")
		// Temporary workaround: If splitting fails, use the entire address string as the host
		host = p.Addr.String()
	}
	clientPort := strconv.Itoa(cfg.Ports.LocalObjectStore) // Replace the ephemeral port with the official LocalObjectStore port
	clientAddress := net.JoinHostPort(host, clientPort)

	uid := req.Uid
	nodeId, exists, err := s.getNodeId(uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "could not get node id: "+err.Error())
	}

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

	for _, u := range req.Args {
		if _, ok := s.objectLocations[uint64(u)]; ok {
			locations[uint64(u)] = &pb.LocationByteTuple{Locations: s.objectLocations[uint64(u)], Bytes: s.objectSizes[uint64(u)]}
		}
	}
	return &pb.ObjectLocationsResponse{Locations: locations}, nil
}
