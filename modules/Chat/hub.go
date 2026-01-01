package chat

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub maintains the set of active clients and broadcasts messages to clients
type Hub struct {
	// Registered clients by group ID
	clients map[uint]map[*Client]bool

	// Inbound messages from the clients
	broadcast chan *MessageBroadcast

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// MessageBroadcast represents a message to be broadcast
type MessageBroadcast struct {
	GroupID  uint   `json:"group_id"`
	Message  Message `json:"message"`
	SenderID uint   `json:"sender_id"`
}

// Client is a middleman between the websocket connection and the hub
type Client struct {
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Group ID this client is connected to
	groupID uint

	// User ID of the client
	userID uint
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uint]map[*Client]bool),
		broadcast:  make(chan *MessageBroadcast),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.groupID] == nil {
				h.clients[client.groupID] = make(map[*Client]bool)
			}
			h.clients[client.groupID][client] = true
			h.mu.Unlock()
			log.Printf("Client registered for group %d (user %d). Total clients: %d", 
				client.groupID, client.userID, len(h.clients[client.groupID]))

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.groupID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.groupID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered for group %d (user %d)", client.groupID, client.userID)

		case message := <-h.broadcast:
			h.mu.RLock()
			clients, ok := h.clients[message.GroupID]
			h.mu.RUnlock()
			
			if ok {
				// Marshal message to JSON
				messageJSON, err := json.Marshal(message.Message)
				if err != nil {
					log.Printf("Error marshaling message: %v", err)
					continue
				}

				// Broadcast to all clients in the group
				for client := range clients {
					select {
					case client.send <- messageJSON:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
		}
	}
}

// BroadcastMessage broadcasts a message to all clients in a group
func (h *Hub) BroadcastMessage(groupID uint, message Message) {
	h.broadcast <- &MessageBroadcast{
		GroupID:  groupID,
		Message:  message,
		SenderID: message.SenderID,
	}
}












