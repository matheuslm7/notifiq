//cmd/main.go

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	"github.com/matheuslm7/notifiq/internal/api"
	"github.com/matheuslm7/notifiq/internal/queue"
	"github.com/redis/go-redis/v9"
)

func main() {

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:4000"
	}

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})

	q := &queue.RedisQueue{Client: rdb, ListKey: "notifiq:events"}

	handler := &api.EventHandler{Publisher: q}
	http.HandleFunc("/events", handler.CreateEvent)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go q.StartConsumer(ctx)

	srv := &http.Server{Addr: ":8080"}
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()
	srv.ListenAndServe()
}
