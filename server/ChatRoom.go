package server

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

const timeout = 60 * time.Second

// endpoint
type ept struct {
	out  chan<- string
	name string
}

// ChatRoom struct
type ChatRoom struct {
	epts     map[ept]bool
	entering chan ept
	leaving  chan ept
	messages chan string // incoming messages
}

// Create return a ChatRoom instance
func Create() *ChatRoom {
	return &ChatRoom{
		make(map[ept]bool),
		make(chan ept),
		make(chan ept),
		make(chan string),
	}
}

// Broadcaster select all channel ChatRoom received
func (r *ChatRoom) Broadcaster() {
	for {
		select {
		case msg := <-r.messages:
			for ept := range r.epts {
				select {
				case ept.out <- msg:
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
	r.messages <- who + " has arrived"
	r.entering <- ept
	idle := time.NewTimer(timeout)

Loop:
	for {
		select {
		case msg := <-in:
			r.messages <- who + ": " + msg
			idle.Reset(timeout)
		case <-idle.C:
			conn.Close()
			break Loop
		}
	}

	r.leaving <- ept
	r.messages <- who + " has left"
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
