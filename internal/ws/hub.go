package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development; restrict in production
		return true
	},
}

// Subscription represents a client subscription to a channel.
type Subscription struct {
	Client  *Client
	Channel string
}

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Channel subscriptions: channel -> set of clients
	channels map[string]map[*Client]bool

	// Broadcast to all clients on a specific channel
	broadcast chan *ChannelMessage

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Subscribe to a channel
	subscribe chan *Subscription

	// Unsubscribe from a channel
	unsubscribe chan *Subscription

	mu sync.RWMutex
}

// ChannelMessage is a message to be broadcast to a channel.
type ChannelMessage struct {
	Channel string
	Data    []byte
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		channels:    make(map[string]map[*Client]bool),
		broadcast:   make(chan *ChannelMessage, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		subscribe:   make(chan *Subscription),
		unsubscribe: make(chan *Subscription),
	}
}

// Run starts the hub's main event loop.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client connected. Total clients: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)

				// Remove from all channel subscriptions
				for channel, clients := range h.channels {
					delete(clients, client)
					if len(clients) == 0 {
						delete(h.channels, channel)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(h.clients))

		case sub := <-h.subscribe:
			h.mu.Lock()
			if h.channels[sub.Channel] == nil {
				h.channels[sub.Channel] = make(map[*Client]bool)
			}
			h.channels[sub.Channel][sub.Client] = true
			sub.Client.Subscribe(sub.Channel)
			h.mu.Unlock()
			log.Printf("Client subscribed to channel: %s", sub.Channel)

		case sub := <-h.unsubscribe:
			h.mu.Lock()
			if clients, ok := h.channels[sub.Channel]; ok {
				delete(clients, sub.Client)
				if len(clients) == 0 {
					delete(h.channels, sub.Channel)
				}
			}
			sub.Client.Unsubscribe(sub.Channel)
			h.mu.Unlock()
			log.Printf("Client unsubscribed from channel: %s", sub.Channel)

		case msg := <-h.broadcast:
			h.mu.RLock()
			if clients, ok := h.channels[msg.Channel]; ok {
				for client := range clients {
					select {
					case client.send <- msg.Data:
					default:
						// Client buffer full, skip
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Subscribe adds a client to a channel.
func (h *Hub) Subscribe(client *Client, channel string) {
	h.subscribe <- &Subscription{Client: client, Channel: channel}
}

// Unsubscribe removes a client from a channel.
func (h *Hub) Unsubscribe(client *Client, channel string) {
	h.unsubscribe <- &Subscription{Client: client, Channel: channel}
}

// Broadcast sends a message to all clients subscribed to a channel.
func (h *Hub) Broadcast(channel string, msgType string, payload interface{}) error {
	msg, err := NewChannelMessage(msgType, channel, payload)
	if err != nil {
		return err
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	h.broadcast <- &ChannelMessage{Channel: channel, Data: data}
	return nil
}

// BroadcastAll sends a message to all connected clients.
func (h *Hub) BroadcastAll(msgType string, payload interface{}) error {
	msg, err := NewMessage(msgType, payload)
	if err != nil {
		return err
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			// Client buffer full, skip
		}
	}
	return nil
}

// BroadcastDetection broadcasts a detection to the detections channel.
func (h *Hub) BroadcastDetection(detection *DetectionPayload) error {
	return h.Broadcast(ChannelDetections, TypeDetection, detection)
}

// BroadcastStatus broadcasts a status update to the status channel.
func (h *Hub) BroadcastStatus(status *StatusPayload) error {
	return h.Broadcast(ChannelStatus, TypeStatus, status)
}

// GetClientCount returns the number of connected clients.
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetChannelClientCount returns the number of clients subscribed to a channel.
func (h *Hub) GetChannelClientCount(channel string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.channels[channel]; ok {
		return len(clients)
	}
	return 0
}

// HandleWebSocket handles WebSocket upgrade requests.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := NewClient(h, conn)
	h.register <- client

	// Auto-subscribe to default channels
	h.Subscribe(client, ChannelDetections)
	h.Subscribe(client, ChannelStatus)

	// Start client goroutines
	go client.WritePump()
	go client.ReadPump()
}
