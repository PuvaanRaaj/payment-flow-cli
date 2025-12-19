// Package main is the entry point for the payment-sim CLI.
package main

import (
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"payment-sim/internal/app"
	"payment-sim/internal/service"
	"payment-sim/internal/store"
)

func main() {
	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutdown requested, exiting...")
		os.Exit(0)
	}()

	// Parse PRE_SETTLEMENT_THRESHOLD from environment
	var threshold *big.Rat
	if thresholdStr := os.Getenv("PRE_SETTLEMENT_THRESHOLD"); thresholdStr != "" && thresholdStr != "0" {
		threshold = new(big.Rat)
		if _, ok := threshold.SetString(thresholdStr); !ok {
			fmt.Fprintf(os.Stderr, "ERROR invalid PRE_SETTLEMENT_THRESHOLD: %s\n", thresholdStr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "PRE_SETTLEMENT_REVIEW enabled for amounts >= %s\n", thresholdStr)
	}

	// Determine input source
	var input io.Reader
	if len(os.Args) > 1 {
		// File input mode
		filename := os.Args[1]
		file, err := os.Open(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR cannot open file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()
		input = file
	} else {
		// Interactive (stdin) mode
		input = os.Stdin
	}

	// Initialize components
	memStore := store.NewMemoryStore()
	processor := service.NewProcessor(memStore, threshold)
	runner := app.NewRunner(processor, input, os.Stdout)

	// Run the main loop
	if err := runner.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR %v\n", err)
		os.Exit(1)
	}
}
