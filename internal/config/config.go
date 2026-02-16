package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds application configuration from env and optional file.
// The server uses Application Default Credentials (e.g. the service account
// attached to the Cloud Run service); no credential path is configured here.
type Config struct {
	// Server
	Port string

	// Gmail (inbox to read from; same identity is used to send notifications via Gmail API)
	GmailTopicName   string
	GmailInboxUser   string // required: inbox address, e.g. billing@example.com
	HistoryIDDocPath string // document ID in ConfigCollection for storing last history ID

	// Firestore
	FirestoreProjectID  string
	RoommatesCollection string
	BillsCollection      string
	// ConfigCollection is the Firestore collection for app config documents (e.g. the
	// gmail_history doc that stores the last processed Gmail history ID for sync).
	ConfigCollection string

	// Filters
	Filters FilterSpec
}

// FilterSpec defines which messages are treated as bills.
type FilterSpec struct {
	BillerSenders []string
	Keywords      []string
	LabelIDs      []string
}

// Load reads configuration from environment and optional config file path.
// GmailInboxUser is required (no default) so the inbox address is not committed to the repo.
func Load(configPath string) (*Config, error) {
	c := &Config{
		Port:                getEnv("PORT", "8080"),
		GmailTopicName:      getEnv("GMAIL_TOPIC_NAME", ""),
		GmailInboxUser:      getEnv("GMAIL_INBOX_USER", ""),
		HistoryIDDocPath:    getEnv("HISTORY_ID_DOC_PATH", "gmail_history"),
		FirestoreProjectID:  getEnv("FIRESTORE_PROJECT_ID", ""),
		RoommatesCollection: getEnv("ROOMMATES_COLLECTION", "roommates"),
		BillsCollection:     getEnv("BILLS_COLLECTION", "bills"),
		ConfigCollection:    getEnv("CONFIG_COLLECTION", "config"),
	}

	if c.GmailInboxUser == "" {
		return nil, fmt.Errorf("GMAIL_INBOX_USER is required")
	}

	if v := getEnv("FILTER_BILLER_SENDERS", ""); v != "" {
		c.Filters.BillerSenders = splitTrim(v)
	}
	if v := getEnv("FILTER_KEYWORDS", ""); v != "" {
		c.Filters.Keywords = splitTrim(v)
	}
	if v := getEnv("FILTER_LABEL_IDS", ""); v != "" {
		c.Filters.LabelIDs = splitTrim(v)
	}

	if configPath != "" {
		if err := loadFile(configPath, c); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("config file %s: %w", configPath, err)
		}
	}

	return c, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func splitTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
