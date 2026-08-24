package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// ErrorCode is a stable, machine-readable error code. The API surface exposes
// these codes so public tests can assert them precisely.
type ErrorCode string

const (
	CodeStrainMismatch         ErrorCode = "strain_mismatch"
	CodeSubstrateMismatch      ErrorCode = "substrate_mismatch"
	CodeStaleSterilizer        ErrorCode = "stale_sterilizer_summary"
	CodeDuplicateBagPosition   ErrorCode = "duplicate_bag_position"
	CodeResourceWindowConflict ErrorCode = "resource_window_conflict"
	CodeGenerationConflict     ErrorCode = "generation_conflict"
	CodeRoleOverlap            ErrorCode = "role_overlap"
	CodeIdempotencyConflict    ErrorCode = "idempotency_conflict"
	CodeReadingOutOfRange      ErrorCode = "reading_out_of_range"
	CodeArithmeticOverflow     ErrorCode = "arithmetic_overflow"
	CodeDeviceFailure          ErrorCode = "device_failure"
	CodeRecheckInsufficient    ErrorCode = "recheck_coverage_insufficient"
	CodeFinalStateRejected     ErrorCode = "final_state_rejected"
)

// Sentinel errors for fixed-point arithmetic failures.
var (
	ErrOverflow       = errors.New("domain: arithmetic overflow")
	ErrDivisionByZero = errors.New("domain: division by zero")
)

// Error is a stable domain error carrying a code and deterministically sorted
// rejection reasons.
type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Reasons []string  `json:"reasons,omitempty"`
}

func (e *Error) Error() string {
	if len(e.Reasons) == 0 {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.Code, e.Message, strings.Join(e.Reasons, ", "))
}

// NewError builds an Error with deterministically sorted reasons.
func NewError(code ErrorCode, message string, reasons ...string) *Error {
	return &Error{Code: code, Message: message, Reasons: SortReasons(reasons)}
}

// SortReasons returns a deterministically sorted copy of reasons.
func SortReasons(reasons []string) []string {
	out := append([]string(nil), reasons...)
	sort.Strings(out)
	return out
}
