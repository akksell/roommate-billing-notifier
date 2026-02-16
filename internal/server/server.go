package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/akksell/rbn/internal/bill"
	"github.com/akksell/rbn/internal/config"
	"github.com/akksell/rbn/internal/filter"
	"github.com/akksell/rbn/internal/gmail"
	"github.com/akksell/rbn/internal/notify"
	"github.com/akksell/rbn/internal/pubsub"
	"github.com/akksell/rbn/internal/split"
	"github.com/akksell/rbn/internal/store"
)

// Server is the HTTP handler for Pub/Sub push and health.
type Server struct {
	cfg     *config.Config
	store   *store.Store
	gmail   *gmail.Client
	extract *bill.Extractor
	notify  *notify.Sender
}

// New builds the HTTP server with push and health handlers.
func New(cfg *config.Config, st *store.Store, gm *gmail.Client, ext *bill.Extractor, n *notify.Sender) (*Server, error) {
	return &Server{cfg: cfg, store: st, gmail: gm, extract: ext, notify: n}, nil
}

// ServeHTTP routes requests.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/health":
		if r.Method == http.MethodGet {
			s.health(w, r)
			return
		}
	case r.URL.Path == "/" || r.URL.Path == "/push":
		if r.Method == http.MethodPost {
			s.push(w, r)
			return
		}
		if r.URL.Path == "/" && r.Method == http.MethodGet {
			s.health(w, r)
			return
		}
	case strings.HasPrefix(r.URL.Path, "/bills/") && strings.HasSuffix(r.URL.Path, "/paid"):
		if r.Method == http.MethodPost || r.Method == http.MethodPatch {
			s.markDebtPaid(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// PubSubPushMessage is the payload sent by Pub/Sub push subscription.
type PubSubPushMessage struct {
	Message struct {
		Data       []byte            `json:"data"`
		MessageID  string            `json:"messageId"`
		Attributes map[string]string  `json:"attributes,omitempty"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

func (s *Server) push(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body PubSubPushMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("push decode: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	pushData, err := pubsub.DecodePushData(body.Message.Data)
	if err != nil {
		log.Printf("push data decode: %v", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := s.processPush(ctx, pushData); err != nil {
		log.Printf("process push: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) processPush(ctx context.Context, push *pubsub.GmailPushData) error {
	startHistoryID, err := s.store.GetHistoryID(ctx)
	if err != nil {
		return err
	}
	if startHistoryID == "" {
		startHistoryID = "1"
	}

	messageIDs, newHistoryID, err := s.gmail.HistoryList(ctx, startHistoryID)
	if err != nil {
		return err
	}

	for _, msgID := range messageIDs {
		if err := s.processMessage(ctx, msgID); err != nil {
			log.Printf("process message %s: %v", msgID, err)
		}
	}

	if newHistoryID != "" {
		if err := s.store.SetHistoryID(ctx, newHistoryID); err != nil {
			return err
		}
	} else if push.HistoryID != "" {
		if err := s.store.SetHistoryID(ctx, push.HistoryID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) processMessage(ctx context.Context, messageID string) error {
	msg, err := s.gmail.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if !filter.Match(&s.cfg.Filters, msg) {
		return nil
	}

	html, plain := gmail.GetMessageBody(msg)
	extracted, ok := s.extract.Extract(msg, html, plain)
	if !ok {
		return nil
	}

	roommates, err := s.store.ListActiveRoommates(ctx)
	if err != nil {
		return err
	}
	if len(roommates) == 0 {
		return nil
	}

	debts := split.Split(extracted.TotalAmount, roommates)

	billDoc := &store.Bill{
		BillerCompany:   extracted.BillerCompany,
		TotalAmount:    extracted.TotalAmount,
		Status:         store.BillStatusUnpaid,
		DueDate:        extracted.DueDate,
		DateReceived:   time.Now(),
		GmailMessageID: messageID,
		Currency:       "USD",
		CreatedAt:     time.Now(),
	}

	if err := s.store.SaveBill(ctx, billDoc, debts); err != nil {
		return err
	}

	dueStr := ""
	if !extracted.DueDate.IsZero() {
		dueStr = extracted.DueDate.Format("2006-01-02")
	}
	excerpt := plain
	if excerpt == "" {
		excerpt = html
	}
	if len(excerpt) > 2000 {
		excerpt = excerpt[:2000] + "..."
	}

		for i, d := range debts {
			_ = s.notify.SendBillNotification(ctx, roommates[i], billDoc.BillerCompany, d.Amount, dueStr, excerpt)
		}

	return nil
}

// markDebtPaid handles POST/PATCH /bills/{billId}/debts/{roommateId}/paid
func (s *Server) markDebtPaid(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/bills/")
	path = strings.TrimSuffix(path, "/paid")
	parts := strings.Split(path, "/debts/")
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	billID := strings.TrimSuffix(parts[0], "/")
	roommateID := parts[1]
	if billID == "" || roommateID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := s.store.MarkDebtPaid(r.Context(), billID, roommateID, time.Now(), ""); err != nil {
		log.Printf("mark debt paid: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
