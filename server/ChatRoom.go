package server

import (
	"fmt"
	"log"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// endpoint
type ept struct {
	out  chan<- string
	name string
}

// ChatRoom struct
type ChatRoom struct {
	id       uuid.UUID
	epts     map[ept]bool
	pubSub   *redis.PubSub
	entering chan ept
	leaving  chan ept
	close    chan struct{}
}

// CreateRoom return a ChatRoom instance
func CreateRoom(uuid uuid.UUID, rdb *redis.Client) *ChatRoom {
	ps := rdb.Subscribe(ctx, fmt.Sprintf("%s-msg", uuid))
	return &ChatRoom{
		uuid,
		make(map[ept]bool),
		ps,
		make(chan ept),
		make(chan ept),
		make(chan struct{}),
	}
}

// PublishMsg puslish msg to redis
func (r *ChatRoom) PublishMsg(clt *redis.Client, msg string) {
	msg = fmt.Sprintf("(%s) %s", r.id.String(), msg)
	clt.Publish(ctx, fmt.Sprintf("%s-msg", r.id.String()), msg)
}

// Close close this room
func (r *ChatRoom) Close() {
	r.close <- struct{}{}
}

// Run select all channel ChatRoom received
func (r *ChatRoom) Run() {
Loop:
	for {
		select {
		case msg, ok := <-r.pubSub.Channel():
			log.Println(msg.Payload, msg.Channel)
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
		case ept := <-r.leaving:
			delete(r.epts, ept)
			close(ept.out)
		case <-r.close:
			log.Println(r.id, " close the room")
			if len(r.epts) == 0 {
				break Loop
			}
		}
	}
}
