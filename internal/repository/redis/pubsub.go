package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/realtime-chat/internal/domain"
)

// EventBus uses a single Redis pub/sub channel to fan out chat messages across
// every server instance. Subscribers filter by RoomID downstream.
type EventBus struct {
	client  *redis.Client
	channel string
}

func NewEventBus(client *redis.Client, channel string) *EventBus {
	if channel == "" {
		channel = "chat.messages"
	}
	return &EventBus{client: client, channel: channel}
}

type messageEnvelope struct {
	ID        uuid.UUID `json:"id"`
	RoomID    uuid.UUID `json:"room_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (b *EventBus) PublishMessage(ctx context.Context, m domain.Message) error {
	env := messageEnvelope{
		ID:        m.ID,
		RoomID:    m.RoomID,
		UserID:    m.UserID,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
	}
	payload, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return b.client.Publish(ctx, b.channel, payload).Err()
}

// SubscribeMessages opens a subscription and returns a channel of incoming
// messages. The returned channel is closed when ctx is cancelled or the
// underlying connection errors.
func (b *EventBus) SubscribeMessages(ctx context.Context) (<-chan domain.Message, error) {
	sub := b.client.Subscribe(ctx, b.channel)
	if _, err := sub.Receive(ctx); err != nil {
		_ = sub.Close()
		return nil, fmt.Errorf("subscribe %s: %w", b.channel, err)
	}

	out := make(chan domain.Message, 256)
	go func() {
		defer close(out)
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var env messageEnvelope
				if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
					continue
				}
				out <- domain.Message{
					ID:        env.ID,
					RoomID:    env.RoomID,
					UserID:    env.UserID,
					Content:   env.Content,
					CreatedAt: env.CreatedAt,
				}
			}
		}
	}()
	return out, nil
}

func (b *EventBus) Close() error {
	return nil
}
