# Payment Processing CLI Simulator

A production-quality Go CLI application that simulates a payment processing pipeline with state machine transitions, idempotency handling, and configurable pre-settlement review.

## Features

- **State Machine**: Payments flow through well-defined states (INITIATED → AUTHORIZED → CAPTURED → SETTLED)
- **Idempotency**: CREATE and SETTLE commands are idempotent with identical inputs
- **Flexible Input**: Read commands from stdin (interactive) or file (batch mode)
- **Robust Parsing**: Inline comments supported after required arguments
- **Configurable**: PRE_SETTLEMENT_REVIEW threshold via environment variable
- **Production Ready**: No panics, graceful error handling, deterministic output

## Installation

```bash
# Clone the repository
cd payment-flow-cli

# Build the binary
go build -o payment-sim ./cmd/payment-sim

# Or run directly
go run ./cmd/payment-sim
```

## Usage

### Interactive Mode (stdin)

```bash
./payment-sim
```

Then type commands interactively:

```
CREATE P001 100.00 USD M001
AUTHORIZE P001
STATUS P001
EXIT
```

### File Input Mode

```bash
./payment-sim input.txt
```

Or using stdin redirection:

```bash
./payment-sim < input.txt
```

### Docker

```bash
# Build the image
docker build -t payment-sim .

# Run with file input
docker run -i payment-sim < input.txt

# Run interactively
docker run -it payment-sim
```

## Commands

| Command    | Syntax                                                  | Description                                |
| ---------- | ------------------------------------------------------- | ------------------------------------------ |
| CREATE     | `CREATE <payment_id> <amount> <currency> <merchant_id>` | Create a new payment                       |
| AUTHORIZE  | `AUTHORIZE <payment_id>`                                | Authorize an initiated payment             |
| CAPTURE    | `CAPTURE <payment_id>`                                  | Capture an authorized payment              |
| VOID       | `VOID <payment_id> [reason_code]`                       | Void an initiated/authorized payment       |
| REFUND     | `REFUND <payment_id> [amount]`                          | Refund a captured payment                  |
| SETTLE     | `SETTLE <payment_id>`                                   | Settle a captured payment                  |
| SETTLEMENT | `SETTLEMENT <batch_id>`                                 | Record a settlement batch (reporting only) |
| STATUS     | `STATUS <payment_id>`                                   | Show payment details                       |
| LIST       | `LIST`                                                  | List all payments (sorted by ID)           |
| AUDIT      | `AUDIT <payment_id>`                                    | Audit request (no side effects)            |
| EXIT       | `EXIT`                                                  | Exit the application                       |

## State Machine

```
┌──────────┐
│ INITIATED│
└────┬─────┘
     │
     ├─────────────────────────────┐
     │                             │
     ▼                             ▼
┌──────────┐                 ┌──────────┐
│AUTHORIZED│                 │  VOIDED  │
└────┬─────┘                 └──────────┘
     │
     ├─────────────────────────────┐
     │                             │
     ▼ (if threshold met)          ▼
┌───────────────────┐        ┌──────────┐
│PRE_SETTLEMENT_    │        │  VOIDED  │
│    REVIEW         │        └──────────┘
└────────┬──────────┘
         │
         ▼
    ┌──────────┐
    │ CAPTURED │
    └────┬─────┘
         │
         ├─────────────────────────────┐
         │                             │
         ▼                             ▼
    ┌──────────┐                 ┌──────────┐
    │ SETTLED  │                 │ REFUNDED │
    └──────────┘                 └──────────┘
```

## Configuration

### PRE_SETTLEMENT_THRESHOLD

Enable automatic PRE_SETTLEMENT_REVIEW for high-value payments:

```bash
# Enable review for payments >= 1000
export PRE_SETTLEMENT_THRESHOLD=1000

# Run the CLI
./payment-sim
```

When enabled, payments with amounts greater than or equal to the threshold will automatically move to PRE_SETTLEMENT_REVIEW after authorization, requiring explicit CAPTURE to proceed.

Set to `0` or leave unset to disable this feature (default).

## Parsing Rules

- Lines may contain inline comments starting with `#`
- `#` is treated as a comment delimiter **ONLY** if it appears after all required arguments
- A line starting with `#` is malformed input, NOT a comment

### Examples

```
CREATE P1001 10.00 MYR M01 # test     ✓ Valid (comment after 4 required args)
AUTHORIZE P1001 # retry               ✓ Valid (comment after 1 required arg)
# CREATE P1002 11.00 MYR M01          ✗ Malformed (# at start is not a comment)
CREATE # P1003 10.00 MYR M01          ✗ Malformed (# before required args met)
```

## Idempotency

### CREATE

- Repeated CREATE with the same `payment_id` and identical `amount`, `currency`, `merchant_id` → idempotent (no error)
- Repeated CREATE with same `payment_id` but different attributes → existing payment marked as FAILED, new CREATE rejected

### SETTLE

- Calling SETTLE on an already SETTLED payment is idempotent (no error, no state change)

## Error Handling

- Invalid input → `ERROR <message>`, continues processing
- Invalid state transition → `ERROR <message>`, state not mutated
- Unknown command → `ERROR <message>`, continues processing
- The application never panics or prints stack traces

## Testing

Run all tests:

```bash
go test ./... -v
```

Run tests with coverage:

```bash
go test ./... -cover
```

## Project Structure

```
payment-sim/
├── go.mod
├── cmd/
│   └── payment-sim/
│       └── main.go              # CLI entrypoint
├── internal/
│   ├── app/
│   │   └── runner.go            # Main loop: read → parse → execute → output
│   ├── parser/
│   │   ├── parser.go            # Line parsing + comment rules
│   │   └── parser_test.go
│   ├── domain/
│   │   ├── payment.go           # Payment struct + states
│   │   ├── transitions.go       # State transition validation
│   │   ├── errors.go            # Domain-level errors
│   │   └── domain_test.go
│   ├── service/
│   │   ├── processor.go         # Command handlers
│   │   └── processor_test.go
│   └── store/
│       ├── memory.go            # In-memory repository
│       └── memory_test.go
├── Dockerfile
├── sample_input.txt
└── README.md
```

## Sample Session

```
$ ./payment-sim
CREATE P001 100.00 USD M001
Payment P001 created: 100.0 USD
AUTHORIZE P001
Payment P001 authorized
CAPTURE P001
Payment P001 captured
STATUS P001
Payment P001: state=CAPTURED amount=100.0 currency=USD merchant=M001
SETTLE P001
Payment P001 settled
LIST
Payments:
  P001: state=SETTLED amount=100.0 USD merchant=M001
EXIT
```

## License

MIT
