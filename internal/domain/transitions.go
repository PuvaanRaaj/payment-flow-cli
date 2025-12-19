package domain

// AllowedTransitions defines the valid state transitions.
// The key is the current state, and the value is a slice of valid target states.
var AllowedTransitions = map[string][]string{
	StateInitiated: {
		StateAuthorized,
		StateVoided,
		StateFailed,
	},
	StateAuthorized: {
		StatePreSettlementReview,
		StateCaptured,
		StateVoided,
	},
	StatePreSettlementReview: {
		StateCaptured,
	},
	StateCaptured: {
		StateSettled,
		StateRefunded,
	},
	StateSettled: {
		StateSettled, // Idempotent
	},
	StateVoided:   {}, // Terminal state
	StateRefunded: {}, // Terminal state
	StateFailed:   {}, // Terminal state
}

// CanTransition checks if a transition from one state to another is allowed.
func CanTransition(from, to string) bool {
	allowed, exists := AllowedTransitions[from]
	if !exists {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ValidateTransition returns an error if the transition is not allowed.
func ValidateTransition(from, to string) error {
	if !CanTransition(from, to) {
		return NewInvalidTransitionError(from, to)
	}
	return nil
}
