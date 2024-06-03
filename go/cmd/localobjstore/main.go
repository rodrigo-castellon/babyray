package main

import (
	// "context"
	context "context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"os"
	// "time"
	"sync"
	"syscall"
	// "golang.org/x/sys/unix"

	"github.com/rodrigo-castellon/babyray/config"
	"github.com/rodrigo-castellon/babyray/customlog"
	"github.com/rodrigo-castellon/babyray/util"
	pb "github.com/rodrigo-castellon/babyray/pkg"
	"google.golang.org/grpc"
)

const DEFAULT_AVG_BANDWIDTH = 0.1 // to start with

func createSharedMemory(name string, size int) ([]byte, int, error) {
    // Create a shared memory segment
	LocalLog("WERE GONNA CREATE A /DEV FILE WITH THIS NAME: %s", "/dev/shm/"+name)
    shmFd, err := syscall.Open("/dev/shm/"+name, syscall.O_CREAT|syscall.O_RDWR, 0666)
    if err != nil {
        return nil, -1, err
    }

    // Set the size of the shared memory object
    if err := syscall.Ftruncate(shmFd, int64(size)); err != nil {
        return nil, -1, err
    }

    // Map the shared memory object into the process's address space
    shmAddr, err := syscall.Mmap(shmFd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
    if err != nil {
        return nil, -1, err
    }

	LocalLog("/DEV FILE CREATED!")

    return shmAddr, shmFd, nil
}

// Function to open and read from a shared memory segment
func readSharedMemory(name string, size int) ([]byte, error) {
	LocalLog("WERE GONNA READ A /DEV FILE WITH THIS NAME: %s", "/dev/shm/"+name)
	// Open the shared memory object
	shmFd, err := syscall.Open("/dev/shm/"+name, syscall.O_RDWR, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open shared memory: %v", err)
	}

	// Map the shared memory object into the process's address space
	shmAddr, err := syscall.Mmap(shmFd, 0, size, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, fmt.Errorf("failed to mmap shared memory: %v", err)
	}

	// Close the file descriptor as it is no longer needed
	if err := syscall.Close(shmFd); err != nil {
		return nil, fmt.Errorf("failed to close shared memory file descriptor: %v", err)
	}

	return shmAddr, nil
}

func deleteSharedMemory(name string, shmFd int) error {
    // Close the file descriptor
    if err := syscall.Close(shmFd); err != nil {
        return err
    }
    // Unlink the shared memory object
    if err := syscall.Unlink("/dev/shm/" + name); err != nil {
        return err
    }
    return nil
}


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

var cfg *config.Config
const EMA_PARAM float32 = .9
var mu sync.RWMutex

func main() {
	customlog.Init()
	cfg = config.GetConfig()                                  // Load configuration
    startServer(":" + strconv.Itoa(cfg.Ports.LocalObjectStore))
	// Create a channel and block on it to prevent the main function from exiting
	block := make(chan struct{})
	<-block
}

func startServer(port string) (*grpc.Server, error) {

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(util.GetServerOptions()...)
    gcsAddress := fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, cfg.NodeIDs.GCS, cfg.Ports.GCSObjectTable)
	conn, _ := grpc.Dial(gcsAddress, util.GetDialOptions()...)
	nodeId, _ := strconv.Atoi(os.Getenv("NODE_ID"))
	pb.RegisterLocalObjStoreServer(s, &server{localObjectStore: make(map[uint64][]byte), objectSizes: make(map[uint64]uint64), localObjectChannels: make(map[uint64]chan *pb.LocationFoundCallback), gcsObjClient: pb.NewGCSObjClient(conn), localNodeID: uint64(nodeId), avgBandwidth: DEFAULT_AVG_BANDWIDTH})


	LocalLog("lobs server listening at %v", lis.Addr())
	go func() {
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
	}()


	return s, nil
}

// server is used to implement your gRPC service.
type server struct {
	pb.UnimplementedLocalObjStoreServer
	localObjectStore    map[uint64][]byte
	// set storing which uid's are stored in POSIX shared memory -> sizes
	objectSizes			map[uint64]uint64
	localObjectChannels map[uint64]chan *pb.LocationFoundCallback
	gcsObjClient        pb.GCSObjClient
	localNodeID         uint64
	avgBandwidth        float32
}

func (s *server) Store(ctx context.Context, req *pb.StoreRequest) (*pb.StatusResponse, error) {
	size := uint64(len(req.ObjectBytes))

	// store to shared memory (internally sync'd)
	shmSlice, _, err := createSharedMemory(strconv.FormatUint(req.Uid, 10), int(size))
	if err != nil {
		return nil, err
	}

	LocalLog("NOW ATTEMPTING TO COPY OBJECT BYTES INTO SHARED MEMORY SLICE")

	copy(shmSlice, req.ObjectBytes)


	LocalLog("COPIED!")

	mu.Lock()
	// s.localObjectStore[req.Uid] = req.ObjectBytes
	// store this in our set
	s.objectSizes[req.Uid] = size;
	mu.Unlock()

	LocalLog("STORED!!!")

	s.gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: req.Uid, NodeId: s.localNodeID, ObjectSize: size})
	return &pb.StatusResponse{Success: true}, nil
}

func (s *server) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	mu.RLock()
	if val, ok := s.objectSizes[req.Uid]; ok {
		mu.RUnlock()
		LocalLog("ITS IN OBJECT SIZES!")
		return &pb.GetResponse{Uid: req.Uid, ObjectBytes: []byte{}, Local: true, Size: val}, nil
	}
	LocalLog("THE UID WAS NOT IN OBJECT SIZES: %v", req.Uid)
	// if val, ok := s.localObjectStore[req.Uid]; ok {
	// 	mu.RUnlock()
	// 	return &pb.GetResponse{Uid: req.Uid, ObjectBytes: val, Local: true}, nil
	// }
	mu.RUnlock()

	// LocalLog("IN GET() RN!")

	s.localObjectChannels[req.Uid] = make(chan *pb.LocationFoundCallback)
	if req.Testing == false {
		s.gcsObjClient.RequestLocation(ctx, &pb.RequestLocationRequest{Uid: req.Uid, Requester: s.localNodeID})
	}

	resp := <-s.localObjectChannels[req.Uid]

	// LocalLog("GOT THE RESPONSE! %v", resp)
	// LocalLog("the locaiton is: %v", resp.Location)

	// handle the response accordingly
	if (!req.Copy) {
		return &pb.GetResponse{Uid: req.Uid}, nil
	}
	var otherLocalAddress string

	if resp.Port == 0 {
		nodeID := resp.Location
		otherLocalAddress = fmt.Sprintf("%s%d:%d", cfg.DNS.NodePrefix, nodeID, cfg.Ports.LocalObjectStore)
	} else {
		otherLocalAddress = fmt.Sprintf("%s:%d", resp.Address, resp.Port)
	}

	conn, err := grpc.Dial(otherLocalAddress, util.GetDialOptions()...)

	if err != nil {
		return &pb.GetResponse{Uid: req.Uid}, errors.New(fmt.Sprintf("failed to dial other LOS @:%s ", otherLocalAddress))
		// return &pb.StatusResponse{Success: false}, errors.New(fmt.Sprintf("failed to dial other LOS @:%s ", otherLocalAddress))
	}

	c := pb.NewLocalObjStoreClient(conn)

	// start := time.Now()

	// LocalLog("CALLING COPY() ON THIS NODE...")
	x, err := c.Copy(ctx, &pb.CopyRequest{Uid: resp.Uid, Requester: s.localNodeID})

	// LocalLog("the err was: %v", err)
	// LocalLog("GETTING THE TOTAL BANDWIDTH FROM THIS...")
	// bandwidth := float32(len(x.ObjectBytes)) / float32((time.Now().Sub(start).Seconds()))

	// s.avgBandwidth = EMA_PARAM * s.avgBandwidth + (1 - EMA_PARAM) * bandwidth 

	if x == nil || err != nil {
		return &pb.GetResponse{Uid: req.Uid}, errors.New(fmt.Sprintf("failed to copy from other LOS @:%s. err was: %v ", otherLocalAddress, err))
	}

	// val := <-s.localObjectChannels[req.Uid]
	if (req.Cache) {
		size := uint64(len(x.ObjectBytes))

		// store to shared memory (internally sync'd)
		shmSlice, _, err := createSharedMemory(strconv.FormatUint(req.Uid, 10), int(size))
		if err != nil {
			return nil, err
		}

		copy(shmSlice, x.ObjectBytes)

		mu.Lock()
		s.objectSizes[req.Uid] = size;

		// s.localObjectStore[req.Uid] = x.ObjectBytes
		mu.Unlock()
		if (resp.Port == 0) {
			s.gcsObjClient.NotifyOwns(ctx, &pb.NotifyOwnsRequest{Uid: resp.Uid, NodeId: s.localNodeID})
		}
		return &pb.GetResponse{Uid: req.Uid, ObjectBytes: x.ObjectBytes, Local: false, Size: size}, nil
	}
	return &pb.GetResponse{Uid: req.Uid, ObjectBytes: x.ObjectBytes, Local: false}, nil
}

func (s *server) LocationFound(ctx context.Context, resp *pb.LocationFoundCallback) (*pb.StatusResponse, error) {

	channel, _ := s.localObjectChannels[resp.Uid]
	channel <- resp

	return &pb.StatusResponse{Success: true}, nil

}

func (s *server) Copy(ctx context.Context, req *pb.CopyRequest) (*pb.CopyResponse, error) {
	// name := fmt.Sprintf("shm-%d", req.Uid)
	// size := 4096 // This should be the size of your shared memory segment
	mu.RLock()
	size, ok := s.objectSizes[req.Uid]
	mu.RUnlock()
	if !ok {
		return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: nil}, errors.New("object was not in LOS")
	}

	data, err := readSharedMemory(strconv.FormatUint(req.Uid, 10), int(size))
	if err != nil {
		LocalLog("got err from read: %v", err)
		return nil, err
	}

	return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: data}, nil

	// mu.RLock()

	// data, ok := s.localObjectStore[req.Uid]
	// mu.RUnlock()
	// if !ok {
	// 	return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: nil}, errors.New("object was not in LOS")
	// }
	// return &pb.CopyResponse{Uid: req.Uid, ObjectBytes: data}, nil
}

func(s *server) AvgBandwidth(ctx context.Context, req *pb.StatusResponse) (*pb.BandwidthResponse, error) {
	return &pb.BandwidthResponse{AvgBandwidth: s.avgBandwidth}, nil
}