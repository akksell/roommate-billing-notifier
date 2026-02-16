package pubsub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// GmailPushData is the decoded payload from Gmail watch (message.data).
type GmailPushData struct {
	EmailAddress string `json:"emailAddress"`
	HistoryID    string `json:"historyId"`
}

// DecodePushData decodes the base64 message.data from a Pub/Sub push and parses the Gmail payload.
// historyId may be a number in JSON; it is normalized to a string.
func DecodePushData(b64 []byte) (*GmailPushData, error) {
	data, err := base64.StdEncoding.DecodeString(string(b64))
	if err != nil {
		return nil, err
	}
	var raw struct {
		EmailAddress string      `json:"emailAddress"`
		HistoryID    interface{} `json:"historyId"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	out := &GmailPushData{EmailAddress: raw.EmailAddress}
	switch v := raw.HistoryID.(type) {
	case string:
		out.HistoryID = v
	case float64:
		out.HistoryID = fmt.Sprintf("%.0f", v)
	default:
		out.HistoryID = ""
	}
	return out, nil
}
