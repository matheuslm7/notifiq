package metrics

import "sync/atomic"

type Snapshot struct {
	Processed uint64 `json:"processed"`
	Sent      uint64 `json:"sent"`
	Retried   uint64 `json:"retried"`
	Failed    uint64 `json:"failed"`
}

type DeliveryStats struct {
	Total     uint64 `json:"total"`
	Pending   uint64 `json:"pending"`
	Sending   uint64 `json:"sending"`
	Sent      uint64 `json:"sent"`
	Failed    uint64 `json:"failed"`
	Cancelled uint64 `json:"cancelled"`
}
type Metrics struct {
	processed atomic.Uint64
	sent      atomic.Uint64
	retried   atomic.Uint64
	failed    atomic.Uint64
}

func (m *Metrics) IncProcessed() { m.processed.Add(1) }
func (m *Metrics) IncSent()      { m.sent.Add(1) }
func (m *Metrics) IncRetried()   { m.retried.Add(1) }
func (m *Metrics) IncFailed()    { m.failed.Add(1) }
func (m *Metrics) Snapshot() Snapshot {
	return Snapshot{Processed: m.processed.Load(), Sent: m.sent.Load(), Retried: m.retried.Load(), Failed: m.failed.Load()}
}
