# Validation Metrics and Monitoring

pgsquash provides comprehensive validation metrics with support for both JSON and Prometheus export formats, enabling detailed monitoring and observability of validation operations.

## Overview

The validation metrics system tracks:

- **Timing metrics**: Duration, average query time, slowest query
- **Count metrics**: Validations, objects, queries, errors, warnings
- **Schema metrics**: Tables, indexes, functions, triggers, constraints, views
- **Docker metrics**: Container usage, validation time, failures
- **Error/Warning breakdown**: By code and severity
- **Resource metrics**: Memory usage, CPU time
- **Success rates**: Validation and fix success rates

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│               ValidationMetrics                           │
│  (Thread-Safe Metrics Collector)                         │
├──────────────────────────────────────────────────────────┤
│ • RecordValidation(success, duration)                    │
│ • RecordQuery(duration)                                  │
│ • RecordError(code, severity)                            │
│ • RecordWarning(code)                                    │
│ • RecordDockerValidation(approach, duration, success)    │
│ • RecordSchemaObject(objectType)                         │
│ • RecordExtensionInstall(success, duration)              │
│ • RecordFix(success)                                     │
│ • UpdateResourceMetrics(memory, cpu)                     │
├──────────────────────────────────────────────────────────┤
│ ExportJSON(writer) → JSON format                         │
│ ExportPrometheus(writer) → Prometheus format             │
└──────────────────────────────────────────────────────────┘
```

## Usage

### Creating a Metrics Collector

```go
import "github.com/CAPYSQUASH/pgsquash-engine/internal/validation"

// Create metrics collector
metrics := validation.NewValidationMetrics()

// Set metadata
metrics.SetValidationLevel("COMPREHENSIVE")
metrics.SetPostgreSQLVersion("15.4")
```

### Recording Metrics

#### Validation Metrics

```go
// Start validation
startTime := time.Now()

// ... perform validation ...

// Record validation result
duration := time.Since(startTime)
metrics.RecordValidation(success, duration)

// Set timing explicitly
metrics.SetValidationTimes(startTime, time.Now())
```

#### Query Metrics

```go
queryStart := time.Now()

// Execute query
_, err := db.Query("SELECT ...")

// Record query
queryDuration := time.Since(queryStart)
metrics.RecordQuery(queryDuration)
```

#### Error and Warning Metrics

```go
// Record error with code and severity
metrics.RecordError("CONSTRAINT_VIOLATION", "HIGH")
metrics.RecordError("SYNTAX_ERROR", "CRITICAL")

// Record warning
metrics.RecordWarning("DEPRECATED_SYNTAX")
metrics.RecordWarning("MISSING_INDEX")
```

#### Schema Object Metrics

```go
// Record validated objects
metrics.RecordSchemaObject("TABLE")
metrics.RecordSchemaObject("INDEX")
metrics.RecordSchemaObject("FUNCTION")
metrics.RecordSchemaObject("CONSTRAINT")
```

#### Docker Validation Metrics

```go
dockerStart := time.Now()

// Perform Docker validation
success := performDockerValidation()

dockerDuration := time.Since(dockerStart)
metrics.RecordDockerValidation("TWO_CONTAINERS", dockerDuration, success)
```

#### Extension Install Metrics

```go
installStart := time.Now()

// Install extension
err := installExtension("pgvector")

installDuration := time.Since(installStart)
metrics.RecordExtensionInstall(err == nil, installDuration)
```

#### SQL Fix Metrics

```go
// Attempt fix
success := applySQLFix(sql)

// Record fix result
metrics.RecordFix(success)
```

#### Resource Metrics

```go
var mem runtime.MemStats
runtime.ReadMemStats(&mem)

cpuTime := getCPUTime() // Implementation-specific

metrics.UpdateResourceMetrics(int64(mem.Alloc), cpuTime)
```

## Export Formats

### JSON Export

Export metrics as structured JSON:

```go
import "os"

// Export to file
file, err := os.Create("validation-metrics.json")
if err != nil {
    log.Fatal(err)
}
defer file.Close()

err = metrics.ExportJSON(file)
if err != nil {
    log.Fatal(err)
}
```

**JSON Output Example**:

```json
{
  "total_duration_ms": 15234,
  "validation_start_time": "2025-01-15T10:30:00Z",
  "validation_end_time": "2025-01-15T10:30:15Z",
  "average_query_duration_ms": 45,
  "slowest_query_duration_ms": 250,
  "total_validations": 10,
  "successful_validations": 9,
  "failed_validations": 1,
  "success_rate_percent": 90.0,
  "objects_validated": 156,
  "queries_executed": 342,
  "errors_found": 3,
  "warnings_found": 12,
  "tables_validated": 45,
  "indexes_validated": 67,
  "functions_validated": 28,
  "triggers_validated": 5,
  "constraints_validated": 89,
  "views_validated": 11,
  "extensions_detected": 4,
  "docker_containers_spun": 2,
  "docker_validation_time_ms": 8500,
  "docker_failures": 0,
  "errors_by_code": {
    "CONSTRAINT_VIOLATION": 2,
    "SYNTAX_ERROR": 1
  },
  "errors_by_severity": {
    "HIGH": 2,
    "CRITICAL": 1
  },
  "warnings_by_code": {
    "DEPRECATED_SYNTAX": 5,
    "MISSING_INDEX": 7
  },
  "approach_usage": {
    "TWO_CONTAINERS": 2
  },
  "extension_install_attempts": 4,
  "extension_install_failures": 0,
  "extension_install_time_ms": 1200,
  "fixes_attempted": 3,
  "fixes_succeeded": 3,
  "fixes_failed": 0,
  "fix_success_rate_percent": 100.0,
  "peak_memory_bytes": 125829120,
  "cpu_time_ms": 3450,
  "validation_level": "COMPREHENSIVE",
  "postgresql_version": "15.4",
  "last_updated": "2025-01-15T10:30:15Z"
}
```

### Prometheus Export

Export metrics in Prometheus format for scraping:

```go
import (
    "net/http"
    "github.com/CAPYSQUASH/pgsquash-engine/internal/validation"
)

// Create metrics collector
metrics := validation.NewValidationMetrics()

// Expose metrics endpoint
http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain; version=0.0.4")
    err := metrics.ExportPrometheus(w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
})

http.ListenAndServe(":9090", nil)
```

**Prometheus Output Example**:

```prometheus
# HELP pgsquash_validation_duration_seconds Total validation duration
# TYPE pgsquash_validation_duration_seconds gauge
pgsquash_validation_duration_seconds 15.234

# HELP pgsquash_validation_query_duration_avg_seconds Average query duration
# TYPE pgsquash_validation_query_duration_avg_seconds gauge
pgsquash_validation_query_duration_avg_seconds 0.045

# HELP pgsquash_validations_total Total validations performed
# TYPE pgsquash_validations_total counter
pgsquash_validations_total 10

# HELP pgsquash_validations_successful_total Successful validations
# TYPE pgsquash_validations_successful_total counter
pgsquash_validations_successful_total 9

# HELP pgsquash_validation_success_rate Validation success rate (0-1)
# TYPE pgsquash_validation_success_rate gauge
pgsquash_validation_success_rate 0.9

# HELP pgsquash_validation_objects_total Total objects validated
# TYPE pgsquash_validation_objects_total counter
pgsquash_validation_objects_total 156

# HELP pgsquash_validation_errors_by_code Errors by error code
# TYPE pgsquash_validation_errors_by_code counter
pgsquash_validation_errors_by_code{code="CONSTRAINT_VIOLATION"} 2
pgsquash_validation_errors_by_code{code="SYNTAX_ERROR"} 1

# HELP pgsquash_docker_containers_total Docker containers spun up
# TYPE pgsquash_docker_containers_total counter
pgsquash_docker_containers_total 2

# HELP pgsquash_memory_peak_bytes Peak memory usage
# TYPE pgsquash_memory_peak_bytes gauge
pgsquash_memory_peak_bytes 125829120
```

## Metric Categories

### Timing Metrics

| Metric                      | Type  | Description                               |
| --------------------------- | ----- | ----------------------------------------- |
| `total_duration_ms`         | Gauge | Total validation duration in milliseconds |
| `average_query_duration_ms` | Gauge | Average query execution time              |
| `slowest_query_duration_ms` | Gauge | Duration of the slowest query             |
| `docker_validation_time_ms` | Gauge | Time spent in Docker validation           |
| `extension_install_time_ms` | Gauge | Time spent installing extensions          |

### Count Metrics

| Metric                   | Type    | Description                    |
| ------------------------ | ------- | ------------------------------ |
| `total_validations`      | Counter | Total validation runs          |
| `successful_validations` | Counter | Successful validation runs     |
| `failed_validations`     | Counter | Failed validation runs         |
| `objects_validated`      | Counter | Total schema objects validated |
| `queries_executed`       | Counter | Total queries executed         |
| `errors_found`           | Counter | Total errors detected          |
| `warnings_found`         | Counter | Total warnings issued          |

### Schema Object Metrics

| Metric                  | Type    | Description                        |
| ----------------------- | ------- | ---------------------------------- |
| `tables_validated`      | Counter | Tables validated                   |
| `indexes_validated`     | Counter | Indexes validated                  |
| `functions_validated`   | Counter | Functions/procedures validated     |
| `triggers_validated`    | Counter | Triggers validated                 |
| `constraints_validated` | Counter | Constraints validated              |
| `views_validated`       | Counter | Views/materialized views validated |
| `extensions_detected`   | Gauge   | Extensions detected in migrations  |

### Docker Metrics

| Metric                      | Type    | Description                |
| --------------------------- | ------- | -------------------------- |
| `docker_containers_spun`    | Counter | Docker containers created  |
| `docker_validation_time_ms` | Gauge   | Time in Docker validation  |
| `docker_failures`           | Counter | Docker validation failures |

### Error/Warning Breakdown

| Metric               | Type    | Labels     | Description                  |
| -------------------- | ------- | ---------- | ---------------------------- |
| `errors_by_code`     | Counter | `code`     | Errors grouped by error code |
| `errors_by_severity` | Counter | `severity` | Errors grouped by severity   |
| `warnings_by_code`   | Counter | `code`     | Warnings grouped by code     |
| `approach_usage`     | Counter | `approach` | Validation approach usage    |

### Fix Metrics

| Metric                     | Type    | Description              |
| -------------------------- | ------- | ------------------------ |
| `fixes_attempted`          | Counter | SQL fixes attempted      |
| `fixes_succeeded`          | Counter | SQL fixes succeeded      |
| `fixes_failed`             | Counter | SQL fixes failed         |
| `fix_success_rate_percent` | Gauge   | Fix success rate (0-100) |

### Resource Metrics

| Metric              | Type  | Description                   |
| ------------------- | ----- | ----------------------------- |
| `peak_memory_bytes` | Gauge | Peak memory usage in bytes    |
| `cpu_time_ms`       | Gauge | CPU time used in milliseconds |

## Integration with Validator

Integrate metrics into the validation flow:

```go
// In SchemaValidator
type SchemaValidator struct {
    config  *ValidationConfig
    db      *sql.DB
    metrics *ValidationMetrics // Add metrics
}

func NewSchemaValidator(config *ValidationConfig) *SchemaValidator {
    return &SchemaValidator{
        config:  config,
        metrics: NewValidationMetrics(),
    }
}

func (v *SchemaValidator) Validate(ctx context.Context, sql string) (*ValidationResult, error) {
    startTime := time.Now()
    v.metrics.SetValidationLevel(string(v.config.Level))

    // Perform validation...
    result, err := v.performValidation(ctx, sql)

    // Record metrics
    duration := time.Since(startTime)
    v.metrics.RecordValidation(result.Success, duration)
    v.metrics.SetValidationTimes(startTime, time.Now())

    return result, err
}

// Get metrics for export
func (v *SchemaValidator) GetMetrics() *ValidationMetrics {
    return v.metrics
}
```

## Monitoring Dashboards

### Prometheus + Grafana

1. **Expose metrics endpoint**:
   ```bash
   # Start metrics server
   pgsquash serve-metrics --port 9090
   ```

2. **Configure Prometheus scraping** (`prometheus.yml`):
   ```yaml
   scrape_configs:
     - job_name: 'pgsquash'
       static_configs:
         - targets: ['localhost:9090']
       scrape_interval: 15s
   ```

3. **Import Grafana dashboard**:
   - Use pre-built dashboard JSON
   - Monitor validation success rates, error trends, performance metrics

### Key Dashboard Panels

**Success Rate Over Time**:

```promql
rate(pgsquash_validations_successful_total[5m]) / rate(pgsquash_validations_total[5m])
```

**Error Rate by Code**:

```promql
sum by (code) (rate(pgsquash_validation_errors_by_code[5m]))
```

**Validation Duration**:

```promql
histogram_quantile(0.95, rate(pgsquash_validation_duration_seconds[5m]))
```

**Memory Usage**:

```promql
pgsquash_memory_peak_bytes
```

## CLI Integration

Export metrics from CLI:

```bash
# Export to JSON file
pgsquash validate migrations/ clean/ --export-metrics validation-metrics.json

# Export to Prometheus format
pgsquash validate migrations/ clean/ --export-metrics-format prometheus --export-metrics metrics.prom

# Start metrics server
pgsquash serve-metrics --port 9090
```

## Best Practices

### 1. Metric Collection

- **Record all operations**: Don't skip metric recording, even on errors
- **Use consistent labels**: Error codes, severities, approaches should be consistent
- **Thread safety**: Metrics collector is thread-safe, safe to use from multiple goroutines

### 2. Export Timing

- **After validation**: Export metrics after validation completes
- **Periodic export**: For long-running services, export periodically
- **On demand**: Expose HTTP endpoint for on-demand metrics

### 3. Resource Monitoring

- **Peak memory**: Track peak memory usage to identify memory leaks
- **CPU time**: Monitor CPU usage for performance optimization
- **Docker overhead**: Track Docker validation overhead separately

### 4. Error Analysis

- **Group by code**: Analyze errors by error code to identify patterns
- **Severity tracking**: Monitor high-severity errors closely
- **Trend analysis**: Track error trends over time

### 5. Performance Optimization

- **Query duration**: Identify slow queries for optimization
- **Validation time**: Track total validation time trends
- **Docker optimization**: Monitor Docker validation overhead

## Troubleshooting

### High Memory Usage

```bash
# Check peak memory
cat validation-metrics.json | jq '.peak_memory_bytes'

# If > 500MB, investigate:
# - Large migrations
# - Memory leaks
# - Docker container overhead
```

### Slow Validations

```bash
# Check slowest query
cat validation-metrics.json | jq '.slowest_query_duration_ms'

# Check average query duration
cat validation-metrics.json | jq '.average_query_duration_ms'

# If high, optimize:
# - Add database indexes
# - Reduce query complexity
# - Enable query caching
```

### High Error Rates

```bash
# Check error breakdown
cat validation-metrics.json | jq '.errors_by_code'
cat validation-metrics.json | jq '.errors_by_severity'

# Focus on HIGH and CRITICAL severity errors
```

## Further Reading

- [Validation Documentation](./validation.md) - Complete validation guide
- [Docker Validation](./deployment/docker.md) - Docker-based validation
- [Configuration Reference](./configuration.md) - Metrics configuration
- [Paranoid Mode](./paranoid-mode-validation.md) - Paranoid mode validation
