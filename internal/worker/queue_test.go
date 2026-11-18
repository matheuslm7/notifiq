package worker

import (
	"context"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/delivery"
)

type queueStub struct{ jobs []delivery.EmailJob }

func (q *queueStub) PublishDeadLetter(context.Context, delivery.EmailJob, string) error { return nil }

func (q *queueStub) ConsumeEmailJob(ctx context.Context, _ time.Duration) (delivery.EmailJob, error) {
	if len(q.jobs) == 0 {
		return delivery.EmailJob{}, context.Canceled
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job, nil
}

func TestRunQueueStopsWhenQueueContextIsCancelled(t *testing.T) {
	queue := &queueStub{}
	processor := &Processor{Sender: &senderStub{}, Repository: &repositoryStub{}}
	processor.RunQueue(context.Background(), queue)
}
