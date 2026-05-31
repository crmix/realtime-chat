package websocket

import (
	"context"
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/realtime-chat/internal/domain"
)

// Hub maintains the set of connected clients and fans out messages received
// from the domain.EventBus to clients that are subscribed to the relevant
// room. The hub does not persist anything — that responsibility lives in the
// service layer.
type Hub struct {
	bus domain.EventBus
	log *slog.Logger

	mu       sync.RWMutex
	bySocket map[*Client]struct{}
	byRoom   map[uuid.UUID]map[*Client]struct{}

	register    chan *Client
	unregister  chan *Client
	subscribe   chan roomMembership
	unsubscribe chan roomMembership

	// rootCtx is the long-lived context owning the hub. WebSocket clients
	// derive their per-connection contexts from this, NOT from the HTTP
	// request — once the upgrade succeeds the request context is cancelled.
	rootCtx context.Context
}

type roomMembership struct {
	client *Client
	roomID uuid.UUID
}

func NewHub(bus domain.EventBus, log *slog.Logger) *Hub {
	return &Hub{
		bus:         bus,
		log:         log,
		bySocket:    map[*Client]struct{}{},
		byRoom:      map[uuid.UUID]map[*Client]struct{}{},
		register:    make(chan *Client, 64),
		unregister:  make(chan *Client, 64),
		subscribe:   make(chan roomMembership, 64),
		unsubscribe: make(chan roomMembership, 64),
	}
}

// Context returns the hub's root context. WebSocket handlers should derive
// per-connection contexts from this, since http.Request.Context() is
// cancelled the moment the connection is hijacked for the WS upgrade.
func (h *Hub) Context() context.Context { return h.rootCtx }

// Run blocks until ctx is cancelled. It subscribes to the event bus and
// dispatches incoming messages to local clients.
func (h *Hub) Run(ctx context.Context) error {
	h.rootCtx = ctx
	events, err := h.bus.SubscribeMessages(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			h.closeAll()
			return nil

		case c := <-h.register:
			h.mu.Lock()
			h.bySocket[c] = struct{}{}
			h.mu.Unlock()

		case c := <-h.unregister:
			h.removeClient(c)

		case m := <-h.subscribe:
			h.mu.Lock()
			clients, ok := h.byRoom[m.roomID]
			if !ok {
				clients = map[*Client]struct{}{}
				h.byRoom[m.roomID] = clients
			}
			clients[m.client] = struct{}{}
			m.client.subscribed[m.roomID] = struct{}{}
			total := len(clients)
			h.mu.Unlock()
			h.log.Debug("hub subscribe", "room_id", m.roomID, "user_id", m.client.userID, "room_total", total)

		case m := <-h.unsubscribe:
			h.mu.Lock()
			if clients, ok := h.byRoom[m.roomID]; ok {
				delete(clients, m.client)
				if len(clients) == 0 {
					delete(h.byRoom, m.roomID)
				}
			}
			delete(m.client.subscribed, m.roomID)
			h.mu.Unlock()

		case msg, ok := <-events:
			if !ok {
				return nil
			}
			h.dispatch(msg)
		}
	}
}

func (h *Hub) dispatch(m domain.Message) {
	h.mu.RLock()
	clients := h.byRoom[m.RoomID]
	targets := make([]*Client, 0, len(clients))
	for c := range clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	h.log.Debug("hub dispatch", "room_id", m.RoomID, "targets", len(targets), "message_id", m.ID)
	for _, c := range targets {
		c.enqueue(m)
	}
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.bySocket[c]; !ok {
		return
	}
	delete(h.bySocket, c)
	for roomID := range c.subscribed {
		if clients, ok := h.byRoom[roomID]; ok {
			delete(clients, c)
			if len(clients) == 0 {
				delete(h.byRoom, roomID)
			}
		}
	}
	c.close()
}

func (h *Hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.bySocket {
		c.close()
	}
	h.bySocket = map[*Client]struct{}{}
	h.byRoom = map[uuid.UUID]map[*Client]struct{}{}
}
