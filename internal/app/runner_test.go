package app

import (
	"bytes"
	"strings"
	"testing"

	"payment-sim/internal/service"
	"payment-sim/internal/store"
)

func TestRunner_BasicFlow(t *testing.T) {
	input := strings.NewReader(`CREATE P001 100.00 USD M001
AUTHORIZE P001
STATUS P001
EXIT
`)
	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "created") {
		t.Errorf("Output missing 'created': %v", result)
	}
	if !strings.Contains(result, "authorized") {
		t.Errorf("Output missing 'authorized': %v", result)
	}
	if !strings.Contains(result, "AUTHORIZED") {
		t.Errorf("Output missing status AUTHORIZED: %v", result)
	}
}

func TestRunner_EmptyLines(t *testing.T) {
	input := strings.NewReader(`

CREATE P001 100.00 USD M001

EXIT
`)
	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "created") {
		t.Errorf("Output missing 'created': %v", result)
	}
}

func TestRunner_ParseError(t *testing.T) {
	input := strings.NewReader(`INVALID_COMMAND
CREATE P001 100.00 USD M001
EXIT
`)
	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()
	// Should have error for invalid command but continue
	if !strings.Contains(result, "ERROR") {
		t.Errorf("Output missing 'ERROR': %v", result)
	}
	if !strings.Contains(result, "created") {
		t.Errorf("Output missing 'created' (should continue after error): %v", result)
	}
}

func TestRunner_ExecutionError(t *testing.T) {
	input := strings.NewReader(`CREATE P001 100.00 USD M001
CAPTURE P001
EXIT
`)
	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()
	// Should have error for invalid transition but continue
	if !strings.Contains(result, "ERROR") {
		t.Errorf("Output missing 'ERROR': %v", result)
	}
}

func TestRunner_EOF(t *testing.T) {
	// No EXIT command, just EOF
	input := strings.NewReader(`CREATE P001 100.00 USD M001
AUTHORIZE P001
`)
	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() should handle EOF gracefully: %v", err)
	}

	result := output.String()
	if !strings.Contains(result, "created") {
		t.Errorf("Output missing 'created': %v", result)
	}
}

// Test that result is not printed when empty (like EXIT command handled in runner)
func TestRunner_EmptyResult(t *testing.T) {
	// EXIT returns empty result - verify no extra output
	input := strings.NewReader(`EXIT
`)
	var output bytes.Buffer

	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, nil)
	runner := NewRunner(processor, input, &output)

	err := runner.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	result := output.String()
	if result != "" {
		t.Errorf("Output should be empty, got: %v", result)
	}
}
