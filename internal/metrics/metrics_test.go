package metrics

import "testing"

func TestSnapshotCountsDeliveryStates(t *testing.T) {
	metrics := &Metrics{}
	metrics.IncProcessed()
	metrics.IncSent()
	metrics.IncRetried()
	metrics.IncFailed()
	if got := metrics.Snapshot(); got != (Snapshot{Processed: 1, Sent: 1, Retried: 1, Failed: 1}) {
		t.Fatalf("unexpected snapshot: %+v", got)
	}
}
