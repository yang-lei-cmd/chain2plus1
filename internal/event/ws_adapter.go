package event

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WSHandler is an adapter that delegates to handler.WSHandler for WebSocket connection management.
// It embeds a callback-based auth function for flexible token verification.
type WSHandler struct {
	hub      *Hub
	authFunc func(tokenString string) (uint, string, string, error)
}

// NewWSHandler creates a new WSHandler bound to the given Hub.
func NewWSHandler(hub *Hub) *WSHandler {
	return &WSHandler{hub: hub}
}

// SetAuthFunc sets the authentication callback used to verify WS tokens.
func (h *WSHandler) SetAuthFunc(fn func(tokenString string) (uint, string, string, error)) {
	h.authFunc = fn
}

// HandleConnection performs WebSocket upgrade + auth and manages the connection lifecycle.
func (h *WSHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, `{"error":"missing token"}`, http.StatusBadRequest)
		return
	}

	if h.authFunc == nil {
		http.Error(w, `{"error":"auth not configured"}`, http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	userID, username, _, err := h.authFunc(tokenStr)
	if err != nil {
		conn.Close()
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return
	}

	client := &Client{
		ID:     username + "-" + string(rune('0'+userID)),
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.hub.RegisterChan() <- client

	done := make(chan struct{})
	go wsReadLoop(conn, done)
	wsWriteLoop(conn, client, done)
	<-done

	h.hub.UnregisterChan() <- client
	conn.Close()
}

// Hub returns the underlying Hub reference for router health checks.
func (h *WSHandler) Hub() *Hub {
	return h.hub
}

func wsReadLoop(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		select {
		case <-done:
			return
		default:
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Printf("[WS] Read error: %v", err)
				}
				return
			}
		}
	}
}

func wsWriteLoop(conn *websocket.Conn, client *Client, done chan struct{}) {
	defer func() {
		conn.Close()
		close(done)
	}()

	for {
		select {
		case msg := <-client.Send:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("[WS] Write error for userID=%d: %v", client.UserID, err)
				return
			}
		case <-done:
			return
		}
	}
}
