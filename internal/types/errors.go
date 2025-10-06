package types

import (
	"fmt"
)

// TypeErrorCode represents specific type-related error codes
type TypeErrorCode string

const (
	ErrorCodeInvalidType         TypeErrorCode = "INVALID_TYPE"
	ErrorCodeTypeNotFound        TypeErrorCode = "TYPE_NOT_FOUND"
	ErrorCodeIncompatibleTypes   TypeErrorCode = "INCOMPATIBLE_TYPES"
	ErrorCodeArraySpecError      TypeErrorCode = "ARRAY_SPEC_ERROR"
	ErrorCodeSizeExtractionError TypeErrorCode = "SIZE_EXTRACTION_ERROR"
	ErrorCodeEnumValidationError TypeErrorCode = "ENUM_VALIDATION_ERROR"
	ErrorCodeCompositeTypeError  TypeErrorCode = "COMPOSITE_TYPE_ERROR"
	ErrorCodeAnalysisError       TypeErrorCode = "ANALYSIS_ERROR"
	// Enhanced Error Recovery codes
	ErrorCodeConsolidationFailed TypeErrorCode = "CONSOLIDATION_FAILED"
	ErrorCodeSQLGenerationFailed TypeErrorCode = "SQL_GENERATION_FAILED"
	ErrorCodeValidationFailed    TypeErrorCode = "VALIDATION_FAILED"
	ErrorCodeRecoveryAttempted   TypeErrorCode = "RECOVERY_ATTEMPTED"
	ErrorCodePartialSuccess      TypeErrorCode = "PARTIAL_SUCCESS"
)

// TypeError represents a structured type-related error
type TypeError struct {
	Code       TypeErrorCode          `json:"code"`
	Message    string                 `json:"message"`
	TypeName   string                 `json:"type_name,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Suggestion string                 `json:"suggestion,omitempty"`
	InnerError error                  `json:"-"`
}

// Error implements the error interface
func (e *TypeError) Error() string {
	if e.TypeName != "" {
		return fmt.Sprintf("[TYPE:%s] %s (type: %s)", e.Code, e.Message, e.TypeName)
	}
	return fmt.Sprintf("[TYPE:%s] %s", e.Code, e.Message)
}

// Unwrap returns the inner error for error unwrapping
func (e *TypeError) Unwrap() error {
	return e.InnerError
}

// NewTypeError creates a new TypeError
func NewTypeError(code TypeErrorCode, message string, typeName string) *TypeError {
	return &TypeError{
		Code:     code,
		Message:  message,
		TypeName: typeName,
		Context:  make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *TypeError) WithContext(key string, value interface{}) *TypeError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion to the error
func (e *TypeError) WithSuggestion(suggestion string) *TypeError {
	e.Suggestion = suggestion
	return e
}

// WithInnerError wraps an inner error
func (e *TypeError) WithInnerError(err error) *TypeError {
	e.InnerError = err
	return e
}

// ConsolidationErrorRecovery represents error recovery strategies for consolidation failures
type ConsolidationErrorRecovery struct {
	OriginalError error
	RecoveryType  RecoveryType
	FallbackSQL   string
	Warnings      []string
	Success       bool
}

// RecoveryType defines different error recovery strategies
type RecoveryType string

const (
	RecoveryTypeSkipObject     RecoveryType = "SKIP_OBJECT"
	RecoveryTypeUseOriginal    RecoveryType = "USE_ORIGINAL"
	RecoveryTypeSimplifySQL    RecoveryType = "SIMPLIFY_SQL"
	RecoveryTypePartialMerge   RecoveryType = "PARTIAL_MERGE"
	RecoveryTypeForceConvert   RecoveryType = "FORCE_CONVERT"
)

// ValidationResult represents the result of SQL validation
type ValidationResult struct {
	Valid        bool
	Errors       []*TypeError
	Warnings     []string
	Suggestions  []string
	RecoveryHint *ConsolidationErrorRecovery
}

// NewValidationResult creates a new validation result
func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		Valid:       true,
		Errors:      []*TypeError{},
		Warnings:    []string{},
		Suggestions: []string{},
	}
}

// AddError adds an error to the validation result
func (vr *ValidationResult) AddError(err *TypeError) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, err)
}

// AddWarning adds a warning to the validation result
func (vr *ValidationResult) AddWarning(warning string) {
	vr.Warnings = append(vr.Warnings, warning)
}

// AddSuggestion adds a suggestion to the validation result
func (vr *ValidationResult) AddSuggestion(suggestion string) {
	vr.Suggestions = append(vr.Suggestions, suggestion)
}

// SetRecoveryHint sets a recovery hint for the validation result
func (vr *ValidationResult) SetRecoveryHint(recovery *ConsolidationErrorRecovery) {
	vr.RecoveryHint = recovery
}

// ErrorWithRecovery represents an error with built-in recovery strategies
type ErrorWithRecovery struct {
	OriginalError *TypeError
	RecoveryFunc  func() (*ConsolidationErrorRecovery, error)
	Context       string
}

// Error implements the error interface
func (ewr *ErrorWithRecovery) Error() string {
	return fmt.Sprintf("Error in %s: %v", ewr.Context, ewr.OriginalError)
}

// AttemptRecovery attempts to recover from the error
func (ewr *ErrorWithRecovery) AttemptRecovery() (*ConsolidationErrorRecovery, error) {
	if ewr.RecoveryFunc == nil {
		return nil, fmt.Errorf("no recovery function available")
	}
	return ewr.RecoveryFunc()
}

// NewErrorWithRecovery creates a new error with recovery capabilities
func NewErrorWithRecovery(originalError *TypeError, context string, recoveryFunc func() (*ConsolidationErrorRecovery, error)) *ErrorWithRecovery {
	return &ErrorWithRecovery{
		OriginalError: originalError,
		Context:       context,
		RecoveryFunc:  recoveryFunc,
	}
}
