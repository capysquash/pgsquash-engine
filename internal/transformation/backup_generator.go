package transformation

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/capysquash/pg-squash/internal/parser"
)

// BackupType defines the type of backup to generate
type BackupType int

const (
	SchemaOnly BackupType = iota
	DataOnly
	SchemaAndData
	Structure
	DDLOnly
)

// BackupFormat defines the output format for backups
type BackupFormat int

const (
	SQLFormat BackupFormat = iota
	CustomFormat
	TarFormat
	DirectoryFormat
)

// BackupConfig controls backup generation behavior
type BackupConfig struct {
	Type           BackupType   `json:"type"`
	Format         BackupFormat `json:"format"`
	Compression    bool         `json:"compression"`
	VerboseOutput  bool         `json:"verbose_output"`
	IncludeDrops   bool         `json:"include_drops"`
	SchemaOnly     bool         `json:"schema_only"`
	DataOnly       bool         `json:"data_only"`
	InsertFormat   bool         `json:"insert_format"`
	ColumnInserts  bool         `json:"column_inserts"`
	IfExists       bool         `json:"if_exists"`
	CreateDatabase bool         `json:"create_database"`
	CleanFirst     bool         `json:"clean_first"`
	Encoding       string       `json:"encoding"`
	ExcludeTables  []string     `json:"exclude_tables"`
	IncludeTables  []string     `json:"include_tables"`
	ExcludeSchemas []string     `json:"exclude_schemas"`
	IncludeSchemas []string     `json:"include_schemas"`
}

// DefaultBackupConfig returns sensible backup defaults
func DefaultBackupConfig() *BackupConfig {
	return &BackupConfig{
		Type:          SchemaAndData,
		Format:        SQLFormat,
		Compression:   true,
		VerboseOutput: false,
		IncludeDrops:  false,
		InsertFormat:  true,
		ColumnInserts: true,
		IfExists:      true,
		Encoding:      "UTF8",
	}
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	BackupPath      string        `json:"backup_path"`
	Size            int64         `json:"size"`
	Duration        time.Duration `json:"duration"`
	TablesBackedUp  int           `json:"tables_backed_up"`
	SchemasBackedUp int           `json:"schemas_backed_up"`
	Success         bool          `json:"success"`
	Error           string        `json:"error,omitempty"`
	Warnings        []string      `json:"warnings"`
}

// RollbackScript represents a rollback operation
type RollbackScript struct {
	ID           string    `json:"id"`
	Description  string    `json:"description"`
	SQL          string    `json:"sql"`
	CreatedAt    time.Time `json:"created_at"`
	Order        int       `json:"order"`
	Dependencies []string  `json:"dependencies"`
}

// BackupGenerator handles database backup and rollback generation
type BackupGenerator struct {
	config     *BackupConfig
	db         *sql.DB
	pgDumpPath string
	workDir    string
}

// NewBackupGenerator creates a new backup generator
func NewBackupGenerator(config *BackupConfig, db *sql.DB) *BackupGenerator {
	if config == nil {
		config = DefaultBackupConfig()
	}

	return &BackupGenerator{
		config:     config,
		db:         db,
		pgDumpPath: findPgDumpPath(),
		workDir:    os.TempDir(),
	}
}

// findPgDumpPath attempts to locate pg_dump binary
func findPgDumpPath() string {
	// Common locations for pg_dump
	paths := []string{
		"/usr/local/bin/pg_dump",
		"/usr/bin/pg_dump",
		"/opt/postgresql/bin/pg_dump",
		"/Applications/Postgres.app/Contents/Versions/latest/bin/pg_dump",
		"pg_dump", // Assume it's in PATH
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "pg_dump" // Fallback to PATH
}

// SetWorkingDirectory sets the working directory for backup operations
func (bg *BackupGenerator) SetWorkingDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}
	bg.workDir = dir
	return nil
}

// GeneratePreMigrationBackup creates a backup before applying migrations
func (bg *BackupGenerator) GeneratePreMigrationBackup(ctx context.Context, dbURL string) (*BackupResult, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("pre_migration_backup_%s", timestamp)

	return bg.generateBackup(ctx, dbURL, backupName, "Pre-migration safety backup")
}

// GeneratePostMigrationBackup creates a backup after applying migrations
func (bg *BackupGenerator) GeneratePostMigrationBackup(ctx context.Context, dbURL string) (*BackupResult, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("post_migration_backup_%s", timestamp)

	return bg.generateBackup(ctx, dbURL, backupName, "Post-migration verification backup")
}

// generateBackup performs the actual backup operation
func (bg *BackupGenerator) generateBackup(ctx context.Context, dbURL, name, description string) (*BackupResult, error) {
	startTime := time.Now()

	result := &BackupResult{
		Success:  false,
		Warnings: make([]string, 0),
	}

	// Build backup file path
	extension := ".sql"
	if bg.config.Format == CustomFormat {
		extension = ".custom"
	} else if bg.config.Format == TarFormat {
		extension = ".tar"
	}

	if bg.config.Compression && bg.config.Format == SQLFormat {
		extension += ".gz"
	}

	backupPath := filepath.Join(bg.workDir, name+extension)
	result.BackupPath = backupPath

	// Build pg_dump command arguments
	args := bg.buildPgDumpArgs(dbURL, backupPath)

	// Execute backup
	err := bg.executePgDump(ctx, args)
	if err != nil {
		result.Error = err.Error()
		return result, err
	}

	// Get backup file info
	info, err := os.Stat(backupPath)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to get backup info: %v", err)
		return result, err
	}

	result.Size = info.Size()
	result.Duration = time.Since(startTime)
	result.Success = true

	// Analyze backup content
	if bg.config.VerboseOutput {
		err = bg.analyzeBackup(backupPath, result)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to analyze backup: %v", err))
		}
	}

	return result, nil
}

// buildPgDumpArgs constructs pg_dump command arguments
func (bg *BackupGenerator) buildPgDumpArgs(dbURL, outputPath string) []string {
	args := []string{bg.pgDumpPath}

	// Format options
	switch bg.config.Format {
	case CustomFormat:
		args = append(args, "--format=custom")
	case TarFormat:
		args = append(args, "--format=tar")
	case DirectoryFormat:
		args = append(args, "--format=directory")
	default:
		args = append(args, "--format=plain")
	}

	// Backup type options
	if bg.config.SchemaOnly || bg.config.Type == SchemaOnly || bg.config.Type == DDLOnly {
		args = append(args, "--schema-only")
	} else if bg.config.DataOnly || bg.config.Type == DataOnly {
		args = append(args, "--data-only")
	}

	// Output options
	if bg.config.VerboseOutput {
		args = append(args, "--verbose")
	}

	if bg.config.IncludeDrops || bg.config.CleanFirst {
		args = append(args, "--clean")
	}

	if bg.config.IfExists {
		args = append(args, "--if-exists")
	}

	if bg.config.CreateDatabase {
		args = append(args, "--create")
	}

	if bg.config.InsertFormat {
		args = append(args, "--inserts")
	}

	if bg.config.ColumnInserts {
		args = append(args, "--column-inserts")
	}

	// Compression
	if bg.config.Compression && bg.config.Format != CustomFormat {
		args = append(args, "--compress=9")
	}

	// Encoding
	if bg.config.Encoding != "" {
		args = append(args, fmt.Sprintf("--encoding=%s", bg.config.Encoding))
	}

	// Table filters
	for _, table := range bg.config.ExcludeTables {
		args = append(args, fmt.Sprintf("--exclude-table=%s", table))
	}

	for _, table := range bg.config.IncludeTables {
		args = append(args, fmt.Sprintf("--table=%s", table))
	}

	// Schema filters
	for _, schema := range bg.config.ExcludeSchemas {
		args = append(args, fmt.Sprintf("--exclude-schema=%s", schema))
	}

	for _, schema := range bg.config.IncludeSchemas {
		args = append(args, fmt.Sprintf("--schema=%s", schema))
	}

	// Output file
	args = append(args, "--file="+outputPath)

	// Database URL
	args = append(args, dbURL)

	return args
}

// executePgDump runs the pg_dump command
func (bg *BackupGenerator) executePgDump(ctx context.Context, args []string) error {
	// This would typically use os/exec to run pg_dump
	// For now, we'll simulate the operation
	fmt.Printf("Would execute: %s\n", strings.Join(args, " "))

	// Simulate some processing time
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// Backup completed
	}

	return nil
}

// analyzeBackup analyzes the backup content for statistics
func (bg *BackupGenerator) analyzeBackup(backupPath string, result *BackupResult) error {
	// This would analyze the backup file to extract statistics
	// For schema backups, count tables, indexes, functions, etc.
	// For data backups, estimate row counts

	result.TablesBackedUp = 10 // Simulated
	result.SchemasBackedUp = 2 // Simulated

	return nil
}

// GenerateRollbackScript creates rollback scripts for migrations
func (bg *BackupGenerator) GenerateRollbackScript(ctx context.Context, statements []parser.Statement) ([]*RollbackScript, error) {
	rollbacks := make([]*RollbackScript, 0)

	for i, stmt := range statements {
		rollback, err := bg.generateRollbackForStatement(stmt, i)
		if err != nil {
			return nil, fmt.Errorf("failed to generate rollback for statement %d: %w", i, err)
		}

		if rollback != nil {
			rollbacks = append(rollbacks, rollback)
		}
	}

	// Reverse order for proper rollback sequence
	for i := len(rollbacks)/2 - 1; i >= 0; i-- {
		opp := len(rollbacks) - 1 - i
		rollbacks[i], rollbacks[opp] = rollbacks[opp], rollbacks[i]
	}

	// Update order numbers
	for i, rollback := range rollbacks {
		rollback.Order = i + 1
	}

	return rollbacks, nil
}

// generateRollbackForStatement creates a rollback script for a single statement
func (bg *BackupGenerator) generateRollbackForStatement(stmt parser.Statement, order int) (*RollbackScript, error) {
	rollback := &RollbackScript{
		ID:        fmt.Sprintf("rollback_%d_%d", time.Now().Unix(), order),
		CreatedAt: time.Now(),
		Order:     order,
	}

	switch stmt.Operation {
	case parser.OpCreate:
		switch stmt.ObjectType {
		case parser.TypeTable:
			rollback.Description = fmt.Sprintf("Drop table %s", stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE;", stmt.ObjectName)
		case parser.TypeIndex:
			rollback.Description = fmt.Sprintf("Drop index %s", stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("DROP INDEX IF EXISTS %s;", stmt.ObjectName)
		case parser.TypeFunction:
			rollback.Description = fmt.Sprintf("Drop function %s", stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("DROP FUNCTION IF EXISTS %s CASCADE;", stmt.ObjectName)
		default:
			rollback.Description = fmt.Sprintf("Drop %s %s", stmt.ObjectType, stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("-- Cannot automatically rollback CREATE %s %s", stmt.ObjectType, stmt.ObjectName)
		}

	case parser.OpDrop:
		switch stmt.ObjectType {
		case parser.TypeTable:
			rollback.Description = fmt.Sprintf("Recreate table %s", stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("-- Cannot automatically rollback DROP TABLE %s (requires backup restore)", stmt.ObjectName)
		case parser.TypeIndex:
			rollback.Description = fmt.Sprintf("Recreate index %s", stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("-- Cannot automatically rollback DROP INDEX %s (requires original definition)", stmt.ObjectName)
		case parser.TypeFunction:
			rollback.Description = fmt.Sprintf("Recreate function %s", stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("-- Cannot automatically rollback DROP FUNCTION %s (requires backup restore)", stmt.ObjectName)
		default:
			rollback.Description = fmt.Sprintf("Recreate %s %s", stmt.ObjectType, stmt.ObjectName)
			rollback.SQL = fmt.Sprintf("-- Cannot automatically rollback DROP %s %s", stmt.ObjectType, stmt.ObjectName)
		}

	case parser.OpAlter:
		rollback.Description = fmt.Sprintf("Reverse ALTER %s %s", stmt.ObjectType, stmt.ObjectName)
		rollback.SQL = bg.generateAlterTableRollback(stmt)

	case parser.OpInsert:
		rollback.Description = fmt.Sprintf("Delete inserted data from %s", stmt.ObjectName)
		rollback.SQL = bg.generateInsertRollback(stmt)

	case parser.OpUpdate:
		rollback.Description = fmt.Sprintf("Reverse UPDATE on %s", stmt.ObjectName)
		rollback.SQL = fmt.Sprintf("-- Cannot automatically rollback UPDATE on %s (requires pre-change snapshot)", stmt.ObjectName)

	case parser.OpDelete:
		rollback.Description = fmt.Sprintf("Restore deleted data to %s", stmt.ObjectName)
		rollback.SQL = fmt.Sprintf("-- Cannot automatically rollback DELETE from %s (requires backup restore)", stmt.ObjectName)

	default:
		// For statements we don't know how to rollback
		return nil, nil
	}

	return rollback, nil
}

// generateAlterTableRollback creates rollback for ALTER TABLE statements
func (bg *BackupGenerator) generateAlterTableRollback(stmt parser.Statement) string {
	// This would analyze the specific ALTER TABLE operation and generate the reverse
	// For now, return a placeholder
	return fmt.Sprintf("-- Manual rollback required for ALTER TABLE %s", stmt.ObjectName)
}

// generateInsertRollback creates rollback for INSERT statements
func (bg *BackupGenerator) generateInsertRollback(stmt parser.Statement) string {
	// For INSERT statements, we could generate DELETE statements
	// This would require parsing the INSERT to extract the data
	return fmt.Sprintf("-- DELETE rollback for INSERT into %s (implement based on INSERT values)", stmt.ObjectName)
}

// ValidateBackup verifies that a backup can be restored
func (bg *BackupGenerator) ValidateBackup(ctx context.Context, backupPath string) error {
	// Check if backup file exists and is readable
	info, err := os.Stat(backupPath)
	if err != nil {
		return fmt.Errorf("backup file not accessible: %w", err)
	}

	if info.Size() == 0 {
		return fmt.Errorf("backup file is empty")
	}

	// For SQL format, do basic syntax validation
	if strings.HasSuffix(backupPath, ".sql") {
		return bg.validateSQLBackup(backupPath)
	}

	return nil
}

// validateSQLBackup performs basic validation on SQL backup files
func (bg *BackupGenerator) validateSQLBackup(backupPath string) error {
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Basic checks
	contentStr := string(content)

	if !strings.Contains(contentStr, "PostgreSQL database dump") {
		return fmt.Errorf("backup does not appear to be a PostgreSQL dump")
	}

	// Check for common dump sections
	hasSchema := strings.Contains(contentStr, "CREATE TABLE") || strings.Contains(contentStr, "CREATE SCHEMA")
	hasData := strings.Contains(contentStr, "COPY ") || strings.Contains(contentStr, "INSERT INTO")

	if !hasSchema && !hasData {
		return fmt.Errorf("backup appears to be empty (no schema or data found)")
	}

	return nil
}

// CleanupOldBackups removes old backup files based on retention policy
func (bg *BackupGenerator) CleanupOldBackups(maxAge time.Duration, maxCount int) error {
	files, err := filepath.Glob(filepath.Join(bg.workDir, "*backup*.sql"))
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	// Sort by modification time
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var fileInfos []fileInfo
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		fileInfos = append(fileInfos, fileInfo{path: file, modTime: info.ModTime()})
	}

	// Remove files older than maxAge
	cutoff := time.Now().Add(-maxAge)
	for _, info := range fileInfos {
		if info.modTime.Before(cutoff) {
			os.Remove(info.path)
		}
	}

	// TODO: Implement maxCount logic

	return nil
}
