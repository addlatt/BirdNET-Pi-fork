package ws

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512KB
)

// Client represents a WebSocket client connection.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	// Channel subscriptions
	channels map[string]bool
	mu       sync.RWMutex
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		channels: make(map[string]bool),
	}
}

// IsSubscribed checks if the client is subscribed to a channel.
func (c *Client) IsSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.channels[channel]
}

// Subscribe adds the client to a channel.
func (c *Client) Subscribe(channel string) {
	c.mu.Lock()
	c.channels[channel] = true
	c.mu.Unlock()
}

// Unsubscribe removes the client from a channel.
func (c *Client) Unsubscribe(channel string) {
	c.mu.Lock()
	delete(c.channels, channel)
	c.mu.Unlock()
}

// GetChannels returns a copy of the client's subscribed channels.
func (c *Client) GetChannels() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	channels := make([]string, 0, len(c.channels))
	for ch := range c.channels {
		channels = append(channels, ch)
	}
	return channels
}

// ReadPump pumps messages from the WebSocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Parse the message
		var msg WSMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse WebSocket message: %v", err)
			continue
		}

		// Handle the message
		c.handleMessage(&msg)
	}
}

// handleMessage processes incoming WebSocket messages.
func (c *Client) handleMessage(msg *WSMessage) {
	switch msg.Type {
	case TypeSubscribe:
		var payload SubscribePayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse subscribe payload: %v", err)
			return
		}
		for _, channel := range payload.Channels {
			c.hub.subscribe <- &Subscription{Client: c, Channel: channel}
		}

	case TypeUnsubscribe:
		var payload SubscribePayload
		if err := msg.ParsePayload(&payload); err != nil {
			log.Printf("Failed to parse unsubscribe payload: %v", err)
			return
		}
		for _, channel := range payload.Channels {
			c.hub.unsubscribe <- &Subscription{Client: c, Channel: channel}
		}

	case TypePing:
		// Respond with pong
		pongMsg, _ := NewMessage(TypePong, nil)
		data, _ := json.Marshal(pongMsg)
		select {
		case c.send <- data:
		default:
		}
	}
}

// WritePump pumps messages from the hub to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
