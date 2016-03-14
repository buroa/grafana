package live

import (
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/log"
)

type hub struct {
	connections map[*connection]bool
	streams     map[string]map[*connection]bool

	register      chan *connection
	unregister    chan *connection
	streamChannel chan *dtos.StreamMessage
	subChannel    chan *streamSubscription
}

type streamSubscription struct {
	conn *connection
	name string
}

var h = hub{
	connections:   make(map[*connection]bool),
	streams:       make(map[string]map[*connection]bool),
	register:      make(chan *connection),
	unregister:    make(chan *connection),
	streamChannel: make(chan *dtos.StreamMessage),
	subChannel:    make(chan *streamSubscription),
}

func (h *hub) removeConnection() {

}

func (h *hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
			log.Info("Live: New connection (Total count: %v)", len(h.connections))
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.send)
			}
		// hand stream subscriptions
		case sub := <-h.subChannel:
			log.Info("Live: Connection subscribing to: %v", sub.name)
			subscribers, exists := h.streams[sub.name]
			if !exists {
				subscribers = make(map[*connection]bool)
				h.streams[sub.name] = subscribers
			}
			subscribers[sub.conn] = true

			// handle stream messages
		case message := <-h.streamChannel:
			subscribers, exists := h.streams[message.Stream]
			if !exists {
				log.Info("Live: Message to stream without subscribers: %v", message.Stream)
				continue
			}

			messageBytes, _ := simplejson.NewFromAny(message).Encode()
			for sub := range subscribers {
				select {
				case sub.send <- messageBytes:
				default:
					close(sub.send)
					delete(h.connections, sub)
					delete(subscribers, sub)
				}
			}
		}
	}
}
