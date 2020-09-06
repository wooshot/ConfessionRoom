package server

import (
	"bufio"
	"fmt"
	"net"
)

// endpoint
type ep chan<- string

// ChatRoom struct
type ChatRoom struct {
	eps      map[ep]bool
	entering chan ep
	leaving  chan ep
	messages chan string // incoming messages
}

// Create return a ChatRoom instance
func Create() *ChatRoom {
	return &ChatRoom{
		make(map[ep]bool),
		make(chan ep),
		make(chan ep),
		make(chan string),
	}
}

// Broadcaster select all channel ChatRoom received
func (r *ChatRoom) Broadcaster() {
	for {
		select {
		case msg := <-r.messages:
			for ep := range r.eps {
				ep <- msg
			}
		case ep := <-r.entering:
			r.eps[ep] = true
		case ep := <-r.leaving:
			delete(r.eps, ep)
			close(ep)
		}
	}
}

// HandleConn Handle the connecting
func (r *ChatRoom) HandleConn(conn net.Conn) {
	ch := make(chan string)
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	ch <- "You are " + who
	r.messages <- who + " has arrived"
	r.entering <- ch

	input := bufio.NewScanner(conn)
	for input.Scan() {
		r.messages <- who + ": " + input.Text()
	}
	// NOTE: ignoring potential errors from input.Err()

	r.leaving <- ch
	r.messages <- who + " has left"
	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}
