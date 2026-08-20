package handlers

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

// Notifier sends email notifications (ticket assigned, engineer onsite/
// offsite, etc.). If SMTP_HOST/SMTP_PORT/SMTP_USER/SMTP_PASS are set, it
// sends real emails over SMTP using only the Go standard library — no
// third-party email service required, any provider that exposes SMTP
// works (Gmail with an app password, Outlook, a transactional email
// provider's SMTP endpoint, etc.).
//
// If those env vars aren't set, it logs what would have been sent
// instead of failing — so the rest of the system (dispatch, attendance)
// keeps working even before email is configured.
type Notifier struct {
	host, port, user, pass, from string
	configured                   bool
}

func NewNotifierFromEnv() *Notifier {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = user
	}

	n := &Notifier{host: host, port: port, user: user, pass: pass, from: from}
	n.configured = host != "" && port != "" && user != "" && pass != ""
	if !n.configured {
		log.Println("Notifier: SMTP not configured (set SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS) — emails will be logged, not sent.")
	}
	return n
}

// Send fires off a plain-text email. Errors are logged, not returned —
// a failed notification should never block the action that triggered it
// (assigning a ticket, checking in, etc.).
func (n *Notifier) Send(to, subject, body string) {
	if to == "" {
		return
	}
	if !n.configured {
		log.Printf("Notifier (SMTP not configured, not sent) — to=%s subject=%q\n%s", to, subject, body)
		return
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", n.from, to, subject, body)
	addr := n.host + ":" + n.port
	auth := smtp.PlainAuth("", n.user, n.pass, n.host)

	if err := smtp.SendMail(addr, auth, n.from, []string{to}, []byte(msg)); err != nil {
		log.Printf("Notifier: failed to send email to %s: %v", to, err)
	}
}

// supervisorEmailFromEnv returns NOTIFY_SUPERVISOR_EMAIL, or "" if unset.
// Used for onsite/offsite attendance alerts.
func supervisorEmailFromEnv() string {
	return os.Getenv("NOTIFY_SUPERVISOR_EMAIL")
}
