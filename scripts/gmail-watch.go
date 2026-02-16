// Program gmail-watch sets up Gmail push notifications for the billing inbox.
// Run with: go run ./scripts/gmail-watch.go (or build and run)
// Requires: GMAIL_INBOX_USER, GMAIL_TOPIC_NAME (e.g. projects/PROJECT_ID/topics/gmail-watch), GOOGLE_APPLICATION_CREDENTIALS.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func main() {
	user := os.Getenv("GMAIL_INBOX_USER")
	topic := os.Getenv("GMAIL_TOPIC_NAME")
	if user == "" || topic == "" {
		log.Fatal("set GMAIL_INBOX_USER and GMAIL_TOPIC_NAME (e.g. projects/PROJECT/topics/gmail-watch)")
	}

	ctx := context.Background()
	creds, err := google.FindDefaultCredentials(ctx, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("credentials: %v", err)
	}
	jwt, err := google.JWTConfigFromJSON(creds.JSON, gmail.GmailReadonlyScope)
	if err != nil {
		log.Fatalf("jwt config: %v", err)
	}
	jwt.Subject = user
	ts := jwt.TokenSource(ctx)
	svc, err := gmail.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		log.Fatalf("gmail service: %v", err)
	}

	resp, err := svc.Users.Watch(user, &gmail.WatchRequest{
		TopicName: topic,
	}).Do()
	if err != nil {
		log.Fatalf("watch: %v", err)
	}
	fmt.Printf("Watch set. HistoryID: %d, Expiration: %d\n", resp.HistoryId, resp.Expiration)
}
