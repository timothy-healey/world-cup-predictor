package mailer

import (
	"fmt"
	"net/smtp"
)

type Mailer struct {
	host, port string
	user, pass string
	from, to   string
}

func New(host, port, user, pass, to string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: user, to: to}
}

func (m *Mailer) Send(subject, body string) error {
	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	auth := smtp.PlainAuth("", m.user, m.pass, m.host)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s",
		m.from, m.to, subject, body))
	return smtp.SendMail(addr, auth, m.from, []string{m.to}, msg)
}
