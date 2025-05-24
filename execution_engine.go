package curator

import (
	"encoding/json"
	"fmt"
	"time"
)

// ExecutionEngine handles the execution of reorganization plans with WAL support
type ExecutionEngine struct {
	filesystem FileSystem
	store      OperationStore
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(filesystem FileSystem, store OperationStore) *ExecutionEngine {
	return &ExecutionEngine{
		filesystem: filesystem,
		store:      store,
	}
}

// ExecutePlan executes a reorganization plan with WAL and conflict handling
func (e *ExecutionEngine) ExecutePlan(planID string, failFast bool) (*ExecutionLog, error) {
	// Get the plan
	plan, err := e.store.GetPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	
	// Create execution log
	execLog := &ExecutionLog{
		PlanID:    planID,
		Timestamp: time.Now(),
		Status:    StatusInProgress,
		Completed: make([]CompletedMove, 0),
		Failed:    make([]FailedMove, 0),
		Skipped:   make([]SkippedMove, 0),
	}
	
	// Check for pending operations from previous interrupted execution
	pendingOps, err := e.store.GetPendingOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending operations: %w", err)
	}
	
	// If there are pending operations for this plan, resume from there
	if len(pendingOps) > 0 {
		fmt.Printf("Resuming execution from %d pending operations\n", len(pendingOps))
		return e.resumeExecution(planID, execLog, failFast)
	}
	
	// Start fresh execution - log all operations to WAL first
	for _, move := range plan.Moves {
		opData, err := json.Marshal(move)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal move operation: %w", err)
		}
		
		op := &Operation{
			ID:        fmt.Sprintf("%s-%s", planID, move.ID),
			Type:      string(move.Type),
			Data:      opData,
			Timestamp: time.Now(),
		}
		
		if err := e.store.LogOperation(op); err != nil {
			return nil, fmt.Errorf("failed to log operation to WAL: %w", err)
		}
	}
	
	// Execute operations
	return e.executeOperations(plan.Moves, execLog, failFast)
}

// resumeExecution resumes execution from pending operations
func (e *ExecutionEngine) resumeExecution(planID string, execLog *ExecutionLog, failFast bool) (*ExecutionLog, error) {
	pendingOps, err := e.store.GetPendingOperations()
	if err != nil {
		return nil, fmt.Errorf("failed to get pending operations: %w", err)
	}
	
	var moves []Move
	for _, op := range pendingOps {
		var move Move
		if err := json.Unmarshal(op.Data, &move); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pending operation: %w", err)
		}
		moves = append(moves, move)
	}
	
	fmt.Printf("Resuming execution of %d pending moves\n", len(moves))
	return e.executeOperations(moves, execLog, failFast)
}

// executeOperations executes a list of moves
func (e *ExecutionEngine) executeOperations(moves []Move, execLog *ExecutionLog, failFast bool) (*ExecutionLog, error) {
	for _, move := range moves {
		success, err := e.executeMove(move)
		opID := fmt.Sprintf("%s-%s", execLog.PlanID, move.ID)
		
		if success {
			execLog.Completed = append(execLog.Completed, CompletedMove{
				MoveID:    move.ID,
				Timestamp: time.Now(),
			})
			
			// Mark operation as complete in WAL
			if err := e.store.MarkOperationComplete(opID); err != nil {
				fmt.Printf("Warning: failed to mark operation complete in WAL: %v\n", err)
			}
			
		} else if err != nil {
			// Check if this is a conflict (file doesn't exist) or a real error
			isConflict := e.isConflictError(move, err)
			
			if isConflict {
				execLog.Skipped = append(execLog.Skipped, SkippedMove{
					MoveID:    move.ID,
					Timestamp: time.Now(),
					Reason:    fmt.Sprintf("Conflict: %v", err),
				})
				
				// Mark as complete since we're skipping it
				if err := e.store.MarkOperationComplete(opID); err != nil {
					fmt.Printf("Warning: failed to mark skipped operation complete in WAL: %v\n", err)
				}
				
			} else {
				execLog.Failed = append(execLog.Failed, FailedMove{
					MoveID:    move.ID,
					Timestamp: time.Now(),
					Error:     err.Error(),
				})
				
				if failFast {
					execLog.Status = StatusFailed
					e.store.SaveExecutionLog(execLog)
					return execLog, fmt.Errorf("execution failed on move %s: %w", move.ID, err)
				}
			}
		}
	}
	
	// Determine final status
	if len(execLog.Failed) > 0 {
		execLog.Status = StatusPartial
	} else {
		execLog.Status = StatusCompleted
	}
	
	// Save execution log
	if err := e.store.SaveExecutionLog(execLog); err != nil {
		return execLog, fmt.Errorf("failed to save execution log: %w", err)
	}
	
	return execLog, nil
}

// executeMove executes a single move operation
func (e *ExecutionEngine) executeMove(move Move) (bool, error) {
	switch move.Type {
	case CreateFolder:
		return e.executeCreateFolder(move)
	case FileMove:
		return e.executeFileMove(move)
	case FolderMove:
		return e.executeFolderMove(move)
	default:
		return false, fmt.Errorf("unknown move type: %s", move.Type)
	}
}

// executeCreateFolder creates a folder
func (e *ExecutionEngine) executeCreateFolder(move Move) (bool, error) {
	exists, err := e.filesystem.Exists(move.Destination)
	if err != nil {
		return false, fmt.Errorf("failed to check if folder exists: %w", err)
	}
	
	if exists {
		// Folder already exists, consider this successful
		return true, nil
	}
	
	if err := e.filesystem.CreateFolder(move.Destination); err != nil {
		return false, fmt.Errorf("failed to create folder %s: %w", move.Destination, err)
	}
	
	return true, nil
}

// executeFileMove moves a file
func (e *ExecutionEngine) executeFileMove(move Move) (bool, error) {
	// Check if source still exists (conflict detection)
	exists, err := e.filesystem.Exists(move.Source)
	if err != nil {
		return false, fmt.Errorf("failed to check if source exists: %w", err)
	}
	
	if !exists {
		return false, fmt.Errorf("source file no longer exists: %s", move.Source)
	}
	
	// Check if destination already exists
	destExists, err := e.filesystem.Exists(move.Destination)
	if err != nil {
		return false, fmt.Errorf("failed to check if destination exists: %w", err)
	}
	
	if destExists {
		return false, fmt.Errorf("destination already exists: %s", move.Destination)
	}
	
	if err := e.filesystem.Move(move.Source, move.Destination); err != nil {
		return false, fmt.Errorf("failed to move file from %s to %s: %w", move.Source, move.Destination, err)
	}
	
	return true, nil
}

// executeFolderMove moves a folder and its contents
func (e *ExecutionEngine) executeFolderMove(move Move) (bool, error) {
	// Check if source still exists (conflict detection)
	exists, err := e.filesystem.Exists(move.Source)
	if err != nil {
		return false, fmt.Errorf("failed to check if source exists: %w", err)
	}
	
	if !exists {
		return false, fmt.Errorf("source folder no longer exists: %s", move.Source)
	}
	
	// Check if destination already exists
	destExists, err := e.filesystem.Exists(move.Destination)
	if err != nil {
		return false, fmt.Errorf("failed to check if destination exists: %w", err)
	}
	
	if destExists {
		return false, fmt.Errorf("destination already exists: %s", move.Destination)
	}
	
	if err := e.filesystem.Move(move.Source, move.Destination); err != nil {
		return false, fmt.Errorf("failed to move folder from %s to %s: %w", move.Source, move.Destination, err)
	}
	
	return true, nil
}

// isConflictError determines if an error is due to a conflict (file moved/deleted by user)
func (e *ExecutionEngine) isConflictError(move Move, err error) bool {
	errStr := err.Error()
	
	// Check for common conflict patterns
	conflictPatterns := []string{
		"no longer exists",
		"not found",
		"already exists",
		"source file not found",
		"destination already exists",
	}
	
	for _, pattern := range conflictPatterns {
		if fmt.Sprintf("%v", errStr) != "" && len(errStr) > 0 {
			// Simple string contains check for conflict detection
			if len(pattern) > 0 && len(errStr) >= len(pattern) {
				for i := 0; i <= len(errStr)-len(pattern); i++ {
					if errStr[i:i+len(pattern)] == pattern {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// GetExecutionStatus returns the current status of plan execution
func (e *ExecutionEngine) GetExecutionStatus(planID string) (*ExecutionLog, error) {
	logs, err := e.store.GetExecutionHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}
	
	// Find the most recent execution log for this plan
	for _, log := range logs {
		if log.PlanID == planID {
			return log, nil
		}
	}
	
	return nil, fmt.Errorf("no execution log found for plan: %s", planID)
}