package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/bitcode-framework/bitcode/internal/domain/event"
)

type Message struct {
	Type    string `json:"type"`
	Channel string `json:"channel"`
	Data    any    `json:"data"`
}

type Client struct {
	Conn     *websocket.Conn
	Channels map[string]bool
	UserID   string
	TenantID string
	mu       sync.Mutex
}

func (c *Client) Send(msg Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	c.Conn.WriteMessage(websocket.TextMessage, data)
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS] client connected (total: %d)", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.Conn.Close()
			}
			h.mu.Unlock()
			log.Printf("[WS] client disconnected (total: %d)", len(h.clients))
		}
	}
}

func (h *Hub) Broadcast(channel string, data any) {
	msg := Message{Type: "event", Channel: channel, Data: data}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if len(client.Channels) == 0 || client.Channels[channel] || client.Channels["*"] {
			go client.Send(msg)
		}
	}
}

func (h *Hub) BroadcastToTenant(tenantID string, channel string, data any) {
	msg := Message{Type: "event", Channel: channel, Data: data}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.TenantID == tenantID {
			if len(client.Channels) == 0 || client.Channels[channel] || client.Channels["*"] {
				go client.Send(msg)
			}
		}
	}
}

func (h *Hub) ConnectToEventBus(bus *event.Bus) {
	bus.SubscribeAll(func(ctx context.Context, eventName string, data map[string]any) error {
		h.Broadcast(eventName, data)
		return nil
	})
}

func (h *Hub) RegisterRoutes(app *fiber.App) {
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		client := &Client{
			Conn:     c,
			Channels: make(map[string]bool),
			UserID:   c.Query("user_id"),
			TenantID: c.Query("tenant_id"),
		}

		h.register <- client
		defer func() { h.unregister <- client }()

		for {
			_, msgBytes, err := c.ReadMessage()
			if err != nil {
				break
			}

			var msg Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				continue
			}

			switch msg.Type {
			case "subscribe":
				client.Channels[msg.Channel] = true
				client.Send(Message{Type: "subscribed", Channel: msg.Channel})
			case "unsubscribe":
				delete(client.Channels, msg.Channel)
				client.Send(Message{Type: "unsubscribed", Channel: msg.Channel})
			case "ping":
				client.Send(Message{Type: "pong"})
			}
		}
	}))
}
