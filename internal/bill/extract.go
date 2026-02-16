package bill

import (
	"regexp"
	"time"

	"google.golang.org/api/gmail/v1"
)

// Extracted holds parsed bill fields from an email.
type Extracted struct {
	TotalAmount float64
	DueDate     time.Time
	BillerCompany string
}

// Extractor parses bill fields from email HTML/body. Per-biller logic can be added later.
type Extractor struct {
	// TotalRegex is used to find the total amount in body (e.g. "Total: $123.45").
	TotalRegex *regexp.Regexp
}

// DefaultExtractor returns an extractor with a default total pattern.
func DefaultExtractor() *Extractor {
	return &Extractor{
		TotalRegex: regexp.MustCompile(`(?i)(?:total|amount due|balance)[:\s]*\$?\s*([\d,]+(?:\.\d{2})?)`),
	}
}

// Extract runs the extractor on the message and returns bill fields if found.
func (e *Extractor) Extract(msg *gmail.Message, html, plain string) (*Extracted, bool) {
	body := html
	if body == "" {
		body = plain
	}
	if body == "" {
		return nil, false
	}

	out := &Extracted{}
	out.BillerCompany = getHeader(msg, "From")

	if e.TotalRegex != nil {
		matches := e.TotalRegex.FindStringSubmatch(body)
		if len(matches) < 2 {
			return nil, false
		}
		var total float64
		if err := parseDecimal(matches[1], &total); err != nil {
			return nil, false
		}
		out.TotalAmount = total
	}

	// Due date: optional; can be extended with per-biller rules
	// out.DueDate left as zero value

	return out, true
}

func getHeader(msg *gmail.Message, name string) string {
	if msg.Payload == nil {
		return ""
	}
	for _, h := range msg.Payload.Headers {
		if h.Name == name {
			return h.Value
		}
	}
	return ""
}
