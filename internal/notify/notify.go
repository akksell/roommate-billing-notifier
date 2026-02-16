package notify

import (
	"context"
	"fmt"
	"strconv"

	"github.com/akksell/rbn/internal/config"
	"github.com/akksell/rbn/internal/gmail"
	"github.com/akksell/rbn/internal/store"
)

// Sender sends notification emails to roommates via the Gmail API (application default credentials).
type Sender struct {
	cfg   *config.Config
	gmail *gmail.Client
}

// NewSender creates a Sender that uses the Gmail API to send as the inbox user.
func NewSender(cfg *config.Config, gmailClient *gmail.Client) *Sender {
	return &Sender{cfg: cfg, gmail: gmailClient}
}

// SendBillNotification sends an email to the roommate with their share and bill details.
// originalBody can be the raw message or a snippet to include.
func (s *Sender) SendBillNotification(ctx context.Context, to store.Roommate, billerCompany string, amount float64, dueDate string, originalBody string) error {
	subject := fmt.Sprintf("Bill split: %s - Your share $%s", billerCompany, formatAmount(amount))
	body := fmt.Sprintf("Your share for the bill from %s is $%s.\n", billerCompany, formatAmount(amount))
	if to.DisplayName != "" {
		body = fmt.Sprintf("Hi %s,\n\nYour share for the bill from %s is $%s.\n", to.DisplayName, billerCompany, formatAmount(amount))
	}
	if dueDate != "" {
		body += fmt.Sprintf("Due date: %s\n", dueDate)
	}
	body += "\n--- Original message excerpt ---\n"
	body += originalBody

	return s.gmail.SendMessage(ctx, s.cfg.GmailInboxUser, to.Email, subject, body)
}

func formatAmount(a float64) string {
	return strconv.FormatFloat(a, 'f', 2, 64)
}
