package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/matheuslm7/notifiq/internal/event"
)

type publisherStub struct {
	event event.Event
	err   error
}

func (p *publisherStub) Publish(_ context.Context, e event.Event) error {
	p.event = e
	return p.err
}

func TestCreateEventPublishesEvent(t *testing.T) {
	publisher := &publisherStub{}
	handler := &EventHandler{Publisher: publisher}
	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"id":"evt-1","type":"user.created","payload":{"name":"Ana"}}`))
	res := httptest.NewRecorder()

	handler.CreateEvent(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, res.Code)
	}
	if publisher.event.ID != "evt-1" || publisher.event.EventType != "user.created" {
		t.Fatalf("unexpected published event: %+v", publisher.event)
	}
	if publisher.event.Timestamp.IsZero() {
		t.Fatal("expected event timestamp to be set")
	}
}

func TestCreateEventRejectsInvalidRequest(t *testing.T) {
	handler := &EventHandler{Publisher: &publisherStub{}}
	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"id":""}`))
	res := httptest.NewRecorder()

	handler.CreateEvent(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestCreateEventReturnsServerErrorWhenPublishingFails(t *testing.T) {
	handler := &EventHandler{Publisher: &publisherStub{err: errors.New("redis unavailable")}}
	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"id":"evt-1","type":"user.created"}`))
	res := httptest.NewRecorder()

	handler.CreateEvent(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, res.Code)
	}
}

func TestCreateEventRejectsUnsupportedMethod(t *testing.T) {
	handler := &EventHandler{Publisher: &publisherStub{}}
	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	res := httptest.NewRecorder()

	handler.CreateEvent(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d", http.StatusMethodNotAllowed, res.Code)
	}
}
