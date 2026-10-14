// Package mail sends email over SMTP using only the standard library.
package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"passwordmanage/server/internal/app/notify"
	"passwordmanage/server/internal/platform/config"
)

const sendTimeout = 30 * time.Second

type SMTP struct {
	cfg config.SMTP
	now func() time.Time
}

func NewSMTP(cfg config.SMTP) *SMTP {
	return &SMTP{cfg: cfg, now: time.Now}
}

func (s *SMTP) Send(ctx context.Context, email notify.Email) error {
	from, err := mail.ParseAddress(s.cfg.From)
	if err != nil {
		return fmt.Errorf("invalid PM_SMTP_FROM: %w", err)
	}
	msg, err := buildMessage(from, email, s.now())
	if err != nil {
		return err
	}
	to, err := mail.ParseAddress(email.To)
	if err != nil {
		return fmt.Errorf("invalid recipient: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()

	client, err := s.dial(ctx)
	if err != nil {
		return err
	}
	// Quit closes the connection on success; this only cleans up after an earlier failure.
	defer func() { _ = client.Close() }()

	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to.Address); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp end of data: %w", err)
	}
	return client.Quit()
}

func (s *SMTP) dial(ctx context.Context) (*smtp.Client, error) {
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	tlsConfig := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}

	var conn net.Conn
	var err error
	if s.cfg.TLS == config.SMTPTLSImplicit {
		dialer := &tls.Dialer{Config: tlsConfig}
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	} else {
		var dialer net.Dialer
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, fmt.Errorf("smtp connect: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline) // best effort; the context timeout still bounds the dial
	}

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close() // already failing; the handshake error is the one worth reporting
		return nil, fmt.Errorf("smtp handshake: %w", err)
	}
	if s.cfg.TLS == config.SMTPTLSStartTLS {
		if err := client.StartTLS(tlsConfig); err != nil {
			_ = client.Close() // already failing; the STARTTLS error is the one worth reporting
			return nil, fmt.Errorf("smtp STARTTLS: %w", err)
		}
	}
	return client, nil
}

var errHeaderInjection = errors.New("email header contains a line break")

// buildMessage renders a plain-text UTF-8 message. Line breaks in headers are
// rejected because they would let input inject extra headers or recipients.
func buildMessage(from *mail.Address, email notify.Email, now time.Time) ([]byte, error) {
	if strings.ContainsAny(email.To, "\r\n") || strings.ContainsAny(email.Subject, "\r\n") {
		return nil, errHeaderInjection
	}
	to, err := mail.ParseAddress(email.To)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient: %w", err)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "From: %s\r\n", from.String())
	fmt.Fprintf(&buf, "To: %s\r\n", to.String())
	fmt.Fprintf(&buf, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", email.Subject))
	fmt.Fprintf(&buf, "Date: %s\r\n", now.Format(time.RFC1123Z))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

	qp := quotedprintable.NewWriter(&buf)
	if _, err := qp.Write([]byte(email.Text)); err != nil {
		return nil, err
	}
	if err := qp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
