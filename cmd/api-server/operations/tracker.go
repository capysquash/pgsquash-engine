package operations

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	_ "github.com/lib/pq"
)

// Status represents the operation status
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "analyzing"  // Matches Platform's "analyzing" status
	StatusCompleted Status = "complete"   // Matches Platform's "complete" status
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

// Operation represents a migration operation
type Operation struct {
	ID          string                 `json:"id"`
	ProjectID   string                 `json:"project_id"`
	UserID      string                 `json:"user_id"`
	Type        string                 `json:"type"`
	SafetyLevel string                 `json:"safety_level"`
	Status      Status                 `json:"status"`
	Progress    int                    `json:"progress"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// OperationTracker manages database-backed operation tracking
type OperationTracker struct {
	db *sql.DB
}

// NewOperationTracker creates a new operation tracker
func NewOperationTracker(databaseURL string) (*OperationTracker, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	return &OperationTracker{db: db}, nil
}

// TrackerMetrics represents operation tracking metrics
type TrackerMetrics struct {
	TotalOperations  int `json:"total_operations"`
	ActiveOperations int `json:"active_operations"`
	CompletedToday   int `json:"completed_today"`
	FailedToday      int `json:"failed_today"`
	AvgDurationMs    int `json:"avg_duration_ms"`
}

// GetMetrics retrieves operation metrics from the database
func (ot *OperationTracker) GetMetrics(ctx context.Context) (*TrackerMetrics, error) {
	metrics := &TrackerMetrics{}
	
	var avgDuration float64
	
	// Total operations
	_ = ot.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM migration_runs`,
	).Scan(&metrics.TotalOperations)
	
	// Active operations (pending or running)
	_ = ot.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM migration_runs WHERE status IN ('pending', 'analyzing')`,
	).Scan(&metrics.ActiveOperations)
	
	// Completed today
	_ = ot.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM migration_runs 
		 WHERE status = 'complete' AND completed_at >= CURRENT_DATE`,
	).Scan(&metrics.CompletedToday)
	
	// Failed today
	_ = ot.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM migration_runs 
		 WHERE status = 'failed' AND updated_at >= CURRENT_DATE`,
	).Scan(&metrics.FailedToday)
	
	// Average duration for completed operations (in milliseconds)
	_ = ot.db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(processing_time_ms), 0) 
		 FROM migration_runs 
		 WHERE status = 'complete' AND processing_time_ms IS NOT NULL`,
	).Scan(&avgDuration)
	
	metrics.AvgDurationMs = int(avgDuration)
	
	return metrics, nil
}

// Create creates a new operation in the database
func (ot *OperationTracker) Create(ctx context.Context, userID, projectID, opType, safetyLevel string) (*Operation, error) {
	op := &Operation{
		ProjectID:   projectID,
		UserID:      userID,
		Type:        opType,
		SafetyLevel: safetyLevel,
		Status:      StatusPending,
		Progress:    0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO migration_runs (
			project_id, created_by, run_type, safety_level,
			status, progress, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	err := ot.db.QueryRowContext(ctx, query,
		op.ProjectID, op.UserID, op.Type, op.SafetyLevel,
		op.Status, op.Progress, op.CreatedAt, op.UpdatedAt,
	).Scan(&op.ID)

	return op, err
}

// Get retrieves an operation by ID
func (ot *OperationTracker) Get(ctx context.Context, operationID string) (*Operation, error) {
	op := &Operation{}

	query := `
		SELECT id, project_id, created_by, run_type, safety_level, status,
		       progress, operations, error, created_at, updated_at, completed_at
		FROM migration_runs
		WHERE id = $1
	`

	var resultJSON sql.NullString
	var completedAt sql.NullTime
	err := ot.db.QueryRowContext(ctx, query, operationID).Scan(
		&op.ID, &op.ProjectID, &op.UserID, &op.Type, &op.SafetyLevel,
		&op.Status, &op.Progress, &resultJSON, &op.Error,
		&op.CreatedAt, &op.UpdatedAt, &completedAt,
	)

	if err != nil {
		return nil, err
	}

	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}

	if resultJSON.Valid && resultJSON.String != "" {
		if err := json.Unmarshal([]byte(resultJSON.String), &op.Result); err != nil {
			// Log but don't fail - result is optional
			_ = err
		}
	}

	return op, nil
}

// UpdateProgress updates the operation progress
func (ot *OperationTracker) UpdateProgress(ctx context.Context, operationID string, progress int, status Status) error {
	query := `
		UPDATE migration_runs
		SET progress = $1, status = $2, updated_at = NOW()
		WHERE id = $3
	`
	_, err := ot.db.ExecContext(ctx, query, progress, status, operationID)
	return err
}

// Complete marks an operation as completed
func (ot *OperationTracker) Complete(ctx context.Context, operationID string, result map[string]interface{}) error {
	resultJSON, _ := json.Marshal(result)

	query := `
		UPDATE migration_runs
		SET status = $1, progress = 100, operations = $2,
		    completed_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`
	_, err := ot.db.ExecContext(ctx, query, StatusCompleted, resultJSON, operationID)
	return err
}

// Fail marks an operation as failed
func (ot *OperationTracker) Fail(ctx context.Context, operationID string, errMsg string) error {
	query := `
		UPDATE migration_runs
		SET status = $1, error = $2, completed_at = NOW(), updated_at = NOW()
		WHERE id = $3
	`
	_, err := ot.db.ExecContext(ctx, query, StatusFailed, errMsg, operationID)
	return err
}

// ListUserOperations lists operations for a user
func (ot *OperationTracker) ListUserOperations(ctx context.Context, userID string, limit int) ([]*Operation, error) {
	query := `
		SELECT id, project_id, created_by, run_type, safety_level, status,
		       progress, error, created_at, updated_at, completed_at
		FROM migration_runs
		WHERE created_by = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := ot.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log but don't fail on close error
			_ = err
		}
	}()

	var operations []*Operation
	for rows.Next() {
		op := &Operation{}
		var completedAt sql.NullTime
		err := rows.Scan(
			&op.ID, &op.ProjectID, &op.UserID, &op.Type, &op.SafetyLevel,
			&op.Status, &op.Progress, &op.Error,
			&op.CreatedAt, &op.UpdatedAt, &completedAt,
		)
		if err != nil {
			return nil, err
		}
		if completedAt.Valid {
			op.CompletedAt = &completedAt.Time
		}
		operations = append(operations, op)
	}

	return operations, nil
}

// Close closes the database connection
func (ot *OperationTracker) Close() error {
	return ot.db.Close()
}
