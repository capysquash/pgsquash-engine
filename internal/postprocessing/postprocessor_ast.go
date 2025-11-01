package postprocessing

import (
	"github.com/CAPYSQUASH/pgsquash-engine/internal/config"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/postprocessing/ast"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/utils"
)

// ProcessorAST orchestrates AST-based post-processing operations.
// This is an alternative to regex-based processing with better accuracy and maintainability.
type ProcessorAST struct {
	logger            *utils.Logger
	config            *config.Config
	funcNormalizer    *ast.FunctionNormalizer
	useASTForFunctions bool
	useASTForEnums    bool
}

// NewProcessorAST creates a new AST-based post-processing processor.
func NewProcessorAST(cfg *config.Config) *ProcessorAST {
	return &ProcessorAST{
		logger:            utils.GetDefaultLogger().WithPrefix("POSTPROCESS-AST"),
		config:            cfg,
		funcNormalizer:    ast.NewFunctionNormalizer(),
		useASTForFunctions: true, // Enable by default
		useASTForEnums:    true, // Enable by default
	}
}

// SetUseASTForFunctions enables or disables AST-based function processing.
func (p *ProcessorAST) SetUseASTForFunctions(use bool) {
	p.useASTForFunctions = use
}

// SetUseASTForEnums enables or disables AST-based ENUM processing.
func (p *ProcessorAST) SetUseASTForEnums(use bool) {
	p.useASTForEnums = use
}

// Apply runs all post-processing fixes using AST where possible, falling back to regex.
// This provides a migration path from regex to AST.
func (p *ProcessorAST) Apply(sql string, enumReplacements map[string]string) (string, error) {
	p.logger.Info("Starting AST-based post-processing pipeline...")

	// ================================================================
	// PHASE 1: Basic Syntax Fixes (Keep regex - operates on malformed SQL)
	// ================================================================
	p.logger.Info("Phase 1: Basic syntax fixes (regex)")
	sql = FixMalformedDropTriggers(sql)
	sql = FixExtensionOrder(sql)
	sql = RemoveOrphanedAlterStatements(sql)
	sql = FixMalformedFunctions(sql)

	// ================================================================
	// PHASE 2: Function Language Normalization (AST or regex)
	// ================================================================
	if p.useASTForFunctions {
		p.logger.Info("Phase 2: Function language normalization (AST)")
		normalizedSQL, err := p.funcNormalizer.NormalizeAll(sql)
		if err != nil {
			p.logger.Info("AST normalization failed, falling back to regex: %v", err)
			sql = p.applyRegexFunctionFixes(sql)
		} else {
			sql = normalizedSQL
		}
	} else {
		p.logger.Info("Phase 2: Function language normalization (regex)")
		sql = p.applyRegexFunctionFixes(sql)
	}

	// ================================================================
	// PHASE 3: Function Body Fixes (Keep regex for now)
	// ================================================================
	p.logger.Info("Phase 3: Function body fixes (regex)")
	sql = FixReturnNextWithOutParams(sql, nil)

	// ================================================================
	// PHASE 4: Final Cleanup
	// ================================================================
	p.logger.Info("Phase 4: Final cleanup")
	sql = FixMissingSemicolons(sql)

	// ENUM replacement (AST or regex)
	if len(enumReplacements) > 0 {
		if p.useASTForEnums {
			p.logger.Info("Phase 4: ENUM replacement (AST)")
			enumReplacer := ast.NewEnumReplacer(enumReplacements)
			replacedSQL, err := enumReplacer.ReplaceEnumReferences(sql)
			if err != nil {
				p.logger.Info("AST ENUM replacement failed, falling back to regex: %v", err)
				sql = fixEliminatedEnumReferences(sql, enumReplacements)
			} else {
				sql = replacedSQL
				p.logger.Info("AST replaced %d ENUM references", enumReplacer.GetReplacedCount())
			}
		} else {
			p.logger.Info("Phase 4: ENUM replacement (regex)")
			sql = fixEliminatedEnumReferences(sql, enumReplacements)
		}
	}

	p.logger.Info("AST-based post-processing pipeline completed")
	return sql, nil
}

// applyRegexFunctionFixes applies regex-based function fixes (fallback).
func (p *ProcessorAST) applyRegexFunctionFixes(sql string) string {
	p.logger.Info("=== Applying regex-based function fixes ===")
	sql = FixRedundantTrailingLanguageClauses(sql)
	sql = FixIncorrectLanguageDeclarations(sql)
	sql = FixMissingLanguageDeclarations(sql)
	sql = FixFunctionLanguageConflicts(sql)
	return sql
}
