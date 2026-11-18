//internal/notifier/notifier.go

package notifier

import (
	"context"

	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/matheuslm7/notifiq/internal/emailsettings"
)

type EmailSender interface {
	Send(context.Context, delivery.EmailJob) (string, error)
}

type OrganizationSender struct {
	Resolver interface {
		ResolveSMTP(context.Context, string) (emailsettings.SMTPConfig, error)
	}
}

func (s OrganizationSender) Send(ctx context.Context, job delivery.EmailJob) (string, error) {
	config, err := s.Resolver.ResolveSMTP(ctx, job.OrganizationID)
	if err != nil {
		return "", err
	}
	return SMTPSender{Host: config.Host, Port: config.Port, Username: config.Username, Password: config.Password}.send(job, config.FromEmail, config.FromName)
}
