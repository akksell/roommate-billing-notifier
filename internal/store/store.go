package store

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	roommatesCollection = "roommates"
	billsCollection     = "bills"
	historyIDDocPath    = "gmail_history"
	configCollection    = "config"
)

// Store handles Firestore access for roommates, bills, debts, and history ID.
type Store struct {
	client *firestore.Client
}

// New creates a Store using the given Firestore client.
func New(client *firestore.Client) *Store {
	return &Store{client: client}
}

// ListActiveRoommates returns roommates where active is true or not set.
func (s *Store) ListActiveRoommates(ctx context.Context) ([]Roommate, error) {
	col := s.client.Collection(roommatesCollection)
	iter := col.Documents(ctx)
	defer iter.Stop()

	var out []Roommate
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		data := doc.Data()
		if v, ok := data["active"].(bool); ok && !v {
			continue
		}
		var r Roommate
		if err := doc.DataTo(&r); err != nil {
			continue
		}
		r.ID = doc.Ref.ID
		r.Active = true
		if v, ok := data["active"].(bool); ok {
			r.Active = v
		}
		out = append(out, r)
	}
	return out, nil
}

// GetHistoryID returns the stored Gmail history ID, or empty if none.
func (s *Store) GetHistoryID(ctx context.Context) (string, error) {
	docRef := s.client.Collection(configCollection).Doc(historyIDDocPath)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", err
	}
	v, ok := doc.Data()["historyId"]
	if !ok {
		return "", nil
	}
	str, _ := v.(string)
	return str, nil
}

// SetHistoryID saves the Gmail history ID for the next sync.
func (s *Store) SetHistoryID(ctx context.Context, historyID string) error {
	docRef := s.client.Collection(configCollection).Doc(historyIDDocPath)
	_, err := docRef.Set(ctx, map[string]interface{}{"historyId": historyID})
	return err
}
