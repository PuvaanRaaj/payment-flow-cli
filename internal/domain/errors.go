package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for the domain layer.
var (
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrDuplicatePayment = errors.New("payment already exists")
	ErrInvalidAmount    = errors.New("invalid amount")
)

// InvalidTransitionError represents an invalid state transition attempt.
type InvalidTransitionError struct {
	From string
	To   string
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid transition from %s to %s", e.From, e.To)
}

// NewInvalidTransitionError creates a new InvalidTransitionError.
func NewInvalidTransitionError(from, to string) *InvalidTransitionError {
	return &InvalidTransitionError{From: from, To: to}
}

// CreateConflictError represents a conflict when creating a payment with the same ID but different attributes.
type CreateConflictError struct {
	PaymentID string
}

func (e *CreateConflictError) Error() string {
	return fmt.Sprintf("create conflict for payment %s: existing payment marked as FAILED", e.PaymentID)
}

// NewCreateConflictError creates a new CreateConflictError.
func NewCreateConflictError(paymentID string) *CreateConflictError {
	return &CreateConflictError{PaymentID: paymentID}
}

// ParseError represents a parsing error.
type ParseError struct {
	Message string
}

func (e *ParseError) Error() string {
	return e.Message
}

// NewParseError creates a new ParseError.
func NewParseError(msg string) *ParseError {
	return &ParseError{Message: msg}
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}
