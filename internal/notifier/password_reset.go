package notifier

import (
	"context"
	"fmt"
	"net/url"

	"github.com/matheuslm7/notifiq/internal/auth"
	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/matheuslm7/notifiq/internal/emailsettings"
)

type PasswordResetSender struct {
	Resolver interface {
		ResolveSMTP(context.Context, string) (emailsettings.SMTPConfig, error)
	}
	BaseURL string
}

func (s PasswordResetSender) SendPasswordReset(ctx context.Context, user auth.User, token string) error {
	config, err := s.Resolver.ResolveSMTP(ctx, user.OrganizationID)
	if err != nil {
		return err
	}
	link := s.BaseURL + "?token=" + url.QueryEscape(token)
	job := delivery.EmailJob{To: user.Email, ToName: user.Name, From: config.FromEmail, FromName: config.FromName, Subject: "Redefinição de senha do NotifiQ", BodyHTML: fmt.Sprintf("<p>Recebemos uma solicitação para redefinir sua senha.</p><p><a href=\"%s\">Redefinir senha</a></p><p>Este link expira em 30 minutos.</p>", link), MessageID: "<password-reset-" + token + "@notifiq.local>"}
	_, err = SMTPSender{Host: config.Host, Port: config.Port, Username: config.Username, Password: config.Password}.Send(ctx, job)
	return err
}
