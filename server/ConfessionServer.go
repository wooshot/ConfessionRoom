package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/wooshot/ConfessionRoom/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/runtime/protoiface"
)

// ConfessionServer ...
type ConfessionServer struct {
	wg   sync.WaitGroup
	room *ChatRoom
}

// New creates new server greeter
func New() *ConfessionServer {
	return &ConfessionServer{room: Create()}
}

// Start starts server
func (c *ConfessionServer) Start() {
	c.wg.Add(1)
	go func() {
		log.Fatal(c.startGRPC())
		c.wg.Done()
	}()
	c.wg.Add(1)
	go func() {
		log.Fatal(c.startREST())
		c.wg.Done()
	}()
	c.wg.Add(1)
	go func() {
		log.Fatal(c.startCharServer())
		c.wg.Done()
	}()
	c.wg.Wait()

}

func (c *ConfessionServer) startGRPC() error {
	lis, err := net.Listen("tcp", "localhost:8091")
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterConfessionServer(grpcServer, c)
	grpcServer.Serve(lis)
	return nil
}

func restHandler(ctx context.Context, w http.ResponseWriter, resp protoiface.MessageV1) error {
	// allow cross domain AJAX requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, PATCH, OPTIONS")
	return nil
}

func (c *ConfessionServer) startREST() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithForwardResponseOption(restHandler),
	)
	opts := []grpc.DialOption{grpc.WithInsecure()}
	err := pb.RegisterConfessionHandlerFromEndpoint(ctx, mux, ":8091", opts)
	if err != nil {
		return err
	}

	return http.ListenAndServe(":8090", mux)
}

func (c *ConfessionServer) startCharServer() error {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	go c.room.Broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go c.room.HandleConn(conn)
	}
}

// HealthCheck ...
func (c *ConfessionServer) HealthCheck(ctx context.Context, r *pb.Empty) (*pb.HealthCheckReply, error) {
	return &pb.HealthCheckReply{
		Message: fmt.Sprintf("Health check ok"),
	}, nil
}
