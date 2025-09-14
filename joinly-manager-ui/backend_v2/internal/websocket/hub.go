package websocket

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"joinly-manager/internal/models"
)

// Hub manages WebSocket connections
type Hub struct {
	clients        map[*Client]bool
	clientsByAgent map[string]map[*Client]bool
	sessionClients map[*Client]bool
	broadcast      chan models.WebSocketMessage
	register       chan *Client
	unregister     chan *Client
	running        bool
	mu             sync.RWMutex
}

// Client represents a WebSocket client
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan models.WebSocketMessage
	agentID   string
	isSession bool // true for session-wide connections
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:        make(map[*Client]bool),
		clientsByAgent: make(map[string]map[*Client]bool),
		sessionClients: make(map[*Client]bool),
		broadcast:      make(chan models.WebSocketMessage, 256),
		register:       make(chan *Client, 256),
		unregister:     make(chan *Client, 256),
		running:        false,
	}
}

// Start starts the WebSocket hub
func (h *Hub) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return
	}

	h.running = true
	go h.run()
	logrus.Info("WebSocket hub started")
}

// Stop stops the WebSocket hub
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}

	h.running = false
	close(h.broadcast)
	close(h.register)
	close(h.unregister)
	logrus.Info("WebSocket hub stopped")
}

// run runs the WebSocket hub
func (h *Hub) run() {
	for {
		select {
		case client, ok := <-h.register:
			if !ok {
				// Channel closed, exit
				return
			}
			if client == nil {
				// Nil client received, skip
				continue
			}
			h.mu.Lock()
			h.clients[client] = true
			if client.isSession {
				h.sessionClients[client] = true
				logrus.Debugf("WebSocket session client registered")
			} else {
				if h.clientsByAgent[client.agentID] == nil {
					h.clientsByAgent[client.agentID] = make(map[*Client]bool)
				}
				h.clientsByAgent[client.agentID][client] = true
				logrus.Debugf("WebSocket client registered for agent %s", client.agentID)
			}
			h.mu.Unlock()

		case client, ok := <-h.unregister:
			if !ok {
				// Channel closed, exit
				return
			}
			if client == nil {
				// Nil client received, skip
				continue
			}
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			if client.isSession {
				delete(h.sessionClients, client)
				logrus.Debugf("WebSocket session client unregistered")
			} else {
				if agentClients, ok := h.clientsByAgent[client.agentID]; ok {
					delete(agentClients, client)
					if len(agentClients) == 0 {
						delete(h.clientsByAgent, client.agentID)
					}
				}
				logrus.Debugf("WebSocket client unregistered for agent %s", client.agentID)
			}
			h.mu.Unlock()

		case message, ok := <-h.broadcast:
			if !ok {
				// Channel closed, exit
				return
			}
			h.mu.RLock()
			// Send to agent-specific clients
			if agentClients, ok := h.clientsByAgent[message.AgentID]; ok {
				for client := range agentClients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
						delete(agentClients, client)
					}
				}
			}
			// Send to session clients (they get all messages)
			for client := range h.sessionClients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
					delete(h.sessionClients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToAgent broadcasts a message to all clients listening to a specific agent
func (h *Hub) BroadcastToAgent(agentID string, message models.WebSocketMessage) {
	if !h.running {
		return
	}

	select {
	case h.broadcast <- message:
	default:
		logrus.Warn("WebSocket broadcast channel full, dropping message")
	}
}

// Broadcast broadcasts a message to all clients
func (h *Hub) Broadcast(message models.WebSocketMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// ServeWs handles WebSocket connections
func (h *Hub) ServeWs(c *gin.Context, agentID string) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from the frontend
			origin := r.Header.Get("Origin")
			return origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000"
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade connection to WebSocket: %v", err)
		return
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan models.WebSocketMessage, 256),
		agentID:   agentID,
		isSession: false,
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// ServeSessionWs handles WebSocket connections for entire user session
func (h *Hub) ServeSessionWs(c *gin.Context) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from the frontend
			origin := r.Header.Get("Origin")
			return origin == "http://localhost:3000" || origin == "http://127.0.0.1:3000"
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logrus.Errorf("Failed to upgrade connection to WebSocket: %v", err)
		return
	}

	client := &Client{
		hub:       h,
		conn:      conn,
		send:      make(chan models.WebSocketMessage, 256),
		agentID:   "", // Empty for session clients
		isSession: true,
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				logrus.Errorf("Failed to write WebSocket message: %v", err)
				return
			}
		}
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.Errorf("WebSocket error: %v", err)
			}
			break
		}
		// For now, we don't handle incoming messages from clients
		// This could be extended to handle client commands in the future
	}
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetAgentClientCount returns the number of clients connected to a specific agent
func (h *Hub) GetAgentClientCount(agentID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if agentClients, ok := h.clientsByAgent[agentID]; ok {
		return len(agentClients)
	}
	return 0
}
