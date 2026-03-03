package config

import (
	"context"
	"fmt"
	"io"
	"os"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	"gopkg.in/yaml.v3"
)

// Config holds application configuration.
// The server uses Application Default Credentials (e.g. the service account
// attached to the Cloud Run service); no credential path is configured here.
type Config struct {
	Port               string     // env: PORT
	FirestoreProjectID string     // env: FIRESTORE_PROJECT_ID
	GmailTopicName     string     // env: GMAIL_TOPIC_NAME
	GmailInboxUser     string     // Secret Manager: gmail-inbox-user
	Filters            FilterSpec // GCS: gs://$CONFIG_BUCKET/config.yaml
}

// FilterSpec defines which messages are treated as bills.
type FilterSpec struct {
	BillerSenders []string `yaml:"billerSenders"`
	Keywords      []string `yaml:"keywords"`
	LabelIDs      []string `yaml:"labelIDs"`
}

type controlPlaneConfig struct {
	Filters FilterSpec `yaml:"filters"`
}

const (
	gmailInboxUserSecret = "gmail-inbox-user"
	gcsConfigObject      = "config.yaml"
)

// Load reads configuration from environment variables, Secret Manager, and GCS.
// Fails fast if any required value is missing or unreachable.
func Load(ctx context.Context) (*Config, error) {
	projectID := getEnv("FIRESTORE_PROJECT_ID", "")
	if projectID == "" {
		return nil, fmt.Errorf("FIRESTORE_PROJECT_ID is required")
	}

	configBucket := getEnv("CONFIG_BUCKET", "")
	if configBucket == "" {
		return nil, fmt.Errorf("CONFIG_BUCKET is required")
	}

	gmailTopicName := getEnv("GMAIL_TOPIC_NAME", "")
	if gmailTopicName == "" {
		return nil, fmt.Errorf("GMAIL_TOPIC_NAME is required")
	}

	inboxUser, err := fetchSecret(ctx, projectID, gmailInboxUserSecret)
	if err != nil {
		return nil, fmt.Errorf("gmail inbox user: %w", err)
	}

	filters, err := fetchGCSConfig(ctx, configBucket)
	if err != nil {
		return nil, fmt.Errorf("GCS config: %w", err)
	}

	return &Config{
		Port:               getEnv("PORT", "8080"),
		FirestoreProjectID: projectID,
		GmailTopicName:     gmailTopicName,
		GmailInboxUser:     inboxUser,
		Filters:            filters,
	}, nil
}

func fetchSecret(ctx context.Context, projectID, secretName string) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("create client: %w", err)
	}
	defer client.Close()

	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)
	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{Name: name})
	if err != nil {
		return "", fmt.Errorf("access version: %w", err)
	}
	return string(result.Payload.Data), nil
}

func fetchGCSConfig(ctx context.Context, bucket string) (FilterSpec, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return FilterSpec{}, fmt.Errorf("create client: %w", err)
	}
	defer client.Close()

	r, err := client.Bucket(bucket).Object(gcsConfigObject).NewReader(ctx)
	if err != nil {
		return FilterSpec{}, fmt.Errorf("open object: %w", err)
	}
	defer r.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		return FilterSpec{}, fmt.Errorf("read: %w", err)
	}

	var cp controlPlaneConfig
	if err := yaml.Unmarshal(data, &cp); err != nil {
		return FilterSpec{}, fmt.Errorf("parse yaml: %w", err)
	}
	return cp.Filters, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
