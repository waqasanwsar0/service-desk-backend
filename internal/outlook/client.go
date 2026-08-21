// Package outlook implements the ticket-intake side of the SOW's Outlook
// integration: it polls a shared mailbox via the Microsoft Graph API for
// unread messages and turns each one into a ticket, using the same
// import-from-email path that already exists for manual/API use.
//
// It needs three values from an Azure App Registration (see the setup
// notes in README.md): a Tenant ID, a Client ID, and a Client Secret,
// plus the mailbox address to watch. All four are read from environment
// variables — nothing is hardcoded. If they aren't set, the poller
// simply doesn't start; every other part of the system is unaffected.
package outlook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	TenantID     string
	ClientID     string
	ClientSecret string
	Mailbox      string // the shared mailbox address to poll, e.g. support@viorasolutions.com
}

// FromEnv reads AZURE_TENANT_ID, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET,
// and AZURE_MAILBOX. Returns (nil, false) if any are missing — the
// caller should skip starting the poller in that case.
func FromEnv() (*Config, bool) {
	c := &Config{
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		Mailbox:      os.Getenv("AZURE_MAILBOX"),
	}
	if c.TenantID == "" || c.ClientID == "" || c.ClientSecret == "" || c.Mailbox == "" {
		return nil, false
	}
	return c, true
}

type Client struct {
	cfg        *Config
	httpClient *http.Client

	token       string
	tokenExpiry time.Time
}

func NewClient(cfg *Config) *Client {
	return &Client{cfg: cfg, httpClient: &http.Client{Timeout: 15 * time.Second}}
}

// accessToken returns a cached token, refreshing it via the OAuth2
// client-credentials flow when it's missing or about to expire.
func (c *Client) accessToken() (string, error) {
	if c.token != "" && time.Now().Before(c.tokenExpiry.Add(-60*time.Second)) {
		return c.token, nil
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", c.cfg.TenantID)
	form := url.Values{
		"client_id":     {c.cfg.ClientID},
		"client_secret": {c.cfg.ClientSecret},
		"scope":         {"https://graph.microsoft.com/.default"},
		"grant_type":    {"client_credentials"},
	}

	resp, err := c.httpClient.PostForm(tokenURL, form)
	if err != nil {
		return "", fmt.Errorf("requesting token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token request failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing token response: %w", err)
	}

	c.token = result.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return c.token, nil
}

// Message is the subset of a Graph API email message this integration
// cares about.
type Message struct {
	ID          string `json:"id"`
	Subject     string `json:"subject"`
	BodyPreview string `json:"bodyPreview"`
	From        struct {
		EmailAddress struct {
			Name    string `json:"name"`
			Address string `json:"address"`
		} `json:"emailAddress"`
	} `json:"from"`
	ReceivedDateTime time.Time `json:"receivedDateTime"`
}

// FetchUnread returns unread messages from the configured mailbox's inbox.
func (c *Client) FetchUnread() ([]Message, error) {
	token, err := c.accessToken()
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf(
		"https://graph.microsoft.com/v1.0/users/%s/mailFolders/inbox/messages?$filter=isRead eq false&$top=25&$select=id,subject,bodyPreview,from,receivedDateTime",
		url.PathEscape(c.cfg.Mailbox),
	)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching messages: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching messages failed (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Value []Message `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing messages response: %w", err)
	}
	return result.Value, nil
}

// MarkRead flags a message as read so it isn't imported again on the
// next poll.
func (c *Client) MarkRead(messageID string) error {
	token, err := c.accessToken()
	if err != nil {
		return err
	}

	endpoint := fmt.Sprintf("https://graph.microsoft.com/v1.0/users/%s/messages/%s",
		url.PathEscape(c.cfg.Mailbox), url.PathEscape(messageID))
	payload, _ := json.Marshal(map[string]bool{"isRead": true})

	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("marking message read: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("marking message read failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// ClientNameFromMessage picks a reasonable client_name for a ticket
// created from this email — prefers the sender's display name, falls
// back to their email address, and finally the domain.
func ClientNameFromMessage(m Message) string {
	if m.From.EmailAddress.Name != "" {
		return m.From.EmailAddress.Name
	}
	if m.From.EmailAddress.Address != "" {
		return m.From.EmailAddress.Address
	}
	return "Unknown sender"
}

// TicketTitle builds a ticket title from the subject, falling back to a
// generic label if the email had none.
func TicketTitle(m Message) string {
	subject := strings.TrimSpace(m.Subject)
	if subject == "" {
		return "Email ticket (no subject)"
	}
	return subject
}
