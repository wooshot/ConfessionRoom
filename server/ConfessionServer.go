package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
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

// NewConfessionServer creates new server greeter
func NewConfessionServer() *ConfessionServer {
	// TODO user config to set connect
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	rs := make(map[uuid.UUID]*ChatRoom)

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
		log.Fatal(c.startCharServer())
		c.wg.Done()
	}()
	c.wg.Wait()
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
	done := make(chan struct{})
	go clientReader(conn, in, done)

	input := bufio.NewScanner(conn)
	var who string
	var roomID uuid.UUID

	// dispatch room and name
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
		room := CreateRoom(roomID, c.rdb)
		go room.Run()
		c.rooms[roomID] = room
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
		case <-done:
			conn.Close()
			break Loop
		}
	}

	log.Println(who + " leaving!!!")
	c.rooms[roomID].PublishMsg(c.rdb, who+" has left")
	c.rooms[roomID].leaving <- ept
	if len(c.rooms[roomID].epts) == 0 {
		c.rooms[roomID].Close()
		delete(c.rooms, roomID)
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, conn.LocalAddr().String()+" "+msg) // NOTE: ignoring network errors
	}
}

func clientReader(conn net.Conn, ch chan<- string, done chan struct{}) {
	input := bufio.NewScanner(conn)
	for input.Scan() {
		ch <- input.Text()
	}
	done <- struct{}{}
	// NOTE: ignoring potential errors from input.Err()
}
