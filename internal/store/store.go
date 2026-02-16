package store

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/akksell/rbn/internal/config"
	"google.golang.org/api/iterator"
)

// Store handles Firestore access for roommates, bills, debts, and history ID.
type Store struct {
	client *firestore.Client
	cfg    *config.Config
}

// New creates a Store using the given Firestore client and config.
func New(client *firestore.Client, cfg *config.Config) *Store {
	return &Store{client: client, cfg: cfg}
}

// ListActiveRoommates returns roommates where active is true or not set.
func (s *Store) ListActiveRoommates(ctx context.Context) ([]Roommate, error) {
	col := s.client.Collection(s.cfg.RoommatesCollection)
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
	docRef := s.client.Collection(s.cfg.ConfigCollection).Doc(s.cfg.HistoryIDDocPath)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if err == firestore.ErrNotFound {
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
	docRef := s.client.Collection(s.cfg.ConfigCollection).Doc(s.cfg.HistoryIDDocPath)
	_, err := docRef.Set(ctx, map[string]interface{}{"historyId": historyID})
	return err
}
