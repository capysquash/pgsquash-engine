# internal/performance package map

## Domain Summary
- Houses infrastructure for memory-aware processing, streaming migration ingestion, and user-facing progress reporting.
- Supports large migration sets via batching, buffer pooling, GC hints, and detailed progress statistics.
- Note: Parallel dependency resolution is handled by internal/tracking/dependency_graph.go (single source of truth).

## Files (alphabetical)

### memory.go
- **Purpose**: Memory pooling, tracking, and deduplication utilities tailored to large SQL workloads.
- **Key Types**
  - `MemoryManager`: Manages statement/buffer pools, memory budgets, GC triggers, usage stats.
  - `MemoryStats`: Snapshot of current memory usage (fields in struct definition).
  - `LRUCache`: Generic LRU cache with hit-rate tracking.
  - `Deduplicator`: Hash-based duplicate detector with stats.
  - `MemoryOptimizedStatement`: Wrapper ensuring statement buffers are returned to pools.
- **Functions / Methods**
  - Constructors: `NewMemoryManager`, `NewLRUCache`, `NewDeduplicator`, `NewMemoryOptimizedStatement`.
  - Pool APIs: `GetStatement`, `PutStatement`, `GetBuffer`, `PutBuffer`.
  - Budget tracking: `TrackMemoryUsage`, `ReleaseMemory`, `triggerGC`, `GetMemoryStats`.
  - LRU ops: `Get`, `Put`, `Size`, `HitRate`, `Clear`.
  - Dedup ops: `IsDuplicate`, `Reset`, `GetStats`.
  - Memory statement helpers: `Release`, `estimateStatementSize`.

### progress.go
- **Purpose**: Aggregates progress updates across phases, reporters, and sources for CLI/TUI feedback.
- **Key Types**
  - `ProcessingPhase`, `ProgressMetrics`, `ProgressSummary`.
  - `DefaultProgressReporter`, `ConsoleProgressReporter`, `ProgressManager`, `ProgressAggregator`.
- **Functions / Methods**
  - Reporter construction: `NewProgressReporter`, `NewConsoleProgressReporter`.
  - Reporter APIs: `StartPhase`, `UpdateProgress`, `FinishPhase`, `SetOverallProgress`, `AddWarning`, `AddError`, `Complete`, `GetSummary`.
  - Manager operations: `NewProgressManager`, `AddReporter`, same API surface as reporters + `GetPrimaryReporter`.
  - Aggregator: `NewProgressAggregator`, `AddSource`, `GetAggregatedProgress`.
  - Internal helper: `notifyUpdate`, console `printProgress`.

### streaming.go
- **Purpose**: Stream-oriented migration processing pipeline with batching, worker pools, and progress tracking.
- **Key Types**
  - `StreamingProcessor`: Coordinates directory scanning, workers, and error handling.
  - `MigrationFile`, `ProcessedFile`, `ProcessingStats`: Data transfer objects (defined in file).
  - `BatchProcessor`: Groups files into size-based batches.
  - `FileStreamReader`: Windowed reader with pooled buffers.
  - `ProgressTracker`: Lightweight throughput tracker for streaming reads.
- **Functions / Methods**
  - Streaming lifecycle: `NewStreamingProcessor`, `Start`, `ProcessDirectory`, worker goroutines (`worker`, `processFile`, `errorHandler`), `GetResults`, `GetStats`, `Stop`.
  - Batching: `NewBatchProcessor`, `AddFile`, `GetCurrentBatch`, `HasPendingBatch`.
  - File streaming: `NewFileStreamReader`, `ReadChunk`, `GetPosition`.
  - Utilities: `isSQLFile`, `NewProgressTracker`, `ProgressTracker.Update`, `ProgressTracker.GetProgress`.

## Subdirectories
- _None._
