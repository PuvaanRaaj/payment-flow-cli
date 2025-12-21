package app

import (
	"bytes"
	"strings"
	"testing"

	"payment-sim/internal/service"
	"payment-sim/internal/store"
)

// Integration tests that verify the full CLI flow: input → parse → execute → output

func TestIntegration_CommentHandling(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldContain []string
		shouldError   bool
	}{
		{
			name:          "Valid comment after 4th token",
			input:         "CREATE P001 100.00 USD M001 # this is valid\nEXIT",
			shouldContain: []string{"created"},
			shouldError:   false,
		},
		{
			name:        "Invalid comment at position 2",
			input:       "LIST # invalid\nEXIT",
			shouldError: true,
		},
		{
			name:        "Invalid comment at position 3",
			input:       "AUTHORIZE P001 # invalid\nEXIT",
			shouldError: true,
		},
		{
			name:          "Valid VOID with reason and comment",
			input:         "CREATE P001 100.00 USD M001\nAUTHORIZE P001\nVOID P001 FRAUD # suspicious\nEXIT",
			shouldContain: []string{"created", "authorized", "voided"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := strings.NewReader(tt.input)
			var output bytes.Buffer

			memStore := store.NewMemoryStore()
			processor := service.NewProcessor(memStore, nil)
			runner := NewRunner(processor, input, &output)

			err := runner.Run()
			if err != nil {
				t.Fatalf("Run() error = %v", err)
			}

			result := output.String()

			if tt.shouldError {
				if !strings.Contains(result, "ERROR") {
					t.Errorf("Expected ERROR in output, got: %v", result)
				}
			} else {
				for _, expected := range tt.shouldContain {
					if !strings.Contains(result, expected) {
						t.Errorf("Expected '%s' in output, got: %v", expected, result)
					}
				}
			}
		})
	}
}

func TestIntegration_CreateIdempotency(t *testing.T) {
	input := strings.NewReader(`CREATE P001 100.00 USD M001
CREATE P001 100.00 USD M001
AUTHORIZE P001
CREATE P001 100.00 USD M001
EXIT`)

	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()

	// First CREATE should succeed
	if !strings.Contains(result, "created") {
		t.Error("Expected 'created' in output")
	}

	// Second CREATE should be idempotent
	if !strings.Contains(result, "idempotent") {
		t.Error("Expected 'idempotent' in output")
	}

	// Third CREATE (after AUTHORIZE) should fail
	if !strings.Contains(result, "already exists in state") {
		t.Error("Expected 'already exists in state' error after authorization")
	}
}

func TestIntegration_FullPaymentFlow(t *testing.T) {
	input := strings.NewReader(`CREATE P001 100.00 USD M001
AUTHORIZE P001
CAPTURE P001
SETTLE P001
STATUS P001
EXIT`)

	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()

	expectedTerms := []string{"created", "authorized", "captured", "settled", "state=SETTLED"}
	for _, term := range expectedTerms {
		if !strings.Contains(result, term) {
			t.Errorf("Expected '%s' in output: %v", term, result)
		}
	}
}

func TestIntegration_ErrorRecovery(t *testing.T) {
	input := strings.NewReader(`CREATE P001 100.00 USD M001
CAPTURE P001
AUTHORIZE P001
CAPTURE P001
STATUS P001
EXIT`)

	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()

	// Should have errors but continue processing
	errorCount := strings.Count(result, "ERROR")
	if errorCount < 1 {
		t.Error("Expected at least one ERROR (invalid CAPTURE before AUTHORIZE)")
	}

	// Should still create and authorize successfully
	if !strings.Contains(result, "created") {
		t.Error("Expected 'created' despite errors")
	}
	if !strings.Contains(result, "authorized") {
		t.Error("Expected 'authorized' despite earlier error")
	}
}

func TestIntegration_ListSortedOutput(t *testing.T) {
	input := strings.NewReader(`CREATE P003 30.00 USD M001
CREATE P001 10.00 USD M001
CREATE P002 20.00 USD M001
LIST
EXIT`)

	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()

	// Extract only the LIST output (after "Payments:")
	listStart := strings.Index(result, "Payments:")
	if listStart < 0 {
		t.Fatalf("No 'Payments:' found in output: %v", result)
	}
	listOutput := result[listStart:]

	// Check P001 appears before P002 which appears before P003 in LIST output
	p1Idx := strings.Index(listOutput, "P001")
	p2Idx := strings.Index(listOutput, "P002")
	p3Idx := strings.Index(listOutput, "P003")

	if p1Idx < 0 || p2Idx < 0 || p3Idx < 0 {
		t.Fatalf("LIST output missing payments: %v", listOutput)
	}

	if !(p1Idx < p2Idx && p2Idx < p3Idx) {
		t.Errorf("LIST not sorted: P001@%d, P002@%d, P003@%d in: %v", p1Idx, p2Idx, p3Idx, listOutput)
	}
}

func TestIntegration_AuditNoSideEffects(t *testing.T) {
	input := strings.NewReader(`CREATE P001 100.00 USD M001
AUTHORIZE P001
AUDIT P001
STATUS P001
EXIT`)

	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()

	// AUDIT should be received
	if !strings.Contains(result, "AUDIT RECEIVED") {
		t.Error("Expected 'AUDIT RECEIVED' in output")
	}

	// State should still be AUTHORIZED (not changed by AUDIT)
	if !strings.Contains(result, "state=AUTHORIZED") {
		t.Error("Expected state=AUTHORIZED (AUDIT should not change state)")
	}
}
