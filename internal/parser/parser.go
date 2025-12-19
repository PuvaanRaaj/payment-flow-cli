// Package parser handles parsing of command lines for the payment processing CLI.
package parser

import (
	"fmt"
	"strings"
)

// Command represents a parsed command with its name and arguments.
type Command struct {
	Name string
	Args []string
}

// commandArgCounts defines the number of REQUIRED arguments for each command.
// Optional arguments are not counted here.
var commandArgCounts = map[string]int{
	"CREATE":     4, // <payment_id> <amount> <currency> <merchant_id>
	"AUTHORIZE":  1, // <payment_id>
	"CAPTURE":    1, // <payment_id>
	"VOID":       1, // <payment_id> [reason_code] - 1 required
	"REFUND":     1, // <payment_id> [amount] - 1 required
	"SETTLE":     1, // <payment_id>
	"SETTLEMENT": 1, // <batch_id>
	"STATUS":     1, // <payment_id>
	"LIST":       0,
	"AUDIT":      1, // <payment_id>
	"EXIT":       0,
}

// Parse parses a command line into a Command struct.
// It handles inline comments that appear ONLY after all required arguments.
// A '#' character is only treated as a comment delimiter if it appears after
// the required arguments for that command have been consumed.
func Parse(line string) (*Command, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("empty input")
	}

	// Tokenize by whitespace
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	// First token is the command name
	cmdName := tokens[0]

	// Check if command is known
	requiredArgs, known := commandArgCounts[cmdName]
	if !known {
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}

	// Extract arguments, handling comments properly
	args, err := extractArgs(tokens[1:], requiredArgs, cmdName)
	if err != nil {
		return nil, err
	}

	return &Command{
		Name: cmdName,
		Args: args,
	}, nil
}

// tokenize splits a line into tokens by whitespace.
func tokenize(line string) []string {
	return strings.Fields(line)
}

// extractArgs extracts arguments from tokens, handling the comment rules.
// Comments start with '#' but ONLY if they appear after all required args.
func extractArgs(tokens []string, requiredCount int, cmdName string) ([]string, error) {
	args := make([]string, 0, requiredCount)

	for _, token := range tokens {
		// Check if we've collected all required args
		if len(args) >= requiredCount {
			// Now '#' can start a comment - we can stop processing
			if strings.HasPrefix(token, "#") {
				break
			}
			// Otherwise, this is an optional argument (e.g., reason_code for VOID)
			args = append(args, token)
			continue
		}

		// Still collecting required args
		// '#' at the start of a token when we need more args is malformed
		if strings.HasPrefix(token, "#") {
			return nil, fmt.Errorf("malformed input: unexpected '#' in required argument position for %s", cmdName)
		}

		// Handle '#' appearing mid-token (e.g., "value#comment")
		// In required args, we treat this as the whole value including '#'
		args = append(args, token)
	}

	// Check if we got enough required args
	if len(args) < requiredCount {
		return nil, fmt.Errorf("insufficient arguments for %s: expected %d, got %d", cmdName, requiredCount, len(args))
	}

	return args, nil
}

// IsValidCommand checks if a command name is valid.
func IsValidCommand(name string) bool {
	_, ok := commandArgCounts[name]
	return ok
}

// GetRequiredArgCount returns the number of required arguments for a command.
func GetRequiredArgCount(name string) (int, bool) {
	count, ok := commandArgCounts[name]
	return count, ok
}
