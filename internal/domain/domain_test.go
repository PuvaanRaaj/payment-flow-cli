package domain

import (
	"math/big"
	"testing"
)

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		want    string // Expected formatted output
	}{
		{
			name:    "valid integer",
			input:   "100",
			wantErr: false,
			want:    "100.0",
		},
		{
			name:    "valid decimal",
			input:   "10.50",
			wantErr: false,
			want:    "10.5",
		},
		{
			name:    "valid small decimal",
			input:   "0.01",
			wantErr: false,
			want:    "0.01",
		},
		{
			name:    "zero - should fail",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "negative - should fail",
			input:   "-10.00",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAmount(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAmount() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && FormatRat(got) != tt.want {
				t.Errorf("ParseAmount() = %v, want %v", FormatRat(got), tt.want)
			}
		})
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		// Valid transitions
		{"INITIATED to AUTHORIZED", StateInitiated, StateAuthorized, true},
		{"INITIATED to VOIDED", StateInitiated, StateVoided, true},
		{"INITIATED to FAILED", StateInitiated, StateFailed, true},
		{"AUTHORIZED to CAPTURED", StateAuthorized, StateCaptured, true},
		{"AUTHORIZED to PRE_SETTLEMENT_REVIEW", StateAuthorized, StatePreSettlementReview, true},
		{"AUTHORIZED to VOIDED", StateAuthorized, StateVoided, true},
		{"PRE_SETTLEMENT_REVIEW to CAPTURED", StatePreSettlementReview, StateCaptured, true},
		{"CAPTURED to SETTLED", StateCaptured, StateSettled, true},
		{"CAPTURED to REFUNDED", StateCaptured, StateRefunded, true},
		{"SETTLED to SETTLED (idempotent)", StateSettled, StateSettled, true},

		// Invalid transitions
		{"INITIATED to CAPTURED", StateInitiated, StateCaptured, false},
		{"INITIATED to SETTLED", StateInitiated, StateSettled, false},
		{"AUTHORIZED to SETTLED", StateAuthorized, StateSettled, false},
		{"CAPTURED to AUTHORIZED", StateCaptured, StateAuthorized, false},
		{"CAPTURED to VOIDED", StateCaptured, StateVoided, false},
		{"VOIDED to anything", StateVoided, StateAuthorized, false},
		{"REFUNDED to anything", StateRefunded, StateSettled, false},
		{"FAILED to anything", StateFailed, StateInitiated, false},
		{"SETTLED to REFUNDED", StateSettled, StateRefunded, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransition(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestPaymentTransitionTo(t *testing.T) {
	amount := big.NewRat(100, 1)
	p := NewPayment("P001", amount, "USD", "M001")

	// Valid transition
	err := p.TransitionTo(StateAuthorized, "AUTHORIZE", "Payment authorized")
	if err != nil {
		t.Errorf("TransitionTo() unexpected error: %v", err)
	}
	if p.State != StateAuthorized {
		t.Errorf("State = %v, want %v", p.State, StateAuthorized)
	}
	if len(p.History) != 2 { // CREATE + AUTHORIZE
		t.Errorf("History length = %v, want 2", len(p.History))
	}

	// Invalid transition
	err = p.TransitionTo(StateSettled, "SETTLE", "")
	if err == nil {
		t.Errorf("TransitionTo() expected error for invalid transition")
	}
	if p.State != StateAuthorized {
		t.Errorf("State changed on invalid transition: %v", p.State)
	}
}

func TestPaymentEquals(t *testing.T) {
	amount1 := big.NewRat(10050, 100) // 100.50
	amount2 := big.NewRat(10050, 100)
	amount3 := big.NewRat(10000, 100)

	p1 := NewPayment("P001", amount1, "USD", "M001")
	p2 := NewPayment("P001", amount2, "USD", "M001")
	p3 := NewPayment("P001", amount3, "USD", "M001")
	p4 := NewPayment("P001", amount1, "EUR", "M001")
	p5 := NewPayment("P002", amount1, "USD", "M001")

	if !p1.Equals(p2) {
		t.Error("p1 should equal p2 (same attributes)")
	}
	if p1.Equals(p3) {
		t.Error("p1 should not equal p3 (different amount)")
	}
	if p1.Equals(p4) {
		t.Error("p1 should not equal p4 (different currency)")
	}
	if p1.Equals(p5) {
		t.Error("p1 should not equal p5 (different ID)")
	}
}

func TestNewPayment(t *testing.T) {
	amount := big.NewRat(5000, 100) // 50.00
	p := NewPayment("P001", amount, "MYR", "M001")

	if p.ID != "P001" {
		t.Errorf("ID = %v, want P001", p.ID)
	}
	if p.State != StateInitiated {
		t.Errorf("State = %v, want INITIATED", p.State)
	}
	if p.Currency != "MYR" {
		t.Errorf("Currency = %v, want MYR", p.Currency)
	}
	if p.MerchantID != "M001" {
		t.Errorf("MerchantID = %v, want M001", p.MerchantID)
	}
	if len(p.History) != 1 {
		t.Errorf("History length = %v, want 1", len(p.History))
	}
}

func TestSetFailed(t *testing.T) {
	amount := big.NewRat(100, 1)
	p := NewPayment("P001", amount, "USD", "M001")

	p.SetFailed("create conflict")

	if p.State != StateFailed {
		t.Errorf("State = %v, want FAILED", p.State)
	}
	if len(p.History) != 2 {
		t.Errorf("History length = %v, want 2", len(p.History))
	}
}

func TestSetVoidReason(t *testing.T) {
	amount := big.NewRat(100, 1)
	p := NewPayment("P001", amount, "USD", "M001")

	p.SetVoidReason("CUSTOMER_REQUEST")

	if p.VoidReason != "CUSTOMER_REQUEST" {
		t.Errorf("VoidReason = %v, want CUSTOMER_REQUEST", p.VoidReason)
	}
}

func TestFormatAmount(t *testing.T) {
	amount := big.NewRat(10050, 100) // 100.50
	p := NewPayment("P001", amount, "USD", "M001")

	formatted := p.FormatAmount()
	if formatted != "100.5" {
		t.Errorf("FormatAmount() = %v, want 100.5", formatted)
	}
}

func TestFormatRat_Nil(t *testing.T) {
	result := FormatRat(nil)
	if result != "0" {
		t.Errorf("FormatRat(nil) = %v, want 0", result)
	}
}

func TestPaymentEquals_DifferentMerchant(t *testing.T) {
	amount := big.NewRat(100, 1)
	p1 := NewPayment("P001", amount, "USD", "M001")
	p2 := NewPayment("P001", amount, "USD", "M002")

	if p1.Equals(p2) {
		t.Error("p1 should not equal p2 (different merchant)")
	}
}

func TestCanTransition_UnknownState(t *testing.T) {
	// Test transition from unknown state
	if CanTransition("UNKNOWN_STATE", StateAuthorized) {
		t.Error("CanTransition from unknown state should return false")
	}
}

// Test error types
func TestInvalidTransitionError(t *testing.T) {
	err := NewInvalidTransitionError("INITIATED", "SETTLED")
	expected := "invalid transition from INITIATED to SETTLED"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestCreateConflictError(t *testing.T) {
	err := NewCreateConflictError("P001")
	expected := "create conflict for payment P001: existing payment marked as FAILED"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestParseError(t *testing.T) {
	err := NewParseError("invalid command")
	expected := "invalid command"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("amount", "must be positive")
	expected := "validation error for amount: must be positive"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}
