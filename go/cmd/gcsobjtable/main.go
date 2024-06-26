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
	log.Printf("[gcsobjtable] %s", logMessage)
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
	// log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

// server is used to implement your gRPC service.
type GCSObjServer struct {
	pb.UnimplementedGCSObjServer
	objectLocations       map[uint64][]uint64 // object uid -> list of nodeIds as uint64
	waitlist              map[uint64][]string // object uid -> list of IP addresses as string
	mu                    sync.Mutex          // lock should be used for both objectLocations and waitlist
	database              *sql.DB             // connection to SQLite persistent datastore
	ticker                *time.Ticker        // for GCS flushing
	lineageMu             sync.RWMutex        // lock should be used for s.lineage
	objectSizes           map[uint64]uint64
	lineage               map[uint64]*pb.GlobalScheduleRequest
	globalSchedulerClient SchedulerClient
	liveNodes             map[uint64]bool
	generating            map[uint64]uint64 //object uid -> node id of original generator
	//used to determine when an original creation of a uid should be restarted
	globalCtx context.Context
}

type SchedulerClient interface {
	Schedule(ctx context.Context, req *pb.GlobalScheduleRequest, opts ...grpc.CallOption) (*pb.StatusResponse, error)
	Heartbeat(ctx context.Context, req *pb.HeartbeatRequest, opts ...grpc.CallOption) (*pb.StatusResponse, error)
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

	globalSchedulerAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GlobalScheduler, cfg.Ports.GlobalScheduler)
	conn, _ := grpc.Dial(globalSchedulerAddress, grpc.WithInsecure())
	globalSchedulerClient := pb.NewGlobalSchedulerClient(conn)

	server := &GCSObjServer{
		objectLocations:       make(map[uint64][]uint64),
		waitlist:              make(map[uint64][]string),
		mu:                    sync.Mutex{},
		database:              database,
		ticker:                nil,
		objectSizes:           make(map[uint64]uint64),
		lineage:               make(map[uint64]*pb.GlobalScheduleRequest),
		globalSchedulerClient: globalSchedulerClient,
		liveNodes:             make(map[uint64]bool),
		generating:            make(map[uint64]uint64),
		globalCtx:             context.Background(),
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
					//log.Printf("Successfully flushed to disk!")
				}
			}
		}()
	}

	logMemoryUsage := true // TODO: REMOVE HARDCODE
	if logMemoryUsage {
		go func() {
			for {
				var memStats runtime.MemStats
				runtime.ReadMemStats(&memStats)

				// HeapAlloc: Bytes of allocated heap objects (heap memory in use). in MB
				log.Printf("%v\n", bToMb(memStats.HeapAlloc))

				// Sleep for 5 seconds before the next iteration
				time.Sleep(5 * time.Second)
			}
		}()
	}

	return server
}

func bToMb(b uint64) float64 {
	return float64(b) / 1024 / 1024
}

func (s *GCSObjServer) flushToDisk() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	garbage_collect := true     // TODO: REMOVE HARDCODED
	flush_to_AOF := false       // TODO: REMOVE HARDCODED
	aof_filename := "./aof.txt" // TODO: REMOVE HARDCODED

	// Can either flush to append-only file or to DB
	if flush_to_AOF {
		err := WriteObjectLocationsToAOF(aof_filename, s.objectLocations)
		if err != nil {
			return err
		}
		//log.Printf("Successfully flushed to AOF!")
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
		//log.Println("Garbage collection triggered")
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
Returns a nodeId that has object uid. If it has never been added to objectLocations,
then return false. Otherwise, return True.
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

	nodesToReturn := make([]uint64, 1, 1)
	for _, n := range nodeIds {
		if !s.liveNodes[n] {
			nodesToReturn = append(nodesToReturn, n)
		}
	}

	if len(nodesToReturn) == 0 {
		return nil, false, nil
	}

	// Note: policy is to pick a random one; in the future it will need to be locality-based
	randomIndex := rand.Intn(len(nodesToReturn))
	nodeId := &nodesToReturn[randomIndex]
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
		//	log.Printf("Failed to connect back to client at %s: %v", clientAddress, err)
		return
	}
	defer conn.Close() // TODO: remove in some eventual universe

	localObjStoreClient := pb.NewLocalObjStoreClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call LocationFound and handle any potential error
	_, err = localObjStoreClient.LocationFound(ctx, &pb.LocationFoundCallback{Uid: uid, Location: nodeId})
	if err != nil {
		//	log.Printf("Failed to send LocationFound callback for UID %d to client at %s: %v", uid, clientAddress, err)
	}
}

func (s *GCSObjServer) NotifyOwns(ctx context.Context, req *pb.NotifyOwnsRequest) (*pb.StatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	uid, nodeId := req.Uid, req.NodeId
	delete(s.generating, uid)
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
		if _, waiting := s.waitlist[uid]; !waiting {
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

func (s *GCSObjServer) RegisterLineage(ctx context.Context, req *pb.GlobalScheduleRequest) (*pb.StatusResponse, error) {
	s.lineageMu.RLock()
	if _, ok := s.lineage[req.Uid]; ok {
		s.lineageMu.RUnlock()
		return &pb.StatusResponse{Success: false}, status.Error(codes.Internal, "tried to register duplicate lineage")
	}
	s.lineageMu.RUnlock()
	s.lineageMu.Lock()
	s.lineage[req.Uid] = req
	s.lineageMu.Unlock()
	return &pb.StatusResponse{Success: false}, nil
}

func (s *GCSObjServer) getNodes(uid uint64) []uint64 {
	var nodes []uint64
	if node, ok := s.generating[uid]; ok {
		nodes = append(nodes, node)
	}

	if objectNodes, ok := s.objectLocations[uid]; ok {
		for _, node := range objectNodes {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (s *GCSObjServer) allDead(nodes []uint64) bool {
	for _, node := range nodes {
		if s.liveNodes[node] {
			return false
		}
	}
	return true
}

func (s *GCSObjServer) RegisterLiveNodes(ctx context.Context, req *pb.LiveNodesRequest) (*pb.StatusResponse, error) {

	// LocalLog("got RegisterLiveNodes() call.")

	s.liveNodes = req.LiveNodes

	// if literally everything is dead: generating AND object locations
	// then reschedule. but if there is at least one live node that is either
	// gonna generate it or storing it then don't

	var regenerateList []uint64
	allUids := make(map[uint64]bool)

	for uid, _ := range s.generating {
		allUids[uid] = true
	}
	for uid, _ := range s.objectLocations {
		allUids[uid] = true
	}

	for uid, _ := range allUids {
		nodes := s.getNodes(uid)

		// LocalLog("the nodes gotten here is: %v", nodes)

		if s.allDead(nodes) {
			regenerateList = append(regenerateList, uid)
		}
	}

	for _, uid := range regenerateList {
		// LocalLog("uid = %v, rescheduling", uid)
		go func() {
			_, err := s.globalSchedulerClient.Schedule(s.globalCtx, s.lineage[uid])
			if err != nil {
				// LocalLog("cannot contact global scheduler, err = %v", err)
			} else {
				// LocalLog("Just ran it on global!")
			}
		}()
	}
	return &pb.StatusResponse{Success: true}, nil
}

func (s *GCSObjServer) RegisterGenerating(ctx context.Context, req *pb.GeneratingRequest) (*pb.StatusResponse, error) {
	// LocalLog("trying to register node %v as a generating node", req.NodeId)
	// if id, ok := s.generating[req.Uid]; ok {
	// 	return &pb.StatusResponse{Success: false}, status.Error(codes.Internal, fmt.Sprintf("node %d is already generating uid %d", id, req.Uid))
	// }
	s.generating[req.Uid] = req.NodeId
	return &pb.StatusResponse{Success: true}, nil
}
