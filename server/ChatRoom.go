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
}

// CreateRoom return a ChatRoom instance
func CreateRoom(uuid uuid.UUID, rdb *redis.Client) *ChatRoom {
	channels := []string{
		fmt.Sprintf("%s-msg", uuid),
		fmt.Sprintf("%s-entering", uuid),
		fmt.Sprintf("%s-leaving", uuid),
	}
	ps := rdb.Subscribe(ctx, channels...)
	return &ChatRoom{
		uuid,
		make(map[ept]bool),
		ps,
		make(chan ept),
		make(chan ept),
	}
}

// PublishMsg puslish msg to redis
func (r *ChatRoom) PublishMsg(clt *redis.Client, msg string) {
	msg = fmt.Sprintf("(%s) %s", r.id.String(), msg)
	clt.Publish(ctx, fmt.Sprintf("%s-msg", r.id.String()), msg)
}

// PublishEntering puslish entering to redis
func (r *ChatRoom) PublishEntering(clt *redis.Client, msg string) {
	msg = fmt.Sprintf("(%s) %s", r.id.String(), msg)
	clt.Publish(ctx, fmt.Sprintf("%s-entering", r.id.String()), msg)
}

// PublishLeaving puslish leaving to redis
func (r *ChatRoom) PublishLeaving(clt *redis.Client, msg string) {
	msg = fmt.Sprintf("(%s) %s", r.id.String(), msg)
	clt.Publish(ctx, fmt.Sprintf("%s-leaving", r.id.String()), msg)
}

// Run select all channel ChatRoom received
func (r *ChatRoom) Run() {
Loop:
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
			if len(r.epts) == 0 {
				break Loop
			}
		}
	}
}
