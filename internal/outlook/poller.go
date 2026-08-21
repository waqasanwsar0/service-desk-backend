package outlook

import (
	"log"
	"time"

	"servicedesk/internal/models"
	"servicedesk/internal/store"
)

// StartPoller runs FetchUnread/MarkRead on a fixed interval for as long
// as the process runs. It's designed to be launched with `go
// outlook.StartPoller(...)` from main.go and never returns.
func StartPoller(client *Client, tickets store.TicketStore, interval time.Duration) {
	log.Printf("Outlook poller started (mailbox: %s, every %s)", client.cfg.Mailbox, interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately, then on every tick.
	pollOnce(client, tickets)
	for range ticker.C {
		pollOnce(client, tickets)
	}
}

func pollOnce(client *Client, tickets store.TicketStore) {
	messages, err := client.FetchUnread()
	if err != nil {
		log.Printf("Outlook poller: fetch failed: %v", err)
		return
	}
	if len(messages) == 0 {
		return
	}
	log.Printf("Outlook poller: %d unread message(s) found", len(messages))

	for _, m := range messages {
		t := &models.Ticket{
			Title:       TicketTitle(m),
			Description: m.BodyPreview,
			ClientName:  ClientNameFromMessage(m),
			Priority:    models.PriorityMedium,
			Source:      models.SourceEmail,
		}
		if err := tickets.Create(t); err != nil {
			log.Printf("Outlook poller: failed creating ticket for message %s: %v", m.ID, err)
			continue
		}
		if err := client.MarkRead(m.ID); err != nil {
			// Ticket was created but marking read failed — log it; worst
			// case the same email is imported again next poll, which is
			// safer than silently losing a ticket.
			log.Printf("Outlook poller: created %s but failed marking message %s read: %v", t.ID, m.ID, err)
			continue
		}
		log.Printf("Outlook poller: created %s from email %q", t.ID, m.Subject)
	}
}
