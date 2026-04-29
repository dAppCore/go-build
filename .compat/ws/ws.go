package ws

import (
	"context"
	"net/http"
	"sync"
	"time"

	core "dappco.re/go"
	"github.com/gorilla/websocket"
)

type MessageType string

const (
	TypeEvent     MessageType = "event"
	TypeSubscribe MessageType = "subscribe"
)

type Message struct {
	Type      MessageType `json:"type"`
	Channel   string      `json:"channel,omitempty"`
	ProcessID string      `json:"processId,omitempty"`
	Data      any         `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp,omitempty"`
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[*Client]bool
	channels map[string]map[*Client]bool
	done     chan struct{}
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Message
}

func NewHub() *Hub {
	return &Hub{
		clients:  make(map[*Client]bool),
		channels: make(map[string]map[*Client]bool),
		done:     make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	if h == nil {
		return
	}
	<-ctx.Done()
	close(h.done)
	h.mu.Lock()
	for client := range h.clients {
		if err := client.conn.Close(); err != nil {
			continue
		}
		close(client.send)
	}
	h.clients = make(map[*Client]bool)
	h.channels = make(map[string]map[*Client]bool)
	h.mu.Unlock()
}

func (h *Hub) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.HandleWebSocket(w, r)
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	if h == nil {
		http.Error(w, "hub unavailable", http.StatusServiceUnavailable)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &Client{hub: h, conn: conn, send: make(chan Message, 32)}
	h.mu.Lock()
	h.clients[client] = true
	h.mu.Unlock()
	go client.writeLoop()
	client.readLoop()
	h.removeClient(client)
}

var upgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func (c *Client) readLoop() {
	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg.Type == TypeSubscribe {
			if channel, ok := msg.Data.(string); ok {
				if subscribed := c.hub.Subscribe(c, channel); !subscribed.OK {
					return
				}
			}
		}
	}
}

func (c *Client) writeLoop() {
	for msg := range c.send {
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now().UTC()
		}
		if err := c.conn.WriteJSON(msg); err != nil {
			return
		}
	}
}

func (h *Hub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	for channel, subscribers := range h.channels {
		delete(subscribers, client)
		if len(subscribers) == 0 {
			delete(h.channels, channel)
		}
	}
	close(client.send)
	if err := client.conn.Close(); err != nil {
		return
	}
}

func (h *Hub) Subscribe(client *Client, channel string) core.Result {
	if h == nil || client == nil || channel == "" {
		return core.Fail(core.NewError("invalid subscription"))
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.channels[channel] == nil {
		h.channels[channel] = make(map[*Client]bool)
	}
	h.channels[channel][client] = true
	return core.Ok(nil)
}

func (h *Hub) SendToChannel(channel string, msg Message) core.Result {
	if h == nil {
		return core.Fail(core.NewError("hub unavailable"))
	}
	if msg.Channel == "" {
		msg.Channel = channel
	}
	h.mu.RLock()
	subscribers := make([]*Client, 0, len(h.channels[channel]))
	for client := range h.channels[channel] {
		subscribers = append(subscribers, client)
	}
	h.mu.RUnlock()
	for _, client := range subscribers {
		select {
		case client.send <- msg:
		default:
		}
	}
	return core.Ok(nil)
}

func (h *Hub) Broadcast(msg Message) core.Result {
	if h == nil {
		return core.Fail(core.NewError("hub unavailable"))
	}
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		select {
		case client.send <- msg:
		default:
		}
	}
	return core.Ok(nil)
}

func (h *Hub) ChannelSubscriberCount(channel string) int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.channels[channel])
}

func (h *Hub) ClientCount() int {
	if h == nil {
		return 0
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
