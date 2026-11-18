package main

import (
	"context"
	"errors"
	"testing"
)

func TestWaitForDependencyRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	err := waitForDependency(context.Background(), "test", func(context.Context) error {
		attempts++
		if attempts < 2 {
			return errors.New("not ready")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected dependency to become ready: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected two attempts, got %d", attempts)
	}
}

func TestWaitForDependencyReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForDependency(ctx, "test", func(context.Context) error {
		return errors.New("not ready")
	})
	if err == nil {
		t.Fatal("expected context error")
	}
}
