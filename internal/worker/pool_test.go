package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/matheuslm7/notifiq/internal/metrics"
)

type senderStub struct {
	err       error
	sent      int
	messageID string
}

func (s *senderStub) Send(_ context.Context, job delivery.EmailJob) (string, error) {
	s.sent++
	if job.MessageID == "" {
		return "", s.err
	}
	s.messageID = job.MessageID
	return job.MessageID, s.err
}

type repositoryStub struct {
	claimed  bool
	attempts int
	sent     bool
	failed   string
	retried  bool
}

func (r *repositoryStub) ClaimRecipient(context.Context, string, string) (bool, int, error) {
	return r.claimed, r.attempts, nil
}
func (r *repositoryStub) MarkRecipientSent(context.Context, string, string, string) error {
	r.sent = true
	return nil
}
func (r *repositoryStub) MarkRecipientFailed(_ context.Context, _, _, message string) error {
	r.failed = message
	return nil
}
func (r *repositoryStub) MarkRecipientRetry(_ context.Context, _, _, message string, _ time.Time) error {
	r.failed = message
	r.retried = true
	return nil
}

func TestProcessorMarksSuccessfulDelivery(t *testing.T) {
	sender := &senderStub{}
	repository := &repositoryStub{claimed: true, attempts: 1}
	processor := &Processor{Sender: sender, Repository: repository}
	if err := processor.Process(context.Background(), delivery.EmailJob{CampaignID: "campaign-1", ContactID: "contact-1"}); err != nil {
		t.Fatal(err)
	}
	if sender.sent != 1 || !repository.sent {
		t.Fatalf("expected successful delivery: sent=%d marked=%v", sender.sent, repository.sent)
	}
	if sender.messageID == "" {
		t.Fatal("expected generated message id")
	}
}

func TestProcessorRecordsDeliveryMetrics(t *testing.T) {
	registry := &metrics.Metrics{}
	processor := &Processor{Sender: &senderStub{}, Repository: &repositoryStub{claimed: true, attempts: 1}, Metrics: registry}
	if err := processor.Process(context.Background(), delivery.EmailJob{CampaignID: "campaign-1", ContactID: "contact-1"}); err != nil {
		t.Fatal(err)
	}
	if got := registry.Snapshot(); got.Processed != 1 || got.Sent != 1 {
		t.Fatalf("unexpected metrics: %+v", got)
	}
}

func TestProcessorMarksFailedDelivery(t *testing.T) {
	sender := &senderStub{err: errors.New("smtp unavailable")}
	repository := &repositoryStub{claimed: true, attempts: 1}
	processor := &Processor{Sender: sender, Repository: repository}
	if err := processor.Process(context.Background(), delivery.EmailJob{CampaignID: "campaign-1", ContactID: "contact-1"}); err == nil {
		t.Fatal("expected send error")
	}
	if repository.failed != "smtp unavailable" || repository.sent || !repository.retried {
		t.Fatalf("expected retry delivery state: failed=%q sent=%v retried=%v", repository.failed, repository.sent, repository.retried)
	}
}

func TestProcessorMarksFailureAfterMaxAttempts(t *testing.T) {
	sender := &senderStub{err: errors.New("smtp unavailable")}
	repository := &repositoryStub{claimed: true, attempts: 3}
	processor := &Processor{Sender: sender, Repository: repository, MaxAttempts: 3}
	if err := processor.Process(context.Background(), delivery.EmailJob{CampaignID: "campaign-1", ContactID: "contact-1"}); err == nil {
		t.Fatal("expected send error")
	}
	if repository.failed != "smtp unavailable" || repository.retried {
		t.Fatalf("expected terminal failure: failed=%q retried=%v", repository.failed, repository.retried)
	}
}

func TestProcessorSkipsAlreadyClaimedRecipient(t *testing.T) {
	sender := &senderStub{}
	repository := &repositoryStub{claimed: false}
	processor := &Processor{Sender: sender, Repository: repository}
	if err := processor.Process(context.Background(), delivery.EmailJob{CampaignID: "campaign-1", ContactID: "contact-1"}); err != nil {
		t.Fatal(err)
	}
	if sender.sent != 0 {
		t.Fatal("already claimed recipient must not be sent again")
	}
}
