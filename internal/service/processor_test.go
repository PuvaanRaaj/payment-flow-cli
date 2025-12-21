package service

import (
	"math/big"
	"strings"
	"testing"

	"payment-sim/internal/parser"
	"payment-sim/internal/store"
)

func newTestProcessor() *Processor {
	return NewProcessor(store.NewMemoryStore(), nil)
}

func newTestProcessorWithThreshold(threshold string) *Processor {
	t := new(big.Rat)
	t.SetString(threshold)
	return NewProcessor(store.NewMemoryStore(), t)
}

func parseCmd(t *testing.T, line string) *parser.Command {
	t.Helper()
	cmd, err := parser.Parse(line)
	if err != nil {
		t.Fatalf("Failed to parse command %q: %v", line, err)
	}
	return cmd
}

// Happy Path Tests

func TestHappyPath_CreateAuthorizeCaptureSettle(t *testing.T) {
	p := newTestProcessor()

	// CREATE
	result, err := p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	if err != nil {
		t.Fatalf("CREATE failed: %v", err)
	}
	if !strings.Contains(result, "created") {
		t.Errorf("CREATE result = %v, want 'created'", result)
	}

	// AUTHORIZE
	result, err = p.Execute(parseCmd(t, "AUTHORIZE P001"))
	if err != nil {
		t.Fatalf("AUTHORIZE failed: %v", err)
	}
	if !strings.Contains(result, "authorized") {
		t.Errorf("AUTHORIZE result = %v, want 'authorized'", result)
	}

	// CAPTURE
	result, err = p.Execute(parseCmd(t, "CAPTURE P001"))
	if err != nil {
		t.Fatalf("CAPTURE failed: %v", err)
	}
	if !strings.Contains(result, "captured") {
		t.Errorf("CAPTURE result = %v, want 'captured'", result)
	}

	// SETTLE
	result, err = p.Execute(parseCmd(t, "SETTLE P001"))
	if err != nil {
		t.Fatalf("SETTLE failed: %v", err)
	}
	if !strings.Contains(result, "settled") {
		t.Errorf("SETTLE result = %v, want 'settled'", result)
	}

	// STATUS
	result, err = p.Execute(parseCmd(t, "STATUS P001"))
	if err != nil {
		t.Fatalf("STATUS failed: %v", err)
	}
	if !strings.Contains(result, "SETTLED") {
		t.Errorf("STATUS result = %v, want state=SETTLED", result)
	}
}

func TestHappyPath_CreateAuthorizeVoid(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 50.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))

	result, err := p.Execute(parseCmd(t, "VOID P001 CUSTOMER_REQUEST"))
	if err != nil {
		t.Fatalf("VOID failed: %v", err)
	}
	if !strings.Contains(result, "voided") {
		t.Errorf("VOID result = %v, want 'voided'", result)
	}

	// Verify status
	result, _ = p.Execute(parseCmd(t, "STATUS P001"))
	if !strings.Contains(result, "VOIDED") {
		t.Errorf("STATUS result = %v, want state=VOIDED", result)
	}
}

func TestHappyPath_CreateAuthorizeCaptureRefund(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 75.00 EUR M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))
	p.Execute(parseCmd(t, "CAPTURE P001"))

	result, err := p.Execute(parseCmd(t, "REFUND P001"))
	if err != nil {
		t.Fatalf("REFUND failed: %v", err)
	}
	if !strings.Contains(result, "refunded") {
		t.Errorf("REFUND result = %v, want 'refunded'", result)
	}

	// Verify status
	result, _ = p.Execute(parseCmd(t, "STATUS P001"))
	if !strings.Contains(result, "REFUNDED") {
		t.Errorf("STATUS result = %v, want state=REFUNDED", result)
	}
}

// Invalid Transition Tests

func TestInvalidTransition_RefundBeforeCapture(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))

	_, err := p.Execute(parseCmd(t, "REFUND P001"))
	if err == nil {
		t.Error("REFUND before CAPTURE should fail")
	}
}

func TestInvalidTransition_CaptureBeforeAuthorize(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))

	_, err := p.Execute(parseCmd(t, "CAPTURE P001"))
	if err == nil {
		t.Error("CAPTURE before AUTHORIZE should fail")
	}
}

func TestInvalidTransition_VoidAfterCapture(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))
	p.Execute(parseCmd(t, "CAPTURE P001"))

	_, err := p.Execute(parseCmd(t, "VOID P001"))
	if err == nil {
		t.Error("VOID after CAPTURE should fail")
	}
}

func TestInvalidTransition_SettleBeforeCapture(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))

	_, err := p.Execute(parseCmd(t, "SETTLE P001"))
	if err == nil {
		t.Error("SETTLE before CAPTURE should fail")
	}
}

// Idempotency Tests

func TestIdempotency_CreateWithIdenticalAttributes(t *testing.T) {
	p := newTestProcessor()

	// First CREATE
	_, err := p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	if err != nil {
		t.Fatalf("First CREATE failed: %v", err)
	}

	// Second CREATE with same attributes - should be idempotent
	result, err := p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	if err != nil {
		t.Errorf("Idempotent CREATE failed: %v", err)
	}
	if !strings.Contains(result, "idempotent") {
		t.Errorf("Idempotent CREATE result = %v, want 'idempotent'", result)
	}
}

func TestIdempotency_CreateWithDifferentAttributes(t *testing.T) {
	p := newTestProcessor()

	// First CREATE
	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))

	// Second CREATE with different amount - should fail and mark original as FAILED
	_, err := p.Execute(parseCmd(t, "CREATE P001 200.00 USD M001"))
	if err == nil {
		t.Error("CREATE with conflict should fail")
	}

	// Verify original is now FAILED
	result, _ := p.Execute(parseCmd(t, "STATUS P001"))
	if !strings.Contains(result, "FAILED") {
		t.Errorf("STATUS result = %v, want state=FAILED", result)
	}
}

func TestIdempotency_SettleOnSettledPayment(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))
	p.Execute(parseCmd(t, "CAPTURE P001"))
	p.Execute(parseCmd(t, "SETTLE P001"))

	// Second SETTLE - should be idempotent
	result, err := p.Execute(parseCmd(t, "SETTLE P001"))
	if err != nil {
		t.Errorf("Idempotent SETTLE failed: %v", err)
	}
	if !strings.Contains(result, "idempotent") {
		t.Errorf("Idempotent SETTLE result = %v, want 'idempotent'", result)
	}
}

// AUDIT Tests

func TestAudit_NoStateChange(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))

	// Get state before AUDIT
	beforeResult, _ := p.Execute(parseCmd(t, "STATUS P001"))

	// AUDIT
	result, err := p.Execute(parseCmd(t, "AUDIT P001"))
	if err != nil {
		t.Fatalf("AUDIT failed: %v", err)
	}
	if result != "AUDIT RECEIVED" {
		t.Errorf("AUDIT result = %v, want 'AUDIT RECEIVED'", result)
	}

	// Verify state hasn't changed
	afterResult, _ := p.Execute(parseCmd(t, "STATUS P001"))
	if beforeResult != afterResult {
		t.Errorf("AUDIT changed state: before=%v, after=%v", beforeResult, afterResult)
	}
}

// PRE_SETTLEMENT_REVIEW Tests

func TestPreSettlementReview_ThresholdTriggered(t *testing.T) {
	p := newTestProcessorWithThreshold("1000")

	p.Execute(parseCmd(t, "CREATE P001 1500.00 USD M001"))
	result, _ := p.Execute(parseCmd(t, "AUTHORIZE P001"))

	if !strings.Contains(result, "PRE_SETTLEMENT_REVIEW") {
		t.Errorf("AUTHORIZE result = %v, want PRE_SETTLEMENT_REVIEW", result)
	}

	// Verify status
	status, _ := p.Execute(parseCmd(t, "STATUS P001"))
	if !strings.Contains(status, "PRE_SETTLEMENT_REVIEW") {
		t.Errorf("STATUS = %v, want state=PRE_SETTLEMENT_REVIEW", status)
	}
}

func TestPreSettlementReview_BelowThreshold(t *testing.T) {
	p := newTestProcessorWithThreshold("1000")

	p.Execute(parseCmd(t, "CREATE P001 500.00 USD M001"))
	result, _ := p.Execute(parseCmd(t, "AUTHORIZE P001"))

	if strings.Contains(result, "PRE_SETTLEMENT_REVIEW") {
		t.Errorf("Below threshold should not trigger PRE_SETTLEMENT_REVIEW: %v", result)
	}
}

func TestPreSettlementReview_CaptureFromReview(t *testing.T) {
	p := newTestProcessorWithThreshold("100")

	p.Execute(parseCmd(t, "CREATE P001 200.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))

	// Should be in PRE_SETTLEMENT_REVIEW, CAPTURE should work
	result, err := p.Execute(parseCmd(t, "CAPTURE P001"))
	if err != nil {
		t.Fatalf("CAPTURE from PRE_SETTLEMENT_REVIEW failed: %v", err)
	}
	if !strings.Contains(result, "captured") {
		t.Errorf("CAPTURE result = %v, want 'captured'", result)
	}
}

// LIST Tests

func TestList_SortedOrder(t *testing.T) {
	p := newTestProcessor()

	// Create in non-sorted order
	p.Execute(parseCmd(t, "CREATE P003 30.00 USD M001"))
	p.Execute(parseCmd(t, "CREATE P001 10.00 USD M001"))
	p.Execute(parseCmd(t, "CREATE P002 20.00 USD M001"))

	result, _ := p.Execute(parseCmd(t, "LIST"))

	// Check that P001 appears before P002 which appears before P003
	p1Idx := strings.Index(result, "P001")
	p2Idx := strings.Index(result, "P002")
	p3Idx := strings.Index(result, "P003")

	if p1Idx < 0 || p2Idx < 0 || p3Idx < 0 {
		t.Fatalf("LIST result missing payments: %v", result)
	}
	if !(p1Idx < p2Idx && p2Idx < p3Idx) {
		t.Errorf("LIST not sorted: P001@%d, P002@%d, P003@%d", p1Idx, p2Idx, p3Idx)
	}
}

func TestList_Empty(t *testing.T) {
	p := newTestProcessor()

	result, _ := p.Execute(parseCmd(t, "LIST"))
	if !strings.Contains(result, "No payments") {
		t.Errorf("Empty LIST result = %v, want 'No payments'", result)
	}
}

// SETTLEMENT Tests

func TestSettlement_RecordsBatchID(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))
	p.Execute(parseCmd(t, "CAPTURE P001"))
	p.Execute(parseCmd(t, "SETTLE P001"))

	result, err := p.Execute(parseCmd(t, "SETTLEMENT BATCH001"))
	if err != nil {
		t.Fatalf("SETTLEMENT failed: %v", err)
	}
	if !strings.Contains(result, "BATCH001") {
		t.Errorf("SETTLEMENT result = %v, want BATCH001", result)
	}
	if !strings.Contains(result, "Settled payments: 1") {
		t.Errorf("SETTLEMENT result = %v, want 'Settled payments: 1'", result)
	}
}

// Edge Case Tests

func TestPaymentNotFound(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "STATUS NONEXISTENT"))
	if err == nil {
		t.Error("STATUS for nonexistent payment should fail")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Error = %v, want 'not found'", err)
	}
}

func TestInvalidCurrency(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "CREATE P001 100.00 USDD M001"))
	if err == nil {
		t.Error("CREATE with invalid currency should fail")
	}
}

func TestInvalidAmount(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "CREATE P001 abc USD M001"))
	if err == nil {
		t.Error("CREATE with invalid amount should fail")
	}
}

func TestVoidFromInitiated(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))

	// VOID from INITIATED should work
	result, err := p.Execute(parseCmd(t, "VOID P001"))
	if err != nil {
		t.Fatalf("VOID from INITIATED failed: %v", err)
	}
	if !strings.Contains(result, "voided") {
		t.Errorf("VOID result = %v, want 'voided'", result)
	}
}

// Additional tests for 100% coverage

func TestRefundWithAmount(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))
	p.Execute(parseCmd(t, "CAPTURE P001"))

	result, err := p.Execute(parseCmd(t, "REFUND P001 50.00"))
	if err != nil {
		t.Fatalf("REFUND with amount failed: %v", err)
	}
	if !strings.Contains(result, "50.00") {
		t.Errorf("REFUND result = %v, want amount in output", result)
	}
}

func TestAuthorizeNotFound(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "AUTHORIZE NONEXISTENT"))
	if err == nil {
		t.Error("AUTHORIZE for nonexistent payment should fail")
	}
}

func TestCaptureNotFound(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "CAPTURE NONEXISTENT"))
	if err == nil {
		t.Error("CAPTURE for nonexistent payment should fail")
	}
}

func TestVoidNotFound(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "VOID NONEXISTENT"))
	if err == nil {
		t.Error("VOID for nonexistent payment should fail")
	}
}

func TestRefundNotFound(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "REFUND NONEXISTENT"))
	if err == nil {
		t.Error("REFUND for nonexistent payment should fail")
	}
}

func TestSettleNotFound(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "SETTLE NONEXISTENT"))
	if err == nil {
		t.Error("SETTLE for nonexistent payment should fail")
	}
}

func TestAuditNotFound(t *testing.T) {
	p := newTestProcessor()

	_, err := p.Execute(parseCmd(t, "AUDIT NONEXISTENT"))
	if err == nil {
		t.Error("AUDIT for nonexistent payment should fail")
	}
}

func TestExecuteExit(t *testing.T) {
	p := newTestProcessor()

	result, err := p.Execute(parseCmd(t, "EXIT"))
	if err != nil {
		t.Errorf("EXIT failed: %v", err)
	}
	if result != "" {
		t.Errorf("EXIT result = %v, want empty", result)
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	p := newTestProcessor()

	// Create a command manually with unknown name
	cmd := &parser.Command{Name: "UNKNOWN", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("Unknown command should fail")
	}
}

func TestSettlementNoSettledPayments(t *testing.T) {
	p := newTestProcessor()

	// Create but don't settle
	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))
	p.Execute(parseCmd(t, "CAPTURE P001"))

	result, err := p.Execute(parseCmd(t, "SETTLEMENT BATCH001"))
	if err != nil {
		t.Fatalf("SETTLEMENT failed: %v", err)
	}
	if !strings.Contains(result, "Settled payments: 0") {
		t.Errorf("SETTLEMENT result = %v, want 'Settled payments: 0'", result)
	}
}

// Tests for empty args validation

func TestCreateEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "CREATE", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("CREATE with empty args should fail")
	}
}

func TestAuthorizeEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "AUTHORIZE", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("AUTHORIZE with empty args should fail")
	}
}

func TestCaptureEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "CAPTURE", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("CAPTURE with empty args should fail")
	}
}

func TestVoidEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "VOID", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("VOID with empty args should fail")
	}
}

func TestRefundEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "REFUND", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("REFUND with empty args should fail")
	}
}

func TestSettleEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "SETTLE", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("SETTLE with empty args should fail")
	}
}

func TestSettlementEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "SETTLEMENT", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("SETTLEMENT with empty args should fail")
	}
}

func TestStatusEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "STATUS", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("STATUS with empty args should fail")
	}
}

func TestAuditEmptyArgs(t *testing.T) {
	p := newTestProcessor()
	cmd := &parser.Command{Name: "AUDIT", Args: []string{}}
	_, err := p.Execute(cmd)
	if err == nil {
		t.Error("AUDIT with empty args should fail")
	}
}

func TestAuthorizeInvalidTransition(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))

	// Try to authorize again - should fail
	_, err := p.Execute(parseCmd(t, "AUTHORIZE P001"))
	if err == nil {
		t.Error("AUTHORIZE twice should fail")
	}
}

func TestCreateAfterPaymentProgressed(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))

	// Try to CREATE again with same attrs - should fail because state is AUTHORIZED
	_, err := p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	if err == nil {
		t.Error("CREATE should fail when payment has progressed beyond INITIATED")
	}
	if !strings.Contains(err.Error(), "already exists in state") {
		t.Errorf("Expected 'already exists in state' error, got: %v", err)
	}
}

func TestCreateAfterSettle(t *testing.T) {
	p := newTestProcessor()

	p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	p.Execute(parseCmd(t, "AUTHORIZE P001"))
	p.Execute(parseCmd(t, "CAPTURE P001"))
	p.Execute(parseCmd(t, "SETTLE P001"))

	// Try to CREATE again - should fail because payment is SETTLED
	_, err := p.Execute(parseCmd(t, "CREATE P001 100.00 USD M001"))
	if err == nil {
		t.Error("CREATE should fail when payment is in SETTLED state")
	}
	if !strings.Contains(err.Error(), "SETTLED") {
		t.Errorf("Expected SETTLED in error, got: %v", err)
	}
}
