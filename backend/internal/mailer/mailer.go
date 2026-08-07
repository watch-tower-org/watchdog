package mailer

import (
	"crypto/tls"
	"fmt"
	"time"

	"gopkg.in/mail.v2"
)

func FormatFrom(fromName, fromEmail string) string {
	if fromName != "" {
		return fmt.Sprintf("%s <%s>", fromName, fromEmail)
	}
	return fromEmail
}

func Send(host string, port int, username, password, fromName, fromEmail, to, subject, body string) error {
	d := mail.NewDialer(host, port, username, password)
	d.SSL = port == 465
	d.Timeout = 10 * time.Second

	if !d.SSL {
		d.TLSConfig = &tls.Config{ServerName: host}
	}

	m := mail.NewMessage()
	m.SetHeader("From", FormatFrom(fromName, fromEmail))
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}
