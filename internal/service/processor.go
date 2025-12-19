// Package service provides command processing for the payment processing CLI.
package service

import (
	"fmt"
	"math/big"
	"strings"

	"payment-sim/internal/domain"
	"payment-sim/internal/parser"
	"payment-sim/internal/store"
)

// Processor handles command execution.
type Processor struct {
	store                  store.Repository
	preSettlementThreshold *big.Rat
}

// NewProcessor creates a new command processor.
// threshold can be nil to disable PRE_SETTLEMENT_REVIEW.
func NewProcessor(store store.Repository, threshold *big.Rat) *Processor {
	return &Processor{
		store:                  store,
		preSettlementThreshold: threshold,
	}
}

// Execute processes a parsed command and returns the result.
func (p *Processor) Execute(cmd *parser.Command) (string, error) {
	switch cmd.Name {
	case "CREATE":
		return p.handleCreate(cmd.Args)
	case "AUTHORIZE":
		return p.handleAuthorize(cmd.Args)
	case "CAPTURE":
		return p.handleCapture(cmd.Args)
	case "VOID":
		return p.handleVoid(cmd.Args)
	case "REFUND":
		return p.handleRefund(cmd.Args)
	case "SETTLE":
		return p.handleSettle(cmd.Args)
	case "SETTLEMENT":
		return p.handleSettlement(cmd.Args)
	case "STATUS":
		return p.handleStatus(cmd.Args)
	case "LIST":
		return p.handleList()
	case "AUDIT":
		return p.handleAudit(cmd.Args)
	case "EXIT":
		// This should be handled by the runner, not here
		return "", nil
	default:
		return "", fmt.Errorf("unknown command: %s", cmd.Name)
	}
}

// handleCreate handles the CREATE command.
func (p *Processor) handleCreate(args []string) (string, error) {
	if len(args) < 4 {
		return "", fmt.Errorf("CREATE requires 4 arguments: <payment_id> <amount> <currency> <merchant_id>")
	}

	paymentID := args[0]
	amountStr := args[1]
	currency := args[2]
	merchantID := args[3]

	// Validate currency (3 letters)
	if len(currency) != 3 {
		return "", fmt.Errorf("currency must be a 3-letter code: %s", currency)
	}

	// Validate merchant_id is non-empty
	if merchantID == "" {
		return "", fmt.Errorf("merchant_id cannot be empty")
	}

	// Parse amount
	amount, err := domain.ParseAmount(amountStr)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %v", err)
	}

	// Check for existing payment
	existing, err := p.store.Get(paymentID)
	if err == nil {
		// Payment exists - check for idempotency
		newPayment := domain.NewPayment(paymentID, amount, currency, merchantID)
		if existing.Equals(newPayment) {
			// Idempotent - same attributes, no error
			return fmt.Sprintf("Payment %s already exists (idempotent)", paymentID), nil
		}
		// Conflict - mark existing as FAILED and reject
		existing.SetFailed("create conflict")
		p.store.Save(existing)
		return "", domain.NewCreateConflictError(paymentID)
	}

	// Create new payment
	payment := domain.NewPayment(paymentID, amount, currency, merchantID)
	if err := p.store.Save(payment); err != nil {
		return "", fmt.Errorf("failed to save payment: %v", err)
	}

	return fmt.Sprintf("Payment %s created: %s %s", paymentID, payment.FormatAmount(), currency), nil
}

// handleAuthorize handles the AUTHORIZE command.
func (p *Processor) handleAuthorize(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("AUTHORIZE requires payment_id")
	}

	paymentID := args[0]
	payment, err := p.store.Get(paymentID)
	if err != nil {
		return "", fmt.Errorf("payment %s not found", paymentID)
	}

	// Transition to AUTHORIZED
	if err := payment.TransitionTo(domain.StateAuthorized, "AUTHORIZE", "Payment authorized"); err != nil {
		return "", err
	}

	// Check if PRE_SETTLEMENT_REVIEW is needed
	if p.preSettlementThreshold != nil && payment.Amount.Cmp(p.preSettlementThreshold) >= 0 {
		if err := payment.TransitionTo(domain.StatePreSettlementReview, "REVIEW", "Amount exceeds threshold"); err != nil {
			// This shouldn't happen, but handle gracefully
			return "", fmt.Errorf("failed to move to pre-settlement review: %v", err)
		}
		p.store.Save(payment)
		return fmt.Sprintf("Payment %s authorized and moved to PRE_SETTLEMENT_REVIEW", paymentID), nil
	}

	p.store.Save(payment)
	return fmt.Sprintf("Payment %s authorized", paymentID), nil
}

// handleCapture handles the CAPTURE command.
func (p *Processor) handleCapture(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("CAPTURE requires payment_id")
	}

	paymentID := args[0]
	payment, err := p.store.Get(paymentID)
	if err != nil {
		return "", fmt.Errorf("payment %s not found", paymentID)
	}

	// Valid from AUTHORIZED or PRE_SETTLEMENT_REVIEW
	if err := payment.TransitionTo(domain.StateCaptured, "CAPTURE", "Payment captured"); err != nil {
		return "", err
	}

	p.store.Save(payment)
	return fmt.Sprintf("Payment %s captured", paymentID), nil
}

// handleVoid handles the VOID command.
func (p *Processor) handleVoid(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("VOID requires payment_id")
	}

	paymentID := args[0]
	reasonCode := ""
	if len(args) > 1 {
		reasonCode = args[1]
	}

	payment, err := p.store.Get(paymentID)
	if err != nil {
		return "", fmt.Errorf("payment %s not found", paymentID)
	}

	// Valid from INITIATED or AUTHORIZED only
	if err := payment.TransitionTo(domain.StateVoided, "VOID", "Payment voided"); err != nil {
		return "", err
	}

	if reasonCode != "" {
		payment.SetVoidReason(reasonCode)
	}

	p.store.Save(payment)
	if reasonCode != "" {
		return fmt.Sprintf("Payment %s voided (reason: %s)", paymentID, reasonCode), nil
	}
	return fmt.Sprintf("Payment %s voided", paymentID), nil
}

// handleRefund handles the REFUND command.
func (p *Processor) handleRefund(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("REFUND requires payment_id")
	}

	paymentID := args[0]
	// Optional amount argument - we accept it but for simplicity mark as fully refunded
	refundAmountStr := ""
	if len(args) > 1 {
		refundAmountStr = args[1]
	}

	payment, err := p.store.Get(paymentID)
	if err != nil {
		return "", fmt.Errorf("payment %s not found", paymentID)
	}

	// Valid from CAPTURED only
	if err := payment.TransitionTo(domain.StateRefunded, "REFUND", "Payment refunded"); err != nil {
		return "", err
	}

	p.store.Save(payment)
	if refundAmountStr != "" {
		return fmt.Sprintf("Payment %s refunded (%s)", paymentID, refundAmountStr), nil
	}
	return fmt.Sprintf("Payment %s refunded", paymentID), nil
}

// handleSettle handles the SETTLE command.
func (p *Processor) handleSettle(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("SETTLE requires payment_id")
	}

	paymentID := args[0]
	payment, err := p.store.Get(paymentID)
	if err != nil {
		return "", fmt.Errorf("payment %s not found", paymentID)
	}

	// Check for idempotency: SETTLED -> SETTLED is allowed
	if payment.State == domain.StateSettled {
		return fmt.Sprintf("Payment %s already settled (idempotent)", paymentID), nil
	}

	// Valid from CAPTURED only
	if err := payment.TransitionTo(domain.StateSettled, "SETTLE", "Payment settled"); err != nil {
		return "", err
	}

	p.store.Save(payment)
	return fmt.Sprintf("Payment %s settled", paymentID), nil
}

// handleSettlement handles the SETTLEMENT command.
func (p *Processor) handleSettlement(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("SETTLEMENT requires batch_id")
	}

	batchID := args[0]

	// Record the batch ID (no state changes to payments)
	p.store.RecordBatchID(batchID)

	// Get all settled payments for summary
	payments, _ := p.store.List()
	settledCount := 0
	var totalAmount big.Rat
	for _, payment := range payments {
		if payment.State == domain.StateSettled {
			settledCount++
			totalAmount.Add(&totalAmount, payment.Amount)
		}
	}

	return fmt.Sprintf("SETTLEMENT %s recorded. Settled payments: %d", batchID, settledCount), nil
}

// handleStatus handles the STATUS command.
func (p *Processor) handleStatus(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("STATUS requires payment_id")
	}

	paymentID := args[0]
	payment, err := p.store.Get(paymentID)
	if err != nil {
		return "", fmt.Errorf("payment %s not found", paymentID)
	}

	return fmt.Sprintf("Payment %s: state=%s amount=%s currency=%s merchant=%s",
		payment.ID, payment.State, payment.FormatAmount(), payment.Currency, payment.MerchantID), nil
}

// handleList handles the LIST command.
func (p *Processor) handleList() (string, error) {
	payments, err := p.store.List()
	if err != nil {
		return "", fmt.Errorf("failed to list payments: %v", err)
	}

	if len(payments) == 0 {
		return "No payments found", nil
	}

	var sb strings.Builder
	sb.WriteString("Payments:\n")
	for _, payment := range payments {
		sb.WriteString(fmt.Sprintf("  %s: state=%s amount=%s %s merchant=%s\n",
			payment.ID, payment.State, payment.FormatAmount(), payment.Currency, payment.MerchantID))
	}

	return strings.TrimSuffix(sb.String(), "\n"), nil
}

// handleAudit handles the AUDIT command.
// AUDIT must have ZERO side effects - it only acknowledges receipt.
func (p *Processor) handleAudit(args []string) (string, error) {
	if len(args) < 1 {
		return "", fmt.Errorf("AUDIT requires payment_id")
	}

	paymentID := args[0]
	// Verify payment exists but do NOT mutate anything
	_, err := p.store.Get(paymentID)
	if err != nil {
		return "", fmt.Errorf("payment %s not found", paymentID)
	}

	return "AUDIT RECEIVED", nil
}
