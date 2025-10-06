package parser

import (
	"context"
	"fmt"
	"strings"
)

// ErrorSeverity represents the severity level of parsing errors
type ErrorSeverity int

const (
	SeverityInfo ErrorSeverity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s ErrorSeverity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARNING"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ErrorCategory represents the category of parsing errors
type ErrorCategory string

const (
	CategorySyntax      ErrorCategory = "SYNTAX"
	CategorySemantic    ErrorCategory = "SEMANTIC"
	CategoryDependency  ErrorCategory = "DEPENDENCY"
	CategoryNaming      ErrorCategory = "NAMING"
	CategoryPermission  ErrorCategory = "PERMISSION"
	CategoryConstraint  ErrorCategory = "CONSTRAINT"
	CategoryDataType    ErrorCategory = "DATA_TYPE"
	CategoryFunction    ErrorCategory = "FUNCTION"
	CategoryIndex       ErrorCategory = "INDEX"
	CategoryPolicy      ErrorCategory = "POLICY"
	CategoryExtension   ErrorCategory = "EXTENSION"
	CategoryPerformance ErrorCategory = "PERFORMANCE"
)

// ParseError represents a structured parsing error with context
type ParseError struct {
	Message     string
	Severity    ErrorSeverity
	Category    ErrorCategory
	Context     *ParseContext
	Suggestion  string
	Code        string
	InnerError  error
	Statement   *Statement
	CanContinue bool
}

// ParseContext provides context information for errors
type ParseContext struct {
	Filename        string
	Line            int
	Column          int
	StatementText   string
	ObjectName      string
	ObjectType      ObjectType
	Schema          string
	MigrationSeq    int
	ProcessingPhase string
}

// Error implements the error interface
func (e *ParseError) Error() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("[%s:%s]", e.Severity, e.Category))

	if e.Context != nil {
		if e.Context.Filename != "" {
			parts = append(parts, fmt.Sprintf("file:%s", e.Context.Filename))
		}
		if e.Context.Line > 0 {
			parts = append(parts, fmt.Sprintf("line:%d", e.Context.Line))
		}
		if e.Context.ObjectName != "" {
			parts = append(parts, fmt.Sprintf("object:%s", e.Context.ObjectName))
		}
	}

	parts = append(parts, e.Message)

	if e.Suggestion != "" {
		parts = append(parts, fmt.Sprintf("suggestion: %s", e.Suggestion))
	}

	return strings.Join(parts, " ")
}

// Unwrap returns the inner error for error unwrapping
func (e *ParseError) Unwrap() error {
	return e.InnerError
}

// ErrorCollector collects and manages parsing errors
type ErrorCollector struct {
	errors   []*ParseError
	warnings []*ParseError
	context  context.Context
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(ctx context.Context) *ErrorCollector {
	return &ErrorCollector{
		errors:   make([]*ParseError, 0),
		warnings: make([]*ParseError, 0),
		context:  ctx,
	}
}

// AddError adds an error to the collector
func (ec *ErrorCollector) AddError(err *ParseError) {
	if err.Severity >= SeverityError {
		ec.errors = append(ec.errors, err)
	} else {
		ec.warnings = append(ec.warnings, err)
	}
}

// AddSyntaxError adds a syntax error
func (ec *ErrorCollector) AddSyntaxError(message string, ctx *ParseContext, innerErr error) {
	ec.AddError(&ParseError{
		Message:     message,
		Severity:    SeverityError,
		Category:    CategorySyntax,
		Context:     ctx,
		InnerError:  innerErr,
		CanContinue: false,
	})
}

// AddSemanticError adds a semantic error
func (ec *ErrorCollector) AddSemanticError(message, suggestion string, ctx *ParseContext) {
	ec.AddError(&ParseError{
		Message:     message,
		Severity:    SeverityError,
		Category:    CategorySemantic,
		Context:     ctx,
		Suggestion:  suggestion,
		CanContinue: true,
	})
}

// AddDependencyError adds a dependency error
func (ec *ErrorCollector) AddDependencyError(message string, ctx *ParseContext) {
	ec.AddError(&ParseError{
		Message:     message,
		Severity:    SeverityError,
		Category:    CategoryDependency,
		Context:     ctx,
		Suggestion:  "Check dependency order and ensure referenced objects exist",
		CanContinue: true,
	})
}

// AddNamingWarning adds a naming convention warning
func (ec *ErrorCollector) AddNamingWarning(message, suggestion string, ctx *ParseContext) {
	ec.AddError(&ParseError{
		Message:     message,
		Severity:    SeverityWarning,
		Category:    CategoryNaming,
		Context:     ctx,
		Suggestion:  suggestion,
		CanContinue: true,
	})
}

// AddPerformanceWarning adds a performance warning
func (ec *ErrorCollector) AddPerformanceWarning(message, suggestion string, ctx *ParseContext) {
	ec.AddError(&ParseError{
		Message:     message,
		Severity:    SeverityWarning,
		Category:    CategoryPerformance,
		Context:     ctx,
		Suggestion:  suggestion,
		CanContinue: true,
	})
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// HasWarnings returns true if there are any warnings
func (ec *ErrorCollector) HasWarnings() bool {
	return len(ec.warnings) > 0
}

// GetErrors returns all errors
func (ec *ErrorCollector) GetErrors() []*ParseError {
	return ec.errors
}

// GetWarnings returns all warnings
func (ec *ErrorCollector) GetWarnings() []*ParseError {
	return ec.warnings
}

// GetAllIssues returns both errors and warnings
func (ec *ErrorCollector) GetAllIssues() []*ParseError {
	var all []*ParseError
	all = append(all, ec.errors...)
	all = append(all, ec.warnings...)
	return all
}

// Clear removes all collected errors and warnings
func (ec *ErrorCollector) Clear() {
	ec.errors = ec.errors[:0]
	ec.warnings = ec.warnings[:0]
}

// Summary returns a summary of collected issues
func (ec *ErrorCollector) Summary() ErrorSummary {
	summary := ErrorSummary{
		TotalErrors:   len(ec.errors),
		TotalWarnings: len(ec.warnings),
		Categories:    make(map[ErrorCategory]int),
		Severities:    make(map[ErrorSeverity]int),
	}

	for _, err := range ec.GetAllIssues() {
		summary.Categories[err.Category]++
		summary.Severities[err.Severity]++
	}

	return summary
}

// ErrorSummary provides a summary of collected errors
type ErrorSummary struct {
	TotalErrors   int
	TotalWarnings int
	Categories    map[ErrorCategory]int
	Severities    map[ErrorSeverity]int
}

// ErrorReporter handles error reporting and formatting
type ErrorReporter struct {
	collector *ErrorCollector
	formatter *ErrorFormatter
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(collector *ErrorCollector) *ErrorReporter {
	return &ErrorReporter{
		collector: collector,
		formatter: NewErrorFormatter(),
	}
}

// ErrorFormatter formats errors for different outputs
type ErrorFormatter struct{}

// NewErrorFormatter creates a new error formatter
func NewErrorFormatter() *ErrorFormatter {
	return &ErrorFormatter{}
}

// FormatError formats a single error
func (ef *ErrorFormatter) FormatError(err *ParseError) string {
	return err.Error()
}

// FormatSummary formats an error summary
func (ef *ErrorFormatter) FormatSummary(summary ErrorSummary) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Total: %d errors, %d warnings",
		summary.TotalErrors, summary.TotalWarnings))

	if len(summary.Categories) > 0 {
		parts = append(parts, "Categories:")
		for category, count := range summary.Categories {
			parts = append(parts, fmt.Sprintf("  %s: %d", category, count))
		}
	}

	return strings.Join(parts, "\n")
}

// FormatErrorList formats a list of errors
func (ef *ErrorFormatter) FormatErrorList(errors []*ParseError) string {
	var parts []string

	for i, err := range errors {
		parts = append(parts, fmt.Sprintf("%d. %s", i+1, ef.FormatError(err)))
	}

	return strings.Join(parts, "\n")
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	collector *ErrorCollector
	reporter  *ErrorReporter
	context   context.Context
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(ctx context.Context) *ErrorHandler {
	collector := NewErrorCollector(ctx)
	reporter := NewErrorReporter(collector)

	return &ErrorHandler{
		collector: collector,
		reporter:  reporter,
		context:   ctx,
	}
}

// HandleParseError handles a parse error with appropriate logging
func (eh *ErrorHandler) HandleParseError(err error, ctx *ParseContext) *ParseError {
	parseErr := &ParseError{
		Message:     err.Error(),
		Severity:    SeverityError,
		Category:    CategorySyntax,
		Context:     ctx,
		InnerError:  err,
		CanContinue: false,
	}

	eh.collector.AddError(parseErr)
	return parseErr
}

// HandleValidationError handles validation errors
func (eh *ErrorHandler) HandleValidationError(message string, ctx *ParseContext) *ParseError {
	parseErr := &ParseError{
		Message:     message,
		Severity:    SeverityError,
		Category:    CategorySemantic,
		Context:     ctx,
		CanContinue: true,
	}

	eh.collector.AddError(parseErr)
	return parseErr
}

// CreateContext creates a parse context from available information
func (eh *ErrorHandler) CreateContext(filename string, line int, stmt *Statement) *ParseContext {
	ctx := &ParseContext{
		Filename: filename,
		Line:     line,
	}

	if stmt != nil {
		ctx.StatementText = stmt.SQL
		ctx.ObjectName = stmt.ObjectName
		ctx.ObjectType = stmt.ObjectType
		ctx.Schema = stmt.Schema
	}

	return ctx
}

// GetCollector returns the error collector
func (eh *ErrorHandler) GetCollector() *ErrorCollector {
	return eh.collector
}

// GetReporter returns the error reporter
func (eh *ErrorHandler) GetReporter() *ErrorReporter {
	return eh.reporter
}

// Recovery handles panic recovery during parsing
func (eh *ErrorHandler) Recovery(filename string, line int) {
	if r := recover(); r != nil {
		ctx := &ParseContext{
			Filename: filename,
			Line:     line,
		}

		parseErr := &ParseError{
			Message:     fmt.Sprintf("Panic during parsing: %v", r),
			Severity:    SeverityCritical,
			Category:    CategorySyntax,
			Context:     ctx,
			CanContinue: false,
		}

		eh.collector.AddError(parseErr)
	}
}

// ShouldContinue determines if processing should continue after an error
func (eh *ErrorHandler) ShouldContinue() bool {
	for _, err := range eh.collector.GetErrors() {
		if !err.CanContinue {
			return false
		}
	}
	return true
}

// LogSummary logs a summary of all collected errors and warnings
func (eh *ErrorHandler) LogSummary() {
	summary := eh.collector.Summary()
	if summary.TotalErrors > 0 || summary.TotalWarnings > 0 {
		fmt.Printf("Parse Summary: %s\n", eh.reporter.formatter.FormatSummary(summary))
	}
}
