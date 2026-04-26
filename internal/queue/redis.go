//internal/queue/redis.go

package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/matheuslm7/notifiq/internal/event"
	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	Client *redis.Client
	ListKey string
}

type Publisher interface {
	Publish(ctx context.Context, event event.Event) error
}

func (q *RedisQueue) Publish(ctx context.Context, e event.Event) error {
	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return q.Client.LPush(ctx, q.ListKey, data).Err()
}

func (q *RedisQueue) StartConsumer(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            log.Println("consumer: shutting down")
            return
        default:
        }

        result, err := q.Client.BRPop(ctx, 5*time.Second, q.ListKey).Result()
        if err != nil {
            if errors.Is(err, redis.Nil) {
                continue // timeout expirou, nenhum item — normal
            }
            if errors.Is(err, context.Canceled) {
                return
            }
            log.Printf("consumer: brpop error: %v", err)
            continue
        }

        // result[0] = nome da lista, result[1] = valor
        var e event.Event
        if err := json.Unmarshal([]byte(result[1]), &e); err != nil {
            log.Printf("consumer: unmarshal error: %v", err)
            continue
        }

        log.Printf("consumer: received event id=%s type=%s", e.ID, e.EventType)
    }
}