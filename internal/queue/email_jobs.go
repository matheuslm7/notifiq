package queue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/redis/go-redis/v9"
)

var ErrDeadLetterNotFound = fmt.Errorf("dead letter not found")

type EmailJobQueue struct {
	Client        *redis.Client
	Key           string
	DeadLetterKey string
}

type EmailJobConsumer interface {
	ConsumeEmailJob(ctx context.Context, timeout time.Duration) (delivery.EmailJob, error)
	PublishDeadLetter(ctx context.Context, job delivery.EmailJob, reason string) error
}

type DeadLetter struct {
	ID       string            `json:"id"`
	Job      delivery.EmailJob `json:"job"`
	Reason   string            `json:"reason"`
	FailedAt time.Time         `json:"failed_at"`
}

func (q *EmailJobQueue) PublishEmailJob(ctx context.Context, job delivery.EmailJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal email job: %w", err)
	}
	if err := q.Client.RPush(ctx, q.Key, data).Err(); err != nil {
		return fmt.Errorf("publish email job: %w", err)
	}
	return nil
}

func (q *EmailJobQueue) ConsumeEmailJob(ctx context.Context, timeout time.Duration) (delivery.EmailJob, error) {
	result, err := q.Client.BLPop(ctx, timeout, q.Key).Result()
	if err != nil {
		return delivery.EmailJob{}, err
	}
	var job delivery.EmailJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		return delivery.EmailJob{}, fmt.Errorf("unmarshal email job: %w", err)
	}
	return job, nil
}

func (q *EmailJobQueue) PublishDeadLetter(ctx context.Context, job delivery.EmailJob, reason string) error {
	data, err := json.Marshal(DeadLetter{ID: newDeadLetterID(), Job: job, Reason: reason, FailedAt: time.Now().UTC()})
	if err != nil {
		return fmt.Errorf("marshal dead letter: %w", err)
	}
	if err := q.Client.RPush(ctx, q.DeadLetterKey, data).Err(); err != nil {
		return fmt.Errorf("publish dead letter: %w", err)
	}
	return nil
}

func (q *EmailJobQueue) ListDeadLetters(ctx context.Context, organizationID string, limit int) ([]DeadLetter, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	items, err := q.Client.LRange(ctx, q.DeadLetterKey, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("list dead letters: %w", err)
	}
	letters := make([]DeadLetter, 0, len(items))
	for _, item := range items {
		var letter DeadLetter
		if err := json.Unmarshal([]byte(item), &letter); err != nil {
			return nil, fmt.Errorf("decode dead letter: %w", err)
		}
		if letter.Job.OrganizationID == organizationID {
			letters = append(letters, letter)
		}
	}
	return letters, nil
}

func (q *EmailJobQueue) RequeueDeadLetter(ctx context.Context, organizationID, id string) error {
	items, err := q.Client.LRange(ctx, q.DeadLetterKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("read dead letter: %w", err)
	}
	for _, item := range items {
		var letter DeadLetter
		if err := json.Unmarshal([]byte(item), &letter); err != nil {
			return fmt.Errorf("decode dead letter: %w", err)
		}
		if letter.ID != id || letter.Job.OrganizationID != organizationID {
			continue
		}
		var jobData []byte
		if jobData, err = json.Marshal(letter.Job); err != nil {
			return fmt.Errorf("marshal requeued job: %w", err)
		}
		removed, err := redis.NewScript(`
			local removed = redis.call('LREM', KEYS[1], 1, ARGV[1])
			if removed == 1 then
				redis.call('RPUSH', KEYS[2], ARGV[2])
			end
			return removed
		`).Run(ctx, q.Client, []string{q.DeadLetterKey, q.Key}, item, string(jobData)).Int()
		if err != nil {
			return fmt.Errorf("requeue dead letter: %w", err)
		}
		if removed == 0 {
			return ErrDeadLetterNotFound
		}
		return nil
	}
	return ErrDeadLetterNotFound
}

func newDeadLetterID() string {
	var value [12]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("dlq-%d", time.Now().UnixNano())
	}
	return "dlq-" + hex.EncodeToString(value[:])
}

var _ EmailJobConsumer = (*EmailJobQueue)(nil)
