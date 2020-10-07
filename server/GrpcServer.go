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

type GrpcServer struct {
	wg sync.WaitGroup
}

func NewGrpcServer() *GrpcServer {
	return &GrpcServer{}
}

func (g *GrpcServer) Start() {
	g.wg.Add(1)
	go func() {
		log.Fatal(g.startGRPC())
		g.wg.Done()
	}()
	g.wg.Add(1)
	go func() {
		log.Fatal(g.startREST())
		g.wg.Done()
	}()
	g.wg.Wait()
}

func (g *GrpcServer) startGRPC() error {
	lis, err := net.Listen("tcp", "localhost:8091")
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterConfessionServer(grpcServer, g)
	grpcServer.Serve(lis)
	return nil
}

func restHandler(ctx context.Context, w http.ResponseWriter, resp protoiface.MessageV1) error {
	// allow cross domain AJAX requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, PATCH, OPTIONS")
	return nil
}

func (g *GrpcServer) startREST() error {
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

// HealthCheck ...
func (g *GrpcServer) HealthCheck(ctx context.Context, r *pb.Empty) (*pb.HealthCheckReply, error) {
	return &pb.HealthCheckReply{
		Message: fmt.Sprintf("Health check ok"),
	}, nil
}

