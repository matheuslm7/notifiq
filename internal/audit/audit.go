package audit

import "context"

type Entry struct {
	OrganizationID string
	UserID         string
	Action         string
	Resource       string
	ResourceID     string
	Metadata       string
}

type Recorder interface {
	Record(context.Context, Entry) error
}
