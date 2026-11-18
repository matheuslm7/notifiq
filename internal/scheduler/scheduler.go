package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/matheuslm7/notifiq/internal/delivery"
)

type Queue interface {
	PublishEmailJob(context.Context, delivery.EmailJob) error
}
type Repository interface {
	ListDueEmailJobs(context.Context, time.Time, int) ([]delivery.EmailJob, error)
	MarkEmailJobQueued(context.Context, string, string) error
	MarkCampaignSending(context.Context, string) error
}

type Scheduler struct {
	queue      Queue
	repository Repository
	interval   time.Duration
	batchSize  int
}

func New(queue Queue, repository Repository) (*Scheduler, error) {
	if queue == nil || repository == nil {
		return nil, fmt.Errorf("queue and repository are required")
	}
	return &Scheduler{queue: queue, repository: repository, interval: 10 * time.Second, batchSize: 100}, nil
}

func (s *Scheduler) RunOnce(ctx context.Context) error {
	jobs, err := s.repository.ListDueEmailJobs(ctx, time.Now().UTC(), s.batchSize)
	if err != nil {
		return fmt.Errorf("list due email jobs: %w", err)
	}
	campaigns := make(map[string]struct{})
	for _, job := range jobs {
		if err := s.queue.PublishEmailJob(ctx, job); err != nil {
			return fmt.Errorf("publish campaign job: %w", err)
		}
		if err := s.repository.MarkEmailJobQueued(ctx, job.CampaignID, job.ContactID); err != nil {
			return fmt.Errorf("mark campaign job queued: %w", err)
		}
		campaigns[job.CampaignID] = struct{}{}
	}
	for campaignID := range campaigns {
		if err := s.repository.MarkCampaignSending(ctx, campaignID); err != nil {
			return fmt.Errorf("mark campaign sending: %w", err)
		}
	}
	return nil
}

func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.RunOnce(ctx)
		}
	}
}
