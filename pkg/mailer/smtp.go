package mailer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const defaultSMTPTimeout = 5 * time.Second

// SMTPMailer отправляет письма по протоколу SMTP с таймаутами и защитой заголовков от инъекций.
type SMTPMailer struct {
	host     string
	port     string
	user     string
	password string
	from     string
	timeout  time.Duration
}

// NewSMTPMailer создаёт экземпляр SMTPMailer.
func NewSMTPMailer(host, port, user, password, from string) (*SMTPMailer, error) {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	from = strings.TrimSpace(from)

	if host == "" {
		return nil, errors.New("smtp host cannot be empty")
	}
	if port == "" {
		port = "1025"
	}
	if hasCRLF(host) || hasCRLF(port) || hasCRLF(from) || from == "" {
		return nil, errors.New("invalid smtp configuration")
	}

	return &SMTPMailer{
		host:     host,
		port:     port,
		user:     strings.TrimSpace(user),
		password: password,
		from:     from,
		timeout:  defaultSMTPTimeout,
	}, nil
}

// SendVerificationCode отправляет 6-значный код подтверждения на указанный email.
func (m *SMTPMailer) SendVerificationCode(ctx context.Context, toEmail, code string) error {
	toEmail = strings.TrimSpace(toEmail)
	subject := "Код подтверждения регистрации на Bboard"

	if hasCRLF(toEmail) || hasCRLF(code) || hasCRLF(m.from) || hasCRLF(subject) {
		return errors.New("invalid header characters in email message")
	}

	addr := net.JoinHostPort(m.host, m.port)
	dialer := &net.Dialer{Timeout: m.timeout}
	var (
		conn net.Conn
		err  error
	)
	if m.port == "465" {
		tlsDialer := &tls.Dialer{
			NetDialer: dialer,
			Config: &tls.Config{
				ServerName: m.host,
				MinVersion: tls.VersionTLS12,
			},
		}
		conn, err = tlsDialer.DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("dial smtp server %s: %w", addr, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(m.timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set smtp connection deadline: %w", err)
	}

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if m.port != "465" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName: m.host,
				MinVersion: tls.VersionTLS12,
			}); err != nil {
				return fmt.Errorf("smtp STARTTLS: %w", err)
			}
		}
	}

	if m.user != "" {
		ok, _ := client.Extension("AUTH")
		if !ok {
			return errors.New("smtp server does not support AUTH")
		}
		auth := smtp.PlainAuth("", m.user, m.password, m.host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(m.from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(toEmail); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}

	encodedSubject := mime.QEncoding.Encode("utf-8", subject)
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"UTF-8\"\r\n\r\nВаш код подтверждения регистрации на Bboard: %s\r\nКод действует 5 минут.\r\n",
		m.from,
		toEmail,
		encodedSubject,
		code,
	)

	if _, err := w.Write([]byte(msg)); err != nil {
		_ = w.Close()
		return fmt.Errorf("write smtp message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close smtp data writer: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp QUIT: %w", err)
	}
	return nil
}

func hasCRLF(s string) bool {
	return strings.ContainsAny(s, "\r\n")
}

// LogMailer — dev-заглушка, логирующая код подтверждения при не заданном SMTP_HOST.
type LogMailer struct {
	log *logrus.Logger
}

// NewLogMailer создаёт LogMailer.
func NewLogMailer(log *logrus.Logger) *LogMailer {
	return &LogMailer{log: log}
}

// SendVerificationCode логирует отправку кода подтверждения в журнал приложения.
func (m *LogMailer) SendVerificationCode(_ context.Context, toEmail, code string) error {
	m.log.WithFields(logrus.Fields{
		"to_email": toEmail,
		"code":     code,
	}).Info("dev log mailer: verification code generated")
	return nil
}
