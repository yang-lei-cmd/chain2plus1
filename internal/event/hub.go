package event

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a single WebSocket client connection.
type Client struct {
	ID       string        `json:"id"`
	UserID   uint          `json:"user_id"`
	Conn     *websocket.Conn `json:"-"`
	Send     chan []byte   `json:"-"`
	mu       sync.Mutex    `json:"-"`
	LastSeen time.Time     `json:"-"`
}

// Hub manages all connected WebSocket clients and broadcasts events.
type Hub struct {
	clients    map[uint]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *BroadcastMessage
	mu         sync.RWMutex
}

// BroadcastMessage represents an event to be sent to one or more clients.
type BroadcastMessage struct {
	TargetUserID uint             `json:"target_user_id"`
	Payload      NotificationPayload `json:"payload"`
}

// RegisterChan returns the register channel for external access.
func (h *Hub) RegisterChan() chan<- *Client {
	return h.register
}

// UnregisterChan returns the unregister channel for external access.
func (h *Hub) UnregisterChan() chan<- *Client {
	return h.unregister
}

// NewHub creates a new Hub instance.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run starts the Hub event loop. Block until the channel is closed.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("[WS] Client registered: userID=%d", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.UserID]; ok {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[WS] Client unregistered: userID=%d", client.UserID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			if msg.TargetUserID == 0 {
				for _, c := range h.clients {
					c.sendOnce(msg.Payload)
				}
			} else {
				if c, ok := h.clients[msg.TargetUserID]; ok {
					c.sendOnce(msg.Payload)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToAll sends a notification to all connected clients.
func (h *Hub) BroadcastToAll(payload NotificationPayload) {
	h.broadcast <- &BroadcastMessage{
		TargetUserID: 0,
		Payload:      payload,
	}
}

// BroadcastToUser sends a notification to a specific user if connected.
func (h *Hub) BroadcastToUser(userID uint, payload NotificationPayload) {
	h.broadcast <- &BroadcastMessage{
		TargetUserID: userID,
		Payload:      payload,
	}
}

// ClientsCount returns the number of connected clients.
func (h *Hub) ClientsCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// MarshalPayload marshals a NotificationPayload to JSON bytes.
func MarshalPayload(p NotificationPayload) []byte {
	data, err := json.Marshal(p)
	if err != nil {
		log.Printf("[WS] Failed to marshal payload: %v", err)
		return nil
	}
	return data
}

// sendOnce sends data to the client's send channel without blocking.
func (c *Client) sendOnce(data NotificationPayload) {
	c.mu.Lock()
	defer c.mu.Unlock()

	encoded := MarshalPayload(data)
	if encoded == nil {
		return
	}

	select {
	case c.Send <- encoded:
	default:
		log.Printf("[WS] Send channel full for client userID=%d", c.UserID)
	}
}
