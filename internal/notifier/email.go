package notifier

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/matheuslm7/notifiq/internal/delivery"
)

type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
}

func (s SMTPSender) Send(_ context.Context, job delivery.EmailJob) (string, error) {
	return s.send(job, job.From, job.FromName)
}

func (s SMTPSender) send(job delivery.EmailJob, from, fromName string) (string, error) {
	if s.Host == "" || s.Port < 1 || job.To == "" || from == "" {
		return "", fmt.Errorf("smtp sender and recipient configuration are incomplete")
	}
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)
	message := []byte("Message-ID: " + job.MessageID + "\r\n" + "From: " + fromName + " <" + from + ">\r\n" + "To: " + job.ToName + " <" + job.To + ">\r\n" + "Subject: " + job.Subject + "\r\n" + "MIME-Version: 1.0\r\n" + "Content-Type: text/html; charset=UTF-8\r\n\r\n" + job.BodyHTML)
	if err := smtp.SendMail(fmt.Sprintf("%s:%d", s.Host, s.Port), auth, from, []string{job.To}, message); err != nil {
		return "", err
	}
	return job.MessageID, nil
}
