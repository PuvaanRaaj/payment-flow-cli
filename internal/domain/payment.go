// Package domain contains the core business logic and entities for the payment processing system.
package domain

import (
	"fmt"
	"math/big"
	"time"
)

// Payment states as string constants
const (
	StateInitiated           = "INITIATED"
	StateAuthorized          = "AUTHORIZED"
	StatePreSettlementReview = "PRE_SETTLEMENT_REVIEW"
	StateCaptured            = "CAPTURED"
	StateSettled             = "SETTLED"
	StateVoided              = "VOIDED"
	StateRefunded            = "REFUNDED"
	StateFailed              = "FAILED"
)

// HistoryEntry represents a single state change in the payment lifecycle.
type HistoryEntry struct {
	Timestamp time.Time
	FromState string
	ToState   string
	Action    string
	Details   string
}

// Payment represents a payment in the system.
type Payment struct {
	ID         string
	Amount     *big.Rat
	Currency   string
	MerchantID string
	State      string
	VoidReason string
	History    []HistoryEntry
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// NewPayment creates a new payment in the INITIATED state.
func NewPayment(id string, amount *big.Rat, currency, merchantID string) *Payment {
	now := time.Now()
	p := &Payment{
		ID:         id,
		Amount:     amount,
		Currency:   currency,
		MerchantID: merchantID,
		State:      StateInitiated,
		History:    make([]HistoryEntry, 0),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	p.addHistory("", StateInitiated, "CREATE", "Payment created")
	return p
}

// addHistory adds a new entry to the payment's history.
func (p *Payment) addHistory(from, to, action, details string) {
	p.History = append(p.History, HistoryEntry{
		Timestamp: time.Now(),
		FromState: from,
		ToState:   to,
		Action:    action,
		Details:   details,
	})
}

// TransitionTo attempts to transition the payment to a new state.
func (p *Payment) TransitionTo(newState, action, details string) error {
	if err := ValidateTransition(p.State, newState); err != nil {
		return err
	}
	oldState := p.State
	p.State = newState
	p.UpdatedAt = time.Now()
	p.addHistory(oldState, newState, action, details)
	return nil
}

// SetFailed marks the payment as failed with a reason.
func (p *Payment) SetFailed(reason string) {
	oldState := p.State
	p.State = StateFailed
	p.UpdatedAt = time.Now()
	p.addHistory(oldState, StateFailed, "FAIL", reason)
}

// SetVoidReason sets the void reason for the payment.
func (p *Payment) SetVoidReason(reason string) {
	p.VoidReason = reason
}

// FormatAmount returns the amount as a formatted string.
func (p *Payment) FormatAmount() string {
	return FormatRat(p.Amount)
}

// Equals checks if two payments have the same creation attributes.
func (p *Payment) Equals(other *Payment) bool {
	if p.ID != other.ID {
		return false
	}
	if p.Amount.Cmp(other.Amount) != 0 {
		return false
	}
	if p.Currency != other.Currency {
		return false
	}
	if p.MerchantID != other.MerchantID {
		return false
	}
	return true
}

// ParseAmount parses a string amount into a *big.Rat.
func ParseAmount(s string) (*big.Rat, error) {
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return nil, fmt.Errorf("invalid amount format: %s", s)
	}
	// Validate it's positive
	if r.Sign() <= 0 {
		return nil, fmt.Errorf("amount must be positive: %s", s)
	}
	return r, nil
}

// FormatRat formats a *big.Rat as a decimal string.
func FormatRat(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	// Get the float representation and format it
	f, _ := r.Float64()
	// Format with appropriate precision
	s := fmt.Sprintf("%.10f", f)
	// Trim trailing zeros after decimal point
	for len(s) > 1 && s[len(s)-1] == '0' && s[len(s)-2] != '.' {
		s = s[:len(s)-1]
	}
	return s
}
