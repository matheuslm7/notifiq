//internal/worker/pool.go

package worker

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/matheuslm7/notifiq/internal/metrics"
	"github.com/redis/go-redis/v9"
)

type Processor struct {
	Sender      delivery.Sender
	Repository  delivery.Repository
	MaxAttempts int
	Metrics     *metrics.Metrics
}

type TerminalError struct{ Err error }

func (e TerminalError) Error() string { return e.Err.Error() }
func (e TerminalError) Unwrap() error { return e.Err }

func (p *Processor) Process(ctx context.Context, job delivery.EmailJob) error {
	if p.Sender == nil || p.Repository == nil {
		return fmt.Errorf("sender and repository are required")
	}
	claimed, attempts, err := p.Repository.ClaimRecipient(ctx, job.CampaignID, job.ContactID)
	if err != nil {
		return fmt.Errorf("claim recipient: %w", err)
	}
	if !claimed {
		return nil
	}
	if job.MessageID == "" {
		job.MessageID = newMessageID()
	}
	if p.Metrics != nil {
		p.Metrics.IncProcessed()
	}
	messageID, err := p.Sender.Send(ctx, job)
	if err != nil {
		maxAttempts := p.MaxAttempts
		if maxAttempts == 0 {
			maxAttempts = 3
		}
		if attempts < maxAttempts {
			delay := time.Duration(1<<(attempts-1)) * time.Second
			if retryErr := p.Repository.MarkRecipientRetry(ctx, job.CampaignID, job.ContactID, err.Error(), time.Now().UTC().Add(delay)); retryErr != nil {
				return fmt.Errorf("send email: %v; schedule retry: %w", err, retryErr)
			}
			if p.Metrics != nil {
				p.Metrics.IncRetried()
			}
			return fmt.Errorf("send email (retry %d/%d): %w", attempts, maxAttempts, err)
		}
		if markErr := p.Repository.MarkRecipientFailed(ctx, job.CampaignID, job.ContactID, err.Error()); markErr != nil {
			return fmt.Errorf("send email: %v; mark failed: %w", err, markErr)
		}
		if p.Metrics != nil {
			p.Metrics.IncFailed()
		}
		return TerminalError{Err: fmt.Errorf("send email: %w", err)}
	}
	if messageID == "" {
		messageID = job.MessageID
	}
	if err := p.Repository.MarkRecipientSent(ctx, job.CampaignID, job.ContactID, messageID); err != nil {
		return fmt.Errorf("mark recipient sent: %w", err)
	}
	if p.Metrics != nil {
		p.Metrics.IncSent()
	}
	return nil
}

func newMessageID() string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "notifiq-generated-message-id"
	}
	return "<" + hex.EncodeToString(value[:]) + "@notifiq.local>"
}

func (p *Processor) Run(ctx context.Context, jobs <-chan delivery.EmailJob) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			_ = p.Process(ctx, job)
		}
	}
}

func (p *Processor) RunQueue(ctx context.Context, queue interface {
	ConsumeEmailJob(context.Context, time.Duration) (delivery.EmailJob, error)
	PublishDeadLetter(context.Context, delivery.EmailJob, string) error
}) {
	for {
		job, err := queue.ConsumeEmailJob(ctx, 5*time.Second)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			continue
		}
		if err := p.Process(ctx, job); err != nil {
			var terminal TerminalError
			if errors.As(err, &terminal) {
				_ = queue.PublishDeadLetter(ctx, job, terminal.Error())
			}
		}
	}
}
