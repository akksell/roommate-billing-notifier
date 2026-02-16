package store

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

// SaveBill creates or updates a bill and its debts subcollection (idempotent by gmailMessageId).
func (s *Store) SaveBill(ctx context.Context, bill *Bill, debts []Debt) error {
	billCol := s.client.Collection(s.cfg.BillsCollection)

	// Use Gmail message ID as doc ID for idempotency if set
	docID := bill.ID
	if docID == "" && bill.GmailMessageID != "" {
		docID = bill.GmailMessageID
	}
	if docID == "" {
		docID = billCol.NewDoc().ID
	}

	ref := billCol.Doc(docID)
	bill.ID = docID

	data := map[string]interface{}{
		"billerCompany":   bill.BillerCompany,
		"totalAmount":     bill.TotalAmount,
		"status":          bill.Status,
		"dueDate":         bill.DueDate,
		"dateReceived":    bill.DateReceived,
		"gmailMessageId":  bill.GmailMessageID,
		"currency":        bill.Currency,
		"createdAt":       bill.CreatedAt,
	}

	_, err := ref.Set(ctx, data)
	if err != nil {
		return err
	}

	debtsCol := ref.Collection("debts")
	for _, d := range debts {
		debtRef := debtsCol.Doc(d.RoommateID)
		debtData := map[string]interface{}{
			"roommateId": d.RoommateID,
			"amount":     d.Amount,
			"status":     d.Status,
		}
		if d.PaidAt != nil {
			debtData["paidAt"] = *d.PaidAt
		}
		if d.PaidBy != "" {
			debtData["paidBy"] = d.PaidBy
		}
		if _, err := debtRef.Set(ctx, debtData); err != nil {
			return err
		}
	}

	return nil
}

// GetBillByGmailMessageID returns a bill doc ref if one exists with that gmailMessageId.
func (s *Store) GetBillByGmailMessageID(ctx context.Context, gmailMessageID string) (*firestore.DocumentRef, error) {
	iter := s.client.Collection(s.cfg.BillsCollection).
		Where("gmailMessageId", "==", gmailMessageID).
		Limit(1).
		Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err != nil {
		return nil, err
	}
	return doc.Ref, nil
}

// MarkDebtPaid updates a debt's status to paid and recomputes the bill's status.
func (s *Store) MarkDebtPaid(ctx context.Context, billID, roommateID string, paidAt time.Time, paidBy string) error {
	billRef := s.client.Collection(s.cfg.BillsCollection).Doc(billID)
	debtRef := billRef.Collection("debts").Doc(roommateID)

	_, err := debtRef.Update(ctx, []firestore.Update{
		{Path: "status", Value: DebtStatusPaid},
		{Path: "paidAt", Value: paidAt},
		{Path: "paidBy", Value: paidBy},
	})
	if err != nil {
		return err
	}

	return s.recomputeBillStatus(ctx, billRef)
}

// recomputeBillStatus reads all debts for the bill and sets bill.status to unpaid/partial/paid.
func (s *Store) recomputeBillStatus(ctx context.Context, billRef *firestore.DocumentRef) error {
	iter := billRef.Collection("debts").Documents(ctx)
	defer iter.Stop()

	var paid, total int
	for {
		doc, err := iter.Next()
		if err != nil {
			break
		}
		total++
		if doc.Data()["status"] == DebtStatusPaid {
			paid++
		}
	}

	status := BillStatusUnpaid
	if paid == total && total > 0 {
		status = BillStatusPaid
	} else if paid > 0 {
		status = BillStatusPartial
	}

	_, err := billRef.Update(ctx, []firestore.Update{{Path: "status", Value: status}})
	return err
}
