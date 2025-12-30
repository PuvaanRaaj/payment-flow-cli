# Payment Processing CLI - Interview Guide

This document explains each file in the project to help you answer interview questions.

---

## Project Overview

This is a **production-quality CLI** that simulates a payment processing pipeline with:

- State machine (INITIATED → AUTHORIZED → CAPTURED → SETTLED)
- 11 commands (CREATE, AUTHORIZE, CAPTURE, VOID, REFUND, SETTLE, etc.)
- Precise decimal handling with `math/big.Rat`
- Graceful error handling (never crashes)

---

## File-by-File Breakdown

### 📁 `cmd/payment-sim/main.go`

**Purpose:** CLI entrypoint - where the program starts

**What it does:**

1. Sets up graceful shutdown (Ctrl+C handling)
2. Reads `PRE_SETTLEMENT_THRESHOLD` env var for configurable review threshold
3. Determines input source (file argument vs stdin)
4. Wires up all components (store → processor → runner)
5. Runs the main loop

**Interview talking point:**

> "This follows the dependency injection pattern - I create all dependencies at the top level and pass them down, making the code testable and loosely coupled."

---

### 📁 `internal/app/runner.go`

**Purpose:** Main application loop (read → parse → execute → output)

**What it does:**

1. Reads lines from input using `bufio.Scanner`
2. Skips empty lines
3. Parses each line into a Command
4. Checks for EXIT to terminate
5. Executes the command via processor
6. Prints results or errors (prefixed with "ERROR")
7. **Never crashes** - catches all errors and continues

**Interview talking point:**

> "This is the orchestration layer. It doesn't know business logic - it just coordinates parsing and execution. Errors are caught and printed, not thrown, so the program continues processing."

---

### 📁 `internal/parser/parser.go`

**Purpose:** Tokenize and validate command syntax

**What it does:**

1. Defines required argument count per command (e.g., CREATE needs 4)
2. Splits input by whitespace
3. **Critical:** Only treats `#` as comment AFTER required args are consumed
4. Returns structured `Command{Name, Args}` or error

**Key code:**

```go
var commandArgCounts = map[string]int{
    "CREATE": 4,  // <id> <amount> <currency> <merchant>
    "AUTHORIZE": 1,
    "LIST": 0,
    // ...
}
```

**Interview talking point:**

> "The tricky part was the comment rule. `# comment` at the start is NOT a comment - it's malformed input. I had to track how many required args I'd collected before treating `#` as a delimiter."

---

### 📁 `internal/domain/payment.go`

**Purpose:** Core business entity + amount handling

**What it does:**

1. Defines `Payment` struct with ID, Amount, Currency, MerchantID, State
2. Uses `*big.Rat` for precise decimal math (no float64 rounding errors!)
3. Tracks state history for audit trail
4. `TransitionTo()` validates transitions before applying

**Key design:**

```go
type Payment struct {
    ID         string
    Amount     *big.Rat  // Precise decimals!
    State      string
    History    []HistoryEntry
}
```

**Interview talking point:**

> "I used `math/big.Rat` instead of float64 because financial calculations need exact precision. `10.30 - 10.00` should always equal `0.30`, not `0.29999999`."

---

### 📁 `internal/domain/transitions.go`

**Purpose:** State machine rules

**What it does:**

1. Declarative map of allowed transitions
2. `CanTransition(from, to)` returns bool
3. `ValidateTransition()` returns error if invalid

**Key code:**

```go
var AllowedTransitions = map[string][]string{
    StateInitiated: {StateAuthorized, StateVoided, StateFailed},
    StateAuthorized: {StateCaptured, StateVoided, StatePreSettlementReview},
    StateCaptured: {StateSettled, StateRefunded},
    StateSettled: {StateSettled}, // Idempotent
}
```

**Interview talking point:**

> "I made the state machine declarative rather than using if/else chains. This makes it easy to visualize and modify the rules, and ensures consistency."

---

### 📁 `internal/domain/errors.go`

**Purpose:** Custom error types for the domain

**What it does:**

1. `ErrPaymentNotFound` - sentinel error
2. `InvalidTransitionError` - includes from/to states
3. `CreateConflictError` - duplicate ID with different attributes

**Interview talking point:**

> "I created typed errors instead of just `errors.New()` so callers can type-assert and handle specific cases differently."

---

### 📁 `internal/service/processor.go`

**Purpose:** Command handlers - all the business logic

**What it does:**

1. `Execute(cmd)` switches on command name
2. Each handler (handleCreate, handleAuthorize, etc.) implements business rules
3. Handles idempotency:
   - CREATE: Same attributes = no-op, different = mark FAILED
   - SETTLE: Already settled = no-op
4. PRE_SETTLEMENT_REVIEW: If amount ≥ threshold, auto-transition after authorize

**Key handlers:**
| Handler | Key Logic |
|---------|-----------|
| `handleCreate` | Check for duplicate, validate currency (3 chars) |
| `handleAuthorize` | Transition + optional pre-settlement review |
| `handleVoid` | Only from INITIATED or AUTHORIZED |
| `handleRefund` | Only from CAPTURED |
| `handleSettle` | Idempotent if already settled |

**Interview talking point:**

> "The processor is pure business logic - no I/O. It takes a command and returns a result. This makes it fully testable without mocking stdin/stdout."

---

### 📁 `internal/store/memory.go`

**Purpose:** In-memory data storage

**What it does:**

1. Implements `Repository` interface
2. Thread-safe with `sync.RWMutex`
3. `List()` returns payments sorted by ID (deterministic output)
4. Also tracks batch IDs for SETTLEMENT command

**Key code:**

```go
type Repository interface {
    Save(payment *domain.Payment) error
    Get(id string) (*domain.Payment, error)
    List() ([]*domain.Payment, error)
}
```

**Interview talking point:**

> "I defined an interface so we could swap in a database later without changing the service layer. The in-memory implementation is just one possible backend."

---

### 📁 Test Files (`*_test.go`)

**Purpose:** Comprehensive test coverage (97-100%)

**Categories tested:**

1. **Happy paths:** CREATE → AUTHORIZE → CAPTURE → SETTLE
2. **Invalid transitions:** REFUND before CAPTURE, VOID after CAPTURE
3. **Idempotency:** Repeated CREATE, repeated SETTLE
4. **Parser edge cases:** `#` at beginning, `#` in wrong position
5. **AUDIT:** Verify no state changed

**Interview talking point:**

> "I have 97-100% coverage on all packages. I tested not just happy paths but also error cases and edge cases like the tricky comment parsing rules."

---

### 📁 `Dockerfile`

**Purpose:** Containerized deployment

**What it does:**

1. Multi-stage build (builder + runtime)
2. Uses Alpine for tiny image (~10MB)
3. Non-root user for security
4. Binary at `/usr/local/bin/payment-sim`

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│                   main.go                       │
│   (wire dependencies, handle CLI args)          │
└─────────────────────┬───────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────┐
│                 runner.go                        │
│   (read input, parse, execute, print output)     │
└───────┬─────────────────────────────┬───────────┘
        │                             │
┌───────▼───────┐             ┌───────▼───────┐
│  parser.go    │             │ processor.go  │
│ (tokenize,    │             │ (business     │
│  validate)    │             │  logic)       │
└───────────────┘             └───────┬───────┘
                                      │
                    ┌─────────────────┼─────────────────┐
                    │                 │                 │
            ┌───────▼───────┐ ┌───────▼───────┐ ┌───────▼───────┐
            │  payment.go   │ │ transitions.go│ │  memory.go    │
            │ (entity)      │ │ (state rules) │ │ (storage)     │
            └───────────────┘ └───────────────┘ └───────────────┘
```

---

## Key Design Principles to Mention

1. **Separation of concerns** - Parser doesn't know business rules; domain doesn't do I/O
2. **Testability** - Everything accepts interfaces, making mocking easy
3. **Graceful errors** - Never crashes; continues processing on errors
4. **Dependency injection** - Dependencies created at top level and passed down
5. **Declarative state machine** - Easy to understand and modify

---

## Quick Commands to Run

```bash
# Run tests
go test ./... -v

# Run with file input
go run ./cmd/payment-sim sample_input.txt

# Run interactively
go run ./cmd/payment-sim

# With PRE_SETTLEMENT_REVIEW enabled
PRE_SETTLEMENT_THRESHOLD=1000 go run ./cmd/payment-sim

# Docker
docker build -t payment-sim .
docker run -i payment-sim < sample_input.txt
```

---

## Test Coverage

| Package | Coverage |
| ------- | -------- |
| app     | 100% ✅  |
| domain  | 100% ✅  |
| store   | 100% ✅  |
| service | 97.2%    |
| parser  | 96.9%    |

Good luck with your interview! 🚀
