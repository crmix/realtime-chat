package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/realtime-chat/internal/domain"
	"github.com/realtime-chat/internal/transport/http/dto"
)

const (
	writeTimeout    = 10 * time.Second
	pongTimeout     = 60 * time.Second
	pingInterval    = (pongTimeout * 9) / 10
	maxMessageBytes = 8 * 1024
	sendBuffer      = 64
)

// Client wraps a single WebSocket connection. It reads inbound commands from
// the browser and writes outbound events from the hub. Two goroutines per
// client: a reader and a writer.
type Client struct {
	conn   *websocket.Conn
	hub    *Hub
	rooms  domain.RoomService
	msgs   domain.MessageService
	log    *slog.Logger
	userID uuid.UUID

	send       chan domain.Message
	subscribed map[uuid.UUID]struct{}

	closeOnce sync.Once
	closed    chan struct{}
}

type inboundEnvelope struct {
	Type    string    `json:"type"`
	RoomID  uuid.UUID `json:"room_id,omitempty"`
	Content string    `json:"content,omitempty"`
}

type outboundEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func newClient(conn *websocket.Conn, hub *Hub, rooms domain.RoomService, msgs domain.MessageService, log *slog.Logger, userID uuid.UUID) *Client {
	return &Client{
		conn:       conn,
		hub:        hub,
		rooms:      rooms,
		msgs:       msgs,
		log:        log,
		userID:     userID,
		send:       make(chan domain.Message, sendBuffer),
		subscribed: map[uuid.UUID]struct{}{},
		closed:     make(chan struct{}),
	}
}

func (c *Client) enqueue(m domain.Message) {
	select {
	case c.send <- m:
	default:
		// Slow consumer — drop and close the connection to avoid backing up
		// the hub. The client can reconnect and replay history via REST.
		c.log.Warn("dropping slow client", "user_id", c.userID)
		c.hub.unregister <- c
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.conn.Close()
	})
}

func (c *Client) readPump(ctx context.Context) {
	defer func() { c.hub.unregister <- c }()

	c.conn.SetReadLimit(maxMessageBytes)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongTimeout))
		return nil
	})

	for {
		var env inboundEnvelope
		if err := c.conn.ReadJSON(&env); err != nil {
			if !errors.Is(err, websocket.ErrCloseSent) {
				var ce *websocket.CloseError
				if !errors.As(err, &ce) {
					c.log.Debug("ws read error", "err", err, "user_id", c.userID)
				}
			}
			return
		}
		c.handleInbound(ctx, env)
	}
}

func (c *Client) handleInbound(ctx context.Context, env inboundEnvelope) {
	switch env.Type {
	case "subscribe":
		if err := c.rooms.EnsureMember(ctx, env.RoomID, c.userID); err != nil {
			c.writeError("subscribe", err)
			return
		}
		c.hub.subscribe <- roomMembership{client: c, roomID: env.RoomID}
		c.writeAck("subscribed", env.RoomID)

	case "unsubscribe":
		c.hub.unsubscribe <- roomMembership{client: c, roomID: env.RoomID}
		c.writeAck("unsubscribed", env.RoomID)

	case "message":
		if _, err := c.msgs.Send(ctx, env.RoomID, c.userID, env.Content); err != nil {
			c.writeError("message", err)
			return
		}

	case "ping":
		c.writeAck("pong", uuid.Nil)

	default:
		c.writeError("unknown", errors.New("unknown message type"))
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case <-c.closed:
			return

		case m, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			payload, err := json.Marshal(dto.MessageFrom(&m))
			if err != nil {
				c.log.Error("marshal outbound", "err", err)
				continue
			}
			env := outboundEnvelope{Type: "message", Payload: payload}
			if err := c.conn.WriteJSON(env); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) writeAck(kind string, roomID uuid.UUID) {
	payload, _ := json.Marshal(map[string]any{"room_id": roomID})
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_ = c.conn.WriteJSON(outboundEnvelope{Type: kind, Payload: payload})
}

func (c *Client) writeError(kind string, err error) {
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_ = c.conn.WriteJSON(outboundEnvelope{Type: kind + ".error", Error: err.Error()})
}
