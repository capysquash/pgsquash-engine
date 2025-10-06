package transformation

import (
	"fmt"
)

// TransformationErrorCode represents specific transformation-related error codes
type TransformationErrorCode string

const (
	ErrorCodeRollbackGenerationFailed   TransformationErrorCode = "ROLLBACK_GENERATION_FAILED"
	ErrorCodeRollbackAnalysisFailed     TransformationErrorCode = "ROLLBACK_ANALYSIS_FAILED"
	ErrorCodeRollbackSaveFailed         TransformationErrorCode = "ROLLBACK_SAVE_FAILED"
	ErrorCodeRollbackNotFound           TransformationErrorCode = "ROLLBACK_NOT_FOUND"
	ErrorCodeExecutionNotFound          TransformationErrorCode = "EXECUTION_NOT_FOUND"
	ErrorCodeDatabaseNotAccessible      TransformationErrorCode = "DATABASE_NOT_ACCESSIBLE"
	ErrorCodeManualInterventionRequired TransformationErrorCode = "MANUAL_INTERVENTION_REQUIRED"
	ErrorCodeInvalidSQL                 TransformationErrorCode = "INVALID_SQL"
	ErrorCodeBackupGenerationFailed     TransformationErrorCode = "BACKUP_GENERATION_FAILED"
	ErrorCodeBackupValidationFailed     TransformationErrorCode = "BACKUP_VALIDATION_FAILED"
	ErrorCodeTransformationFailed       TransformationErrorCode = "TRANSFORMATION_FAILED"
	ErrorCodeWorkingDirCreationFailed   TransformationErrorCode = "WORKING_DIR_CREATION_FAILED"
)

// TransformationError represents a structured transformation-related error
type TransformationError struct {
	Code       TransformationErrorCode `json:"code"`
	Message    string                  `json:"message"`
	Context    map[string]interface{}  `json:"context,omitempty"`
	Suggestion string                  `json:"suggestion,omitempty"`
	InnerError error                   `json:"-"`
}

// Error implements the error interface
func (e *TransformationError) Error() string {
	return fmt.Sprintf("[TRANSFORMATION:%s] %s", e.Code, e.Message)
}

// Unwrap returns the inner error for error unwrapping
func (e *TransformationError) Unwrap() error {
	return e.InnerError
}

// NewTransformationError creates a new TransformationError
func NewTransformationError(code TransformationErrorCode, message string) *TransformationError {
	return &TransformationError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *TransformationError) WithContext(key string, value interface{}) *TransformationError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion to the error
func (e *TransformationError) WithSuggestion(suggestion string) *TransformationError {
	e.Suggestion = suggestion
	return e
}

// WithInnerError wraps an inner error
func (e *TransformationError) WithInnerError(err error) *TransformationError {
	e.InnerError = err
	return e
}
