// Package performance provides performance optimization and streaming capabilities.
// It handles parallel processing, memory management, and progress tracking
// for large-scale migration squashing operations.
package performance

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/CAPYSQUASH/pgsquash-engine/internal/errors"
	"github.com/CAPYSQUASH/pgsquash-engine/internal/types"
)

// ParallelDependencyResolver handles concurrent dependency resolution with topological sorting
type ParallelDependencyResolver struct {
	graph       *DependencyGraph
	ready       chan ObjectID
	processing  chan ObjectID
	completed   chan ObjectID
	errors      chan error
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	workerCount int
	stats       *ResolutionStats
	processor   DependencyProcessor
}

// ObjectID represents a unique object identifier
type ObjectID struct {
	Type   types.ObjectType `json:"type"`
	Schema string            `json:"schema"`
	Name   string            `json:"name"`
}

// String returns a string representation of the ObjectID
func (oid ObjectID) String() string {
	if oid.Schema != "" && oid.Schema != "public" {
		return fmt.Sprintf("%s.%s::%s", oid.Schema, oid.Name, oid.Type)
	}
	return fmt.Sprintf("%s::%s", oid.Name, oid.Type)
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	ID           ObjectID
	Dependencies []ObjectID
	Dependents   []ObjectID
	Processed    bool
	Processing   bool
	Data         interface{} // Associated data (e.g., Statement, Migration)
	Priority     int         // Processing priority
	Level        int         // Dependency level (0 = no dependencies)
}

// DependencyGraph manages object dependencies and topological ordering
type DependencyGraph struct {
	nodes    map[ObjectID]*DependencyNode
	levels   [][]ObjectID // Nodes grouped by dependency level
	mu       sync.RWMutex
	modified bool
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[ObjectID]*DependencyNode),
	}
}

// AddNode adds a node to the graph
func (dg *DependencyGraph) AddNode(id ObjectID, data interface{}) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if _, exists := dg.nodes[id]; !exists {
		dg.nodes[id] = &DependencyNode{
			ID:           id,
			Dependencies: make([]ObjectID, 0),
			Dependents:   make([]ObjectID, 0),
			Data:         data,
		}
		dg.modified = true
	}
}

// AddDependency adds a dependency relationship
func (dg *DependencyGraph) AddDependency(dependent, dependency ObjectID) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// Ensure both nodes exist
	if _, exists := dg.nodes[dependent]; !exists {
		dg.nodes[dependent] = &DependencyNode{
			ID:           dependent,
			Dependencies: make([]ObjectID, 0),
			Dependents:   make([]ObjectID, 0),
		}
	}

	if _, exists := dg.nodes[dependency]; !exists {
		dg.nodes[dependency] = &DependencyNode{
			ID:           dependency,
			Dependencies: make([]ObjectID, 0),
			Dependents:   make([]ObjectID, 0),
		}
	}

	// Add the dependency relationship
	dependentNode := dg.nodes[dependent]
	dependencyNode := dg.nodes[dependency]

	// Check if dependency already exists
	for _, dep := range dependentNode.Dependencies {
		if dep == dependency {
			return // Already exists
		}
	}

	dependentNode.Dependencies = append(dependentNode.Dependencies, dependency)
	dependencyNode.Dependents = append(dependencyNode.Dependents, dependent)
	dg.modified = true
}

// TopologicalSort performs topological sorting and returns processing levels
func (dg *DependencyGraph) TopologicalSort() ([][]ObjectID, error) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if !dg.modified && len(dg.levels) > 0 {
		return dg.levels, nil
	}

	// Kahn's algorithm for topological sorting
	inDegree := make(map[ObjectID]int)
	for id, node := range dg.nodes {
		inDegree[id] = len(node.Dependencies)
	}

	// Find nodes with no dependencies (level 0)
	var queue []ObjectID
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	levels := make([][]ObjectID, 0)
	processed := make(map[ObjectID]bool)

	level := 0
	for len(queue) > 0 {
		currentLevel := make([]ObjectID, len(queue))
		copy(currentLevel, queue)
		levels = append(levels, currentLevel)

		nextQueue := make([]ObjectID, 0)

		for _, id := range queue {
			processed[id] = true
			dg.nodes[id].Level = level

			// Reduce in-degree for dependents
			for _, dependent := range dg.nodes[id].Dependents {
				if !processed[dependent] {
					inDegree[dependent]--
					if inDegree[dependent] == 0 {
						nextQueue = append(nextQueue, dependent)
					}
				}
			}
		}

		queue = nextQueue
		level++
	}

	// Check for cycles
	if len(processed) != len(dg.nodes) {
		var cycleNodes []ObjectID
		for id := range dg.nodes {
			if !processed[id] {
				cycleNodes = append(cycleNodes, id)
			}
		}
		return nil, errors.New(errors.ErrorCodeValidationFailed, errors.CategoryPerformance, fmt.Sprintf("dependency cycle detected involving: %v", cycleNodes), nil)
	}

	dg.levels = levels
	dg.modified = false

	return levels, nil
}

// DetectCycles finds circular dependencies in the graph
func (dg *DependencyGraph) DetectCycles() [][]ObjectID {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	var cycles [][]ObjectID
	visited := make(map[ObjectID]bool)
	recursionStack := make(map[ObjectID]bool)

	var dfs func(ObjectID, []ObjectID) bool
	dfs = func(id ObjectID, path []ObjectID) bool {
		visited[id] = true
		recursionStack[id] = true
		path = append(path, id)

		node := dg.nodes[id]
		for _, dep := range node.Dependencies {
			if !visited[dep] {
				if dfs(dep, path) {
					return true
				}
			} else if recursionStack[dep] {
				// Found a cycle
				cycleStart := -1
				for i, pathNode := range path {
					if pathNode == dep {
						cycleStart = i
						break
					}
				}
				if cycleStart >= 0 {
					cycle := make([]ObjectID, len(path)-cycleStart)
					copy(cycle, path[cycleStart:])
					cycles = append(cycles, cycle)
				}
				return true
			}
		}

		recursionStack[id] = false
		return false
	}

	for id := range dg.nodes {
		if !visited[id] {
			dfs(id, make([]ObjectID, 0))
		}
	}

	return cycles
}

// GetReadyNodes returns nodes that are ready for processing
func (dg *DependencyGraph) GetReadyNodes() []ObjectID {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	var ready []ObjectID
	for id, node := range dg.nodes {
		if !node.Processed && !node.Processing {
			// Check if all dependencies are processed
			allDepsProcessed := true
			for _, dep := range node.Dependencies {
				if depNode := dg.nodes[dep]; depNode != nil && !depNode.Processed {
					allDepsProcessed = false
					break
				}
			}

			if allDepsProcessed {
				ready = append(ready, id)
			}
		}
	}

	// Sort by priority and level
	sort.Slice(ready, func(i, j int) bool {
		nodeI, nodeJ := dg.nodes[ready[i]], dg.nodes[ready[j]]
		if nodeI.Level != nodeJ.Level {
			return nodeI.Level < nodeJ.Level
		}
		return nodeI.Priority > nodeJ.Priority
	})

	return ready
}

// MarkProcessing marks a node as currently being processed
func (dg *DependencyGraph) MarkProcessing(id ObjectID) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if node := dg.nodes[id]; node != nil {
		node.Processing = true
	}
}

// MarkCompleted marks a node as completed
func (dg *DependencyGraph) MarkCompleted(id ObjectID) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	if node := dg.nodes[id]; node != nil {
		node.Processing = false
		node.Processed = true
	}
}

// GetNode returns a node by ID
func (dg *DependencyGraph) GetNode(id ObjectID) *DependencyNode {
	dg.mu.RLock()
	defer dg.mu.RUnlock()
	return dg.nodes[id]
}

// GetStats returns graph statistics
func (dg *DependencyGraph) GetStats() GraphStats {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	stats := GraphStats{
		TotalNodes: len(dg.nodes),
		Levels:     len(dg.levels),
	}

	for _, node := range dg.nodes {
		if node.Processed {
			stats.ProcessedNodes++
		} else if node.Processing {
			stats.ProcessingNodes++
		} else {
			stats.PendingNodes++
		}

		if len(node.Dependencies) > stats.MaxDependencies {
			stats.MaxDependencies = len(node.Dependencies)
		}
	}

	return stats
}

// GraphStats provides dependency graph statistics
type GraphStats struct {
	TotalNodes      int `json:"total_nodes"`
	ProcessedNodes  int `json:"processed_nodes"`
	ProcessingNodes int `json:"processing_nodes"`
	PendingNodes    int `json:"pending_nodes"`
	Levels          int `json:"levels"`
	MaxDependencies int `json:"max_dependencies"`
}

// DependencyProcessor interface for processing objects
type DependencyProcessor interface {
	Process(ctx context.Context, node *DependencyNode) error
}

// ResolutionStats tracks dependency resolution statistics
type ResolutionStats struct {
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	TotalNodes        int       `json:"total_nodes"`
	ProcessedNodes    int       `json:"processed_nodes"`
	FailedNodes       int       `json:"failed_nodes"`
	CyclesDetected    int       `json:"cycles_detected"`
	ParallelismFactor float64   `json:"parallelism_factor"`
}

// NewParallelDependencyResolver creates a new parallel dependency resolver
func NewParallelDependencyResolver(workerCount int, processor DependencyProcessor) *ParallelDependencyResolver {
	ctx, cancel := context.WithCancel(context.Background())

	return &ParallelDependencyResolver{
		graph:       NewDependencyGraph(),
		ready:       make(chan ObjectID, workerCount*2),
		processing:  make(chan ObjectID, workerCount*2),
		completed:   make(chan ObjectID, workerCount*2),
		errors:      make(chan error, workerCount),
		ctx:         ctx,
		cancel:      cancel,
		workerCount: workerCount,
		stats:       &ResolutionStats{},
		processor:   processor,
	}
}

// AddDependency adds a dependency relationship to the resolver
func (pdr *ParallelDependencyResolver) AddDependency(dependent, dependency ObjectID, data interface{}) {
	pdr.graph.AddNode(dependent, data)
	pdr.graph.AddNode(dependency, nil)
	pdr.graph.AddDependency(dependent, dependency)
}

// Resolve starts the parallel dependency resolution process
func (pdr *ParallelDependencyResolver) Resolve() error {
	pdr.stats.StartTime = time.Now()
	defer func() {
		pdr.stats.EndTime = time.Now()
	}()

	// Check for cycles first
	cycles := pdr.graph.DetectCycles()
	if len(cycles) > 0 {
		pdr.stats.CyclesDetected = len(cycles)
		return errors.New(errors.ErrorCodeValidationFailed, errors.CategoryPerformance, fmt.Sprintf("dependency cycles detected: %v", cycles), nil)
	}

	// Perform topological sort
	levels, err := pdr.graph.TopologicalSort()
	if err != nil {
		return err
	}

	// Start workers
	for i := 0; i < pdr.workerCount; i++ {
		pdr.wg.Add(1)
		go pdr.worker(i)
	}

	// Start result handler
	go pdr.resultHandler()

	// Start dependency tracker
	go pdr.dependencyTracker()

	// Process levels sequentially, but nodes within levels in parallel
	for levelIndex, level := range levels {
		fmt.Printf("Processing level %d with %d nodes\n", levelIndex, len(level))

		// Send all nodes in this level to workers
		for _, nodeID := range level {
			select {
			case pdr.ready <- nodeID:
			case <-pdr.ctx.Done():
				return pdr.ctx.Err()
			}
		}

		// Wait for all nodes in this level to complete
		for i := 0; i < len(level); i++ {
			select {
			case <-pdr.completed:
				pdr.stats.ProcessedNodes++
			case err := <-pdr.errors:
				pdr.stats.FailedNodes++
				return err
			case <-pdr.ctx.Done():
				return pdr.ctx.Err()
			}
		}
	}

	// Signal completion and wait for workers
	close(pdr.ready)
	pdr.wg.Wait()

	// Calculate parallelism factor
	totalTime := pdr.stats.EndTime.Sub(pdr.stats.StartTime).Seconds()
	if totalTime > 0 {
		pdr.stats.ParallelismFactor = float64(pdr.stats.ProcessedNodes) / (totalTime * float64(pdr.workerCount))
	}

	return nil
}

// worker processes dependencies
func (pdr *ParallelDependencyResolver) worker(id int) {
	defer pdr.wg.Done()

	for {
		select {
		case nodeID, ok := <-pdr.ready:
			if !ok {
				return // Channel closed
			}

			// Mark as processing
			pdr.graph.MarkProcessing(nodeID)
			pdr.processing <- nodeID

			// Process the node
			node := pdr.graph.GetNode(nodeID)
			if node != nil {
				err := pdr.processor.Process(pdr.ctx, node)
				if err != nil {
					pdr.errors <- errors.Wrap(err, errors.ErrorCodeValidationFailed, errors.CategoryPerformance, fmt.Sprintf("worker %d failed to process %s", id, nodeID), nil)
					continue
				}
			}

			// Mark as completed
			pdr.graph.MarkCompleted(nodeID)
			pdr.completed <- nodeID

		case <-pdr.ctx.Done():
			return
		}
	}
}

// resultHandler handles processing results
func (pdr *ParallelDependencyResolver) resultHandler() {
	for {
		select {
		case nodeID := <-pdr.processing:
			// Log or handle processing start
			_ = nodeID

		case <-pdr.ctx.Done():
			return
		}
	}
}

// dependencyTracker tracks dependency resolution progress
func (pdr *ParallelDependencyResolver) dependencyTracker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := pdr.graph.GetStats()
			fmt.Printf("Dependency resolution progress: %d/%d processed, %d processing, %d pending\n",
				stats.ProcessedNodes, stats.TotalNodes, stats.ProcessingNodes, stats.PendingNodes)

		case <-pdr.ctx.Done():
			return
		}
	}
}

// GetStats returns current resolution statistics
func (pdr *ParallelDependencyResolver) GetStats() ResolutionStats {
	graphStats := pdr.graph.GetStats()
	stats := *pdr.stats
	stats.TotalNodes = graphStats.TotalNodes
	return stats
}

// Stop gracefully stops the dependency resolver
func (pdr *ParallelDependencyResolver) Stop() {
	pdr.cancel()
	pdr.wg.Wait()
}

// SimpleDependencyProcessor is a basic implementation of DependencyProcessor
type SimpleDependencyProcessor struct {
	handler func(ctx context.Context, node *DependencyNode) error
}

// NewSimpleDependencyProcessor creates a simple dependency processor
func NewSimpleDependencyProcessor(handler func(ctx context.Context, node *DependencyNode) error) *SimpleDependencyProcessor {
	return &SimpleDependencyProcessor{handler: handler}
}

// Process processes a dependency node
func (sdp *SimpleDependencyProcessor) Process(ctx context.Context, node *DependencyNode) error {
	if sdp.handler != nil {
		return sdp.handler(ctx, node)
	}
	return nil
}
