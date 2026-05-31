package domain

import "context"

// EventBus is the port through which the service layer publishes realtime
// events. Concrete implementations (e.g. Redis pub/sub, in-memory) live in the
// repository layer. The transport layer (WebSocket hub) subscribes to receive
// fan-out delivery to connected clients.
type EventBus interface {
	PublishMessage(ctx context.Context, m Message) error
	SubscribeMessages(ctx context.Context) (<-chan Message, error)
	Close() error
}
