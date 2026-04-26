//internal/event/event.go

package event

import "time"

type Event struct {
	ID        string
	EventType string
	Payload   map[string]any
	Timestamp time.Time
}
