// Package mailer sends transactional emails over SMTP with optional
// STARTTLS or implicit TLS (SSL) encryption and binary attachments.
package mailer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/odysight/crm/internal/settings"
)

// Mailer delivers messages through the configured SMTP server.
type Mailer struct {
	cfg settings.SMTPSettings
}

func New(cfg settings.SMTPSettings) *Mailer {
	return &Mailer{cfg: cfg}
}

// Enabled reports whether a usable mail server is configured.
func (m *Mailer) Enabled() bool {
	return m.cfg.Enabled && strings.TrimSpace(m.cfg.Host) != "" &&
		strings.TrimSpace(m.cfg.FromEmail) != ""
}

// Attachment is an inline binary file carried in the message.
type Attachment struct {
	FileName string
	Data     []byte
}

// Send delivers an HTML email (and optional attachment) to a single recipient.
func (m *Mailer) Send(ctx context.Context, to, subject, bodyHTML string, att *Attachment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !m.Enabled() {
		return fmt.Errorf("smtp is not enabled")
	}

	from := m.cfg.FromEmail
	if name := strings.TrimSpace(m.cfg.FromName); name != "" {
		from = fmt.Sprintf("%s <%s>", mimeQ(name), m.cfg.FromEmail)
	}

	msg, err := buildMessage(from, to, subject, bodyHTML, att)
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(m.cfg.Host, fmt.Sprintf("%d", m.cfg.Port))
	var auth smtp.Auth
	if strings.TrimSpace(m.cfg.Username) != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	if m.cfg.Encryption == "ssl" {
		client, err := tlsDial(addr)
		if err != nil {
			return fmt.Errorf("tls dial %s: %w", addr, err)
		}
		defer client.Close()
		if err := deliver(ctx, client, auth, from, []string{to}, msg); err != nil {
			return err
		}
		return nil
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial %s: %w", addr, err)
	}
	defer client.Close()
	if m.cfg.Encryption == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host}); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}
	return deliver(ctx, client, auth, from, []string{to}, msg)
}

type sendCloser interface {
	Hello(string) error
	Auth(smtp.Auth) error
	Mail(string) error
	Rcpt(string) error
	Data() (io.WriteCloser, error)
	Quit() error
	Close() error
}

func tlsDial(addr string) (*smtp.Client, error) {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: hostOf(addr)})
	if err != nil {
		return nil, err
	}
	return smtp.NewClient(conn, hostOf(addr))
}

func hostOf(addr string) string {
	h, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return h
}

func deliver(ctx context.Context, c sendCloser, auth smtp.Auth, from string, to []string, msg []byte) error {
	if err := c.Hello("odysight-crm"); err != nil {
		return fmt.Errorf("smtp hello: %w", err)
	}
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp mail: %w", err)
	}
	for _, addr := range to {
		if err := c.Rcpt(addr); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", addr, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	_ = c.Quit()
	return nil
}

// buildMessage renders a MIME multipart/mixed message: quoted-printable
// HTML body plus an optional base64 application/octet-stream attachment.
func buildMessage(from, to, subject, bodyHTML string, att *Attachment) ([]byte, error) {
	var buf bytes.Buffer
	hdrs := textproto.MIMEHeader{}
	hdrs.Set("From", from)
	hdrs.Set("To", to)
	hdrs.Set("Subject", mimeQ(subject))
	hdrs.Set("Date", time.Now().Format(time.RFC1123Z))
	hdrs.Set("MIME-Version", "1.0")

	if att == nil {
		hdrs.Set("Content-Type", "text/html; charset=\"utf-8\"")
		hdrs.Set("Content-Transfer-Encoding", "quoted-printable")
		for k, v := range hdrs {
			fmt.Fprintf(&buf, "%s: %s\r\n", k, strings.Join(v, ", "))
		}
		buf.WriteString("\r\n")
		q := quotedprintable.NewWriter(&buf)
		if _, err := q.Write([]byte(bodyHTML)); err != nil {
			return nil, err
		}
		return buf.Bytes(), q.Close()
	}

	boundary := fmt.Sprintf("odysight-%d", time.Now().UnixNano())
	hdrs.Set("Content-Type", "multipart/mixed; boundary=\""+boundary+"\"")
	hdrs.Set("Content-Disposition", "")

	for k, v := range hdrs {
		if k == "Content-Disposition" {
			continue
		}
		fmt.Fprintf(&buf, "%s: %s\r\n", k, strings.Join(v, ", "))
	}
	buf.WriteString("\r\n")

	mw := multipart.NewWriter(&buf)
	if err := mw.SetBoundary(boundary); err != nil {
		return nil, err
	}

	// HTML part.
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Type", "text/html; charset=\"utf-8\"")
	hdr.Set("Content-Transfer-Encoding", "quoted-printable")
	part, err := mw.CreatePart(hdr)
	if err != nil {
		return nil, err
	}
	q := quotedprintable.NewWriter(part)
	if _, err := q.Write([]byte(bodyHTML)); err != nil {
		return nil, err
	}
	if err := q.Close(); err != nil {
		return nil, err
	}

	// Attachment part.
	ahdr := textproto.MIMEHeader{}
	ahdr.Set("Content-Type", "application/octet-stream")
	ahdr.Set("Content-Transfer-Encoding", "base64")
	ahdr.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, att.FileName))
	apart, err := mw.CreatePart(ahdr)
	if err != nil {
		return nil, err
	}
	enc := base64.NewEncoder(base64.StdEncoding, apart)
	if _, err := enc.Write(att.Data); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}

	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mimeQ(s string) string {
	// RFC 2047-encode non-ASCII headers via utf-8 base64.
	if isASCII(s) {
		return s
	}
	return "=?utf-8?B?" + base64.StdEncoding.EncodeToString([]byte(s)) + "?="
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}