package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

var gmailScopes = []string{gmail.GmailReadonlyScope, gmail.GmailSendScope}

// Client wraps the Gmail API for reading messages and history.
type Client struct {
	svc    *gmail.Service
	userID string
}

// NewClient creates a Gmail client that impersonates the given user (for domain-wide delegation).
// Credentials are read from GOOGLE_APPLICATION_CREDENTIALS; the service account must have
// domain-wide delegation for the Gmail API.
func NewClient(ctx context.Context, userID string) (*Client, error) {
	creds, err := google.FindDefaultCredentials(ctx, gmailScopes...)
	if err != nil {
		return nil, fmt.Errorf("credentials: %w", err)
	}

	// If userID is set, create JWT config and use impersonation (domain-wide delegation).
	if userID != "" {
		jwt, err := google.JWTConfigFromJSON(creds.JSON, gmailScopes...)
		if err != nil {
			return nil, fmt.Errorf("jwt config: %w", err)
		}
		jwt.Subject = userID
		ts := jwt.TokenSource(ctx)
		svc, err := gmail.NewService(ctx, option.WithTokenSource(ts))
		if err != nil {
			return nil, fmt.Errorf("gmail service: %w", err)
		}
		return &Client{svc: svc, userID: userID}, nil
	}

	svc, err := gmail.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("gmail service: %w", err)
	}
	return &Client{svc: svc, userID: "me"}, nil
}

// HistoryList calls users.history.list with startHistoryId and returns message IDs that were added.
func (c *Client) HistoryList(ctx context.Context, startHistoryID string) (messageIDs []string, newHistoryID string, err error) {
	startID, _ := strconv.ParseUint(startHistoryID, 10, 64)
	call := c.svc.Users.History.List(c.userID).StartHistoryId(startID).HistoryTypes("messageAdded")
	var nextPage string
	for {
		if nextPage != "" {
			call = call.PageToken(nextPage)
		}
		resp, err := call.Context(ctx).Do()
		if err != nil {
			return nil, "", err
		}
		for _, h := range resp.History {
			for _, m := range h.MessagesAdded {
				if m.Message != nil && m.Message.Id != "" {
					messageIDs = append(messageIDs, m.Message.Id)
				}
			}
		}
		if resp.NextPageToken == "" {
			if resp.HistoryId != 0 {
				newHistoryID = fmt.Sprintf("%d", resp.HistoryId)
			}
			break
		}
		nextPage = resp.NextPageToken
	}
	return messageIDs, newHistoryID, nil
}

// GetMessage fetches a full message by ID.
func (c *Client) GetMessage(ctx context.Context, messageID string) (*gmail.Message, error) {
	return c.svc.Users.Messages.Get(c.userID, messageID).Format("full").Context(ctx).Do()
}

// SendMessage sends an email from the configured user (inbox identity) via Gmail API.
// from is the From address (typically the same as the inbox user); to, subject, and body are plain text.
func (c *Client) SendMessage(ctx context.Context, from, to, subject, body string) error {
	raw := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		mail.FormatAddress(from, ""), to, mimeEncodeSubject(subject), body)
	rawB64 := base64.RawURLEncoding.EncodeToString([]byte(raw))
	msg := &gmail.Message{Raw: rawB64}
	_, err := c.svc.Users.Messages.Send(c.userID, msg).Context(ctx).Do()
	return err
}

func mimeEncodeSubject(s string) string {
	// Simple ASCII subject; no need for MIME encoding for typical bill subjects
	return s
}

// GetMessageBody returns the HTML or plain body of the message.
func GetMessageBody(msg *gmail.Message) (html string, plain string) {
	if msg.Payload == nil {
		return "", ""
	}
	for _, p := range msg.Payload.Parts {
		if p.MimeType == "text/html" && p.Body != nil && p.Body.AttachmentId == "" {
			html = decodeBody(p.Body)
			break
		}
	}
	if html == "" && msg.Payload.Body != nil {
		if msg.Payload.MimeType == "text/html" {
			html = decodeBody(msg.Payload.Body)
		} else {
			plain = decodeBody(msg.Payload.Body)
		}
	}
	for _, p := range msg.Payload.Parts {
		if p.MimeType == "text/plain" && p.Body != nil && p.Body.AttachmentId == "" {
			plain = decodeBody(p.Body)
			break
		}
	}
	return html, plain
}

func decodeBody(b *gmail.MessagePartBody) string {
	if b == nil {
		return ""
	}
	data := b.Data
	if data == "" {
		return ""
	}
	// Gmail API returns base64url
	data = strings.ReplaceAll(data, "-", "+")
	data = strings.ReplaceAll(data, "_", "/")
	switch len(data) % 4 {
	case 2:
		data += "=="
	case 3:
		data += "="
	}
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return ""
	}
	return string(decoded)
}
