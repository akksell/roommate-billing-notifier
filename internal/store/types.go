package store

import (
	"time"
)

// Roommate represents a roommate document from the roommates collection.
type Roommate struct {
	ID          string `firestore:"-"` // document ID, set from Ref
	Email       string `firestore:"email"`
	DisplayName string `firestore:"displayName"`
	Active      bool   `firestore:"active"`
}

// Bill represents a bill document in the bills collection.
type Bill struct {
	ID              string    `firestore:"-"` // document ID, set from Ref
	BillerCompany   string    `firestore:"billerCompany"`
	TotalAmount     float64   `firestore:"totalAmount"`
	Status          string    `firestore:"status"` // unpaid, partial, paid (derived)
	DueDate         time.Time `firestore:"dueDate"`
	DateReceived    time.Time `firestore:"dateReceived"`
	GmailMessageID  string    `firestore:"gmailMessageId"`
	Currency        string    `firestore:"currency"`
	CreatedAt       time.Time `firestore:"createdAt"`
}

// Debt represents a roommate's debt for a bill (bills/{billId}/debts).
type Debt struct {
	RoommateID string     `firestore:"roommateId"`
	Amount     float64    `firestore:"amount"`
	Status     string     `firestore:"status"` // pending, paid
	PaidAt     *time.Time `firestore:"paidAt,omitempty"`
	PaidBy     string     `firestore:"paidBy,omitempty"`
}

// BillStatusUnpaid is the derived status when no roommate has paid.
const BillStatusUnpaid = "unpaid"

// BillStatusPartial is the derived status when some roommates have paid.
const BillStatusPartial = "partial"

// BillStatusPaid is the derived status when all roommates have paid.
const BillStatusPaid = "paid"

// DebtStatusPending is the debt status before payment.
const DebtStatusPending = "pending"

// DebtStatusPaid is the debt status after payment.
const DebtStatusPaid = "paid"
