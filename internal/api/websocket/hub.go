// Package websocket provides a WebSocket hub for pushing real-time updates
// to connected clients. Clients can subscribe to channels ("entities",
// "actions", "sync") and will only receive events matching their subscriptions.
package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gantrydev/gantry/internal/events"
	ws "github.com/gorilla/websocket"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// pingPeriod sends pings at this interval. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum message size allowed from the peer.
	maxMessageSize = 4096
)

// upgrader configures the WebSocket upgrade with permissive origin checks.
// In production, CheckOrigin should be restricted to allowed origins.
var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// channelForEvent maps event types to subscription channels.
var channelForEvent = map[events.EventType]string{
	events.EntityCreated:   "entities",
	events.EntityUpdated:   "entities",
	events.EntityDeleted:   "entities",
	events.ActionTriggered: "actions",
	events.ActionCompleted: "actions",
	events.ActionFailed:    "actions",
}

// clientMessage represents a JSON message received from a WebSocket client.
type clientMessage struct {
	Action  string `json:"action"`  // "subscribe" or "unsubscribe"
	Channel string `json:"channel"` // "entities", "actions", "sync"
}

// Client represents a single WebSocket connection to the hub.
type Client struct {
	hub           *Hub
	conn          *ws.Conn
	send          chan []byte
	subscriptions map[string]bool
	mu            sync.RWMutex
}

// isSubscribed reports whether the client is subscribed to the given channel.
func (c *Client) isSubscribed(channel string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.subscriptions[channel]
}

// subscribe adds a channel subscription for this client.
func (c *Client) subscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subscriptions[channel] = true
}

// unsubscribe removes a channel subscription for this client.
func (c *Client) unsubscribe(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.subscriptions, channel)
}

// readPump reads messages from the WebSocket connection.
// It handles subscribe/unsubscribe commands and enforces read deadlines.
func (c *Client) readPump() {
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
			if ws.IsUnexpectedCloseError(err, ws.CloseGoingAway, ws.CloseNormalClosure) {
				log.Printf("websocket: unexpected close: %v", err)
			}
			break
		}

		var msg clientMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("websocket: invalid message from client: %v", err)
			continue
		}

		switch msg.Action {
		case "subscribe":
			c.subscribe(msg.Channel)
		case "unsubscribe":
			c.unsubscribe(msg.Channel)
		default:
			log.Printf("websocket: unknown action %q from client", msg.Action)
		}
	}
}

// writePump writes messages from the send channel to the WebSocket connection.
// It sends periodic pings to detect dead connections.
func (c *Client) writePump() {
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
				// Hub closed the channel.
				c.conn.WriteMessage(ws.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(ws.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Drain any queued messages into the current write.
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte("\n"))
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(ws.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Hub maintains the set of active WebSocket clients and broadcasts
// messages to clients based on their channel subscriptions.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new WebSocket hub. Call Run() in a goroutine to start
// processing client registrations and broadcasts.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run is the main event loop for the hub. It processes client registrations,
// unregistrations, and broadcast messages. It should be run in its own goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client send buffer is full; disconnect it.
					go func(c *Client) {
						h.mu.Lock()
						if _, ok := h.clients[c]; ok {
							delete(h.clients, c)
							close(c.send)
						}
						h.mu.Unlock()
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ServeWS upgrades an HTTP connection to a WebSocket and registers the
// resulting client with the hub. It should be used as an http.HandlerFunc.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket: upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:           h,
		conn:          conn,
		send:          make(chan []byte, 256),
		subscriptions: make(map[string]bool),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// Broadcast serializes an event to JSON and sends it to all connected clients
// that are subscribed to the matching channel. Events that do not map to a
// known channel are delivered to all clients.
func (h *Hub) Broadcast(event events.Event) {
	// Wrap the event with channel information for client-side filtering.
	channel, hasChannel := channelForEvent[event.Type]

	payload := struct {
		Channel string       `json:"channel"`
		Event   events.Event `json:"event"`
	}{
		Channel: channel,
		Event:   event,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("websocket: marshal event: %v", err)
		return
	}

	if !hasChannel {
		// No specific channel mapping; broadcast to everyone.
		h.broadcast <- data
		return
	}

	// Send only to clients subscribed to the relevant channel.
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.isSubscribed(channel) {
			select {
			case client.send <- data:
			default:
				// Client send buffer full; schedule disconnect.
				go func(c *Client) {
					h.mu.Lock()
					if _, ok := h.clients[c]; ok {
						delete(h.clients, c)
						close(c.send)
					}
					h.mu.Unlock()
				}(client)
			}
		}
	}
}
