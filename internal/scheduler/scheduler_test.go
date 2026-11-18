package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/delivery"
)

type queueStub struct{ jobs []delivery.EmailJob }

func (q *queueStub) PublishEmailJob(_ context.Context, job delivery.EmailJob) error {
	q.jobs = append(q.jobs, job)
	return nil
}

type repositoryStub struct {
	jobs   []delivery.EmailJob
	marked []string
	queued int
}

func (r *repositoryStub) MarkEmailJobQueued(context.Context, string, string) error {
	r.queued++
	return nil
}

func (r *repositoryStub) ListDueEmailJobs(context.Context, time.Time, int) ([]delivery.EmailJob, error) {
	return r.jobs, nil
}
func (r *repositoryStub) MarkCampaignSending(_ context.Context, id string) error {
	r.marked = append(r.marked, id)
	return nil
}

func TestRunOncePublishesJobsAndMarksCampaign(t *testing.T) {
	queue := &queueStub{}
	repository := &repositoryStub{jobs: []delivery.EmailJob{{CampaignID: "campaign-1", ContactID: "contact-1"}, {CampaignID: "campaign-1", ContactID: "contact-2"}}}
	scheduler, _ := New(queue, repository)
	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(queue.jobs) != 2 || repository.queued != 2 || len(repository.marked) != 1 || repository.marked[0] != "campaign-1" {
		t.Fatalf("unexpected scheduling: jobs=%d queued=%d marked=%v", len(queue.jobs), repository.queued, repository.marked)
	}
}
