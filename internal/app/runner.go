// Package app provides the application runner for the payment processing CLI.
package app

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"payment-sim/internal/parser"
	"payment-sim/internal/service"
)

// Runner handles the main read-parse-execute-output loop.
type Runner struct {
	processor *service.Processor
	reader    *bufio.Scanner
	writer    io.Writer
}

// NewRunner creates a new application runner.
func NewRunner(processor *service.Processor, input io.Reader, output io.Writer) *Runner {
	return &Runner{
		processor: processor,
		reader:    bufio.NewScanner(input),
		writer:    output,
	}
}

// Run executes the main loop until EXIT is received or EOF is reached.
func (r *Runner) Run() error {
	for r.reader.Scan() {
		line := strings.TrimSpace(r.reader.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Parse the command
		cmd, err := parser.Parse(line)
		if err != nil {
			fmt.Fprintf(r.writer, "ERROR %s\n", err)
			continue
		}

		// Handle EXIT command
		if cmd.Name == "EXIT" {
			return nil
		}

		// Execute the command
		result, err := r.processor.Execute(cmd)
		if err != nil {
			fmt.Fprintf(r.writer, "ERROR %s\n", err)
			continue
		}

		// Print result if non-empty
		if result != "" {
			fmt.Fprintln(r.writer, result)
		}
	}

	// Check for scanner errors
	if err := r.reader.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}
