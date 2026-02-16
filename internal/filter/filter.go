package filter

import (
	"strings"

	"github.com/akksell/rbn/internal/config"
	"google.golang.org/api/gmail/v1"
)

// Match returns true if the Gmail message matches the filter spec (biller senders, keywords, labels).
func Match(cfg *config.FilterSpec, msg *gmail.Message) bool {
	if cfg == nil {
		return true
	}

	from := getHeader(msg, "From")
	subject := getHeader(msg, "Subject")
	snippet := ""
	if msg.Snippet != "" {
		snippet = msg.Snippet
	}
	body := from + " " + subject + " " + snippet
	labelIDs := make(map[string]bool)
	for _, id := range msg.LabelIds {
		labelIDs[id] = true
	}

	if len(cfg.BillerSenders) > 0 {
		matched := false
		for _, allow := range cfg.BillerSenders {
			if strings.Contains(strings.ToLower(from), strings.ToLower(allow)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(cfg.Keywords) > 0 {
		lower := strings.ToLower(body)
		for _, kw := range cfg.Keywords {
			if !strings.Contains(lower, strings.ToLower(kw)) {
				return false
			}
		}
	}

	if len(cfg.LabelIDs) > 0 {
		for _, need := range cfg.LabelIDs {
			if !labelIDs[need] {
				return false
			}
		}
	}

	return true
}

func getHeader(msg *gmail.Message, name string) string {
	if msg.Payload == nil {
		return ""
	}
	for _, h := range msg.Payload.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}
