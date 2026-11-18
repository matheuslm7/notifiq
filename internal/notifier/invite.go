package notifier

import (
	"context"
	"fmt"
	"net/url"

	"github.com/matheuslm7/notifiq/internal/delivery"
	"github.com/matheuslm7/notifiq/internal/emailsettings"
	"github.com/matheuslm7/notifiq/internal/members"
)

type InviteSender struct {
	Resolver interface {
		ResolveSMTP(context.Context, string) (emailsettings.SMTPConfig, error)
	}
	BaseURL string
}

func (s InviteSender) SendInvite(ctx context.Context, invite members.Invite) error {
	config, err := s.Resolver.ResolveSMTP(ctx, invite.OrganizationID)
	if err != nil {
		return err
	}
	link := s.BaseURL + "?token=" + url.QueryEscape(invite.Token)
	job := delivery.EmailJob{To: invite.Email, From: config.FromEmail, FromName: config.FromName, Subject: "Você foi convidado para o NotifiQ", BodyHTML: fmt.Sprintf("<p>Você foi convidado para participar de uma organização no NotifiQ.</p><p><a href=\"%s\">Aceitar convite</a></p><p>Este convite expira em 72 horas.</p>", link), MessageID: "<invite-" + invite.Token + "@notifiq.local>"}
	_, err = SMTPSender{Host: config.Host, Port: config.Port, Username: config.Username, Password: config.Password}.Send(ctx, job)
	return err
}
