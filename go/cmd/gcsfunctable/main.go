package main

import (
    "context"
    "log"
    "net"
    "strconv"
    "sync"
    "crypto/rand"
	"encoding/binary"

    "google.golang.org/grpc"
    pb "github.com/rodrigo-castellon/babyray/pkg"
    "github.com/rodrigo-castellon/babyray/config"

    "google.golang.org/grpc/status"
    "google.golang.org/grpc/codes"
)

func main() {
    cfg := config.GetConfig() // Load configuration
    address := ":" + strconv.Itoa(cfg.Ports.GCSFunctionTable) // Prepare the network address

    lis, err := net.Listen("tcp", address)
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    _ = lis;
    s := grpc.NewServer()
    pb.RegisterGCSFuncServer(s, NewGCSFuncServer())
    log.Printf("server listening at %v", lis.Addr())
    if err := s.Serve(lis); err != nil {
       log.Fatalf("failed to serve: %v", err)
    }
}

// server is used to implement your gRPC service.
type GCSFuncServer struct {
   pb.UnimplementedGCSFuncServer
   functionStore map[uint64][]byte
   mu            sync.Mutex // Use a mutex to manage concurrent access
}

func NewGCSFuncServer() *GCSFuncServer {
    return &GCSFuncServer{
        functionStore: make(map[uint64][]byte),
    }
}

// Note: This is cryptographically random, whereas basic random might be sufficient for our needs
// If this slows things down, change the implementation to basic rangom
func generateUID() (uint64, error) {
	var n uint64
	err := binary.Read(rand.Reader, binary.BigEndian, &n)
	if err != nil {
		log.Println("Error generating random number:", err)
		return 0, err  // Return 0 and the error
	}
	return n, nil  // Return the generated number and nil for the error
}

// Implement your service methods here.
func (s *GCSFuncServer) RegisterFunc(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
    // Lock and unlock the mutex to handle concurrent writes safely
    s.mu.Lock()
    defer s.mu.Unlock()

    // Generate UID for this newly registered function
    uid, err := generateUID()
	if err != nil {
		log.Println("Failed to generate UID:", err)
        return nil, status.Errorf(codes.Internal, "Internal failed to generate UID")
	}

    // Logic to register the function in the server's function store
    s.functionStore[uid] = req.SerializedFunc
    
    return &pb.RegisterResponse{Name: uid}, nil
}

func (s *GCSFuncServer) FetchFunc(ctx context.Context, req *pb.FetchRequest) (*pb.FetchResponse, error) {
    s.mu.Lock()
    serializedFunc, ok := s.functionStore[req.Name]
    s.mu.Unlock()
    
    if !ok {
        return nil, status.Errorf(codes.NotFound, "function not found")
    }
    
    return &pb.FetchResponse{SerializedFunc: serializedFunc}, nil
}