package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var ctx = context.Background()

const timeout = 60 * time.Second

// endpoint
type ept struct {
	out  chan<- string
	name string
}

// ChatRoom struct
type ChatRoom struct {
	id       uuid.UUID
	rdb      *redis.Client
	pubSub   *redis.PubSub
	epts     map[ept]bool
	entering chan ept
	leaving  chan ept
}

// Create return a ChatRoom instance
func Create(uuid uuid.UUID) *ChatRoom {
	// TODO user config to set connect
	rdb := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	pubSub := rdb.Subscribe(ctx, fmt.Sprintf("%s-msg", uuid))
	return &ChatRoom{
		uuid,
		rdb,
		pubSub,
		make(map[ept]bool),
		make(chan ept),
		make(chan ept),
	}
}

// PublishMsg puslish msg to redis
func (r *ChatRoom) PublishMsg(msg string) {
	msg = fmt.Sprintf("(%s) %s", r.id.String(), msg)
	r.rdb.Publish(ctx, fmt.Sprintf("%s-msg", r.id.String()), msg)
}

// Broadcaster select all channel ChatRoom received
func (r *ChatRoom) Broadcaster() {
	for {
		select {
		case msg, ok := <-r.pubSub.Channel():
			if !ok {
				log.Fatal("pubSub Channel failed")
				break
			}

			for ept := range r.epts {
				select {
				case ept.out <- msg.Payload:
				default:
					// Skip client if it's reading messages slowly.
				}
			}
		case ept := <-r.entering:
			r.epts[ept] = true
			ept.out <- "Present:"
			for e := range r.epts {
				ept.out <- e.name
			}
		case ept := <-r.leaving:
			delete(r.epts, ept)
			close(ept.out)
		}
	}
}

// HandleConn Handle the connecting
func (r *ChatRoom) HandleConn(conn net.Conn) {
	out := make(chan string, 10) // outgoing client messages
	go clientWriter(conn, out)
	in := make(chan string) // incoming client messages
	go clientReader(conn, in)

	input := bufio.NewScanner(conn)
	var who string
	if input.Scan() {
		who = input.Text()
	}
	ept := ept{out, who}
	out <- "You are " + who
	r.PublishMsg(who + " has arrived")
	r.entering <- ept
	idle := time.NewTimer(timeout)

Loop:
	for {
		select {
		case msg := <-in:
			r.PublishMsg(who + ": " + msg)
			idle.Reset(timeout)
		case <-idle.C:
			conn.Close()
			break Loop
		}
	}

	r.leaving <- ept
	r.PublishMsg(who + " has left")
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
