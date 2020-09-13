package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	pb "github.com/wooshot/ConfessionRoom/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/runtime/protoiface"
)

var ctx = context.Background()

const timeout = 5 * 60 * time.Second

// ConfessionServer ...
type ConfessionServer struct {
	wg       sync.WaitGroup
	rooms    map[uuid.UUID]*ChatRoom
	rdb      *redis.Client
	pubsub   *redis.PubSub
	entering chan *ChatRoom
	leaving  chan *ChatRoom
}

// New creates new server greeter
func New() *ConfessionServer {
	// TODO user config to set connect
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	rs := make(map[uuid.UUID]*ChatRoom)
	// tmp
	id, _ := uuid.Parse("a8ababb2-e2f5-4819-99e2-ca09a8ac9c05")
	r := CreateRoom(id, rdb)
	rs[id] = r
	go r.Run()

	return &ConfessionServer{
		rooms:  rs,
		rdb:    rdb,
		pubsub: rdb.Subscribe(ctx),
	}
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
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go c.HandleConn(conn)
	}
}

// HandleConn Handle the connecting
func (c *ConfessionServer) HandleConn(conn net.Conn) {
	out := make(chan string, 10) // outgoing client messages
	go clientWriter(conn, out)
	in := make(chan string) // incoming client messages
	go clientReader(conn, in)

	input := bufio.NewScanner(conn)
	var who string
	var roomID uuid.UUID

	if input.Scan() {
		params := strings.Split(input.Text(), ":")
		who = params[0]
		if id, err := uuid.Parse(params[1]); err != nil {
			log.Fatal(fmt.Sprintf("Invalid roomID: %s", params[1]))
		} else {
			roomID = id
		}
	}
	ept := ept{out, who}
	out <- "You are " + who
	if _, ok := c.rooms[roomID]; !ok {
		c.rooms[roomID] = CreateRoom(roomID, c.rdb)
		go c.rooms[roomID].Run()
	}

	c.rooms[roomID].PublishMsg(c.rdb, who+" has arrived")
	c.rooms[roomID].entering <- ept
	idle := time.NewTimer(timeout)

Loop:
	for {
		select {
		case msg := <-in:
			c.rooms[roomID].PublishMsg(c.rdb, who+": "+msg)
			idle.Reset(timeout)
		case <-idle.C:
			conn.Close()
			break Loop
		}
	}

	c.rooms[roomID].leaving <- ept
	c.rooms[roomID].PublishMsg(c.rdb, who+" has left")
	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

func clientReader(conn net.Conn, ch chan<- string) {
	input := bufio.NewScanner(conn)
	for input.Scan() {
		ch <- input.Text()
	}
	// NOTE: ignoring potential errors from input.Err()
}

// HealthCheck ...
func (c *ConfessionServer) HealthCheck(ctx context.Context, r *pb.Empty) (*pb.HealthCheckReply, error) {
	return &pb.HealthCheckReply{
		Message: fmt.Sprintf("Health check ok"),
	}, nil
}

// CreateChatRoom return uuid/roomName
func (c *ConfessionServer) CreateChatRoom(ctx context.Context, in *pb.Empty) (*pb.UUID, error) {
	id := uuid.New()

	if _, ok := c.rooms[id]; ok {
		log.Print(fmt.Sprintf("Create chat room %s failed, already existed.", id.String()))
	} else {
		c.rooms[id] = CreateRoom(id, c.rdb)

	}
	go c.rooms[id].Run()
	return &pb.UUID{Uuid: id.String()}, nil
}

// CloseChatRoom close a chatRoom by given id
func (c *ConfessionServer) CloseChatRoom(ctx context.Context, in *pb.UUID) (*pb.Empty, error) {
	// if id, err := uuid.Parse(in.Uuid); err != nil {
	// 	log.Printf("CloseChatRoom failed with UUID: %s", in.Uuid)
	// 	return nil, err
	// }
	// delete(c.rooms, id)
	return nil, nil
}
