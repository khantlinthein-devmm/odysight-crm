package mailer

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"

	"github.com/odysight/crm/internal/settings"
)

// fakeSMTP accepts one message and records the MAIL FROM line and the data.
func fakeSMTP(t *testing.T) (port int, mailFrom, data *string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var from, body string
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		r := bufio.NewReader(c)
		w := func(s string) { _, _ = c.Write([]byte(s + "\r\n")) }
		w("220 fake")
		inData := false
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			if inData {
				if line == ".\r\n" {
					inData = false
					w("250 ok")
					continue
				}
				body += line
				continue
			}
			switch cmd := strings.ToUpper(strings.TrimSpace(line)); {
			case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
				w("250 fake")
			case strings.HasPrefix(cmd, "MAIL FROM"):
				from = strings.TrimSpace(line)
				w("250 ok")
			case strings.HasPrefix(cmd, "RCPT"):
				w("250 ok")
			case cmd == "DATA":
				inData = true
				w("354 go")
			case cmd == "QUIT":
				w("221 bye")
				return
			default:
				w("250 ok")
			}
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port, &from, &body
}

func TestSendUsesBareEnvelopeSender(t *testing.T) {
	port, from, data := fakeSMTP(t)
	m := New(settings.SMTPSettings{Enabled: true, Host: "127.0.0.1", Port: port,
		FromEmail: "billing@smileclean.test", FromName: "Smile Clean", Encryption: "none"})
	if err := m.Send(context.Background(), "customer@example.com", "Quotation Q-1", "<p>hi</p>",
		&Attachment{FileName: "Q-1.pdf", Data: []byte("%PDF-1.4")}); err != nil {
		t.Fatal(err)
	}
	if *from != "MAIL FROM:<billing@smileclean.test>" {
		t.Fatalf("envelope sender = %q", *from)
	}
	if !strings.Contains(*data, "Smile Clean") || !strings.Contains(*data, "Q-1.pdf") {
		t.Fatalf("header/attachment missing:\n%s", *data)
	}
}

func TestAttachmentLinesStayShort(t *testing.T) {
	big := make([]byte, 50_000)
	for i := range big {
		big[i] = byte(i)
	}
	msg, err := buildMessage("a@b.c", "c@d.e", "PDF", "<p>x</p>", &Attachment{FileName: "Q-1.pdf", Data: big})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(msg), "\r\n") {
		if len(line) > 998 {
			t.Fatalf("line of %d bytes; SMTP allows 998", len(line))
		}
	}
	if !strings.Contains(string(msg), "Content-Type: application/pdf") {
		t.Fatal("PDF attachment should be labelled application/pdf")
	}
}
