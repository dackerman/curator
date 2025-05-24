package curator

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"
)

// ExecutionEngine handles executing reorganization plans with WAL support
type ExecutionEngine struct {
	fs    FileSystem
	store OperationStore
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(fs FileSystem, store OperationStore) *ExecutionEngine {
	return &ExecutionEngine{
		fs:    fs,
		store: store,
	}
}

// ExecutePlan executes a reorganization plan with full WAL support and conflict handling
func (e *ExecutionEngine) ExecutePlan(planID string, failFast bool) (*ExecutionLog, error) {
	// Get the plan
	plan, err := e.store.GetPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	
	// Initialize execution log
	execLog := &ExecutionLog{
		PlanID:    planID,
		Timestamp: time.Now(),
		Status:    StatusInProgress,
		Completed: make([]CompletedMove, 0),
		Failed:    make([]FailedMove, 0),
		Skipped:   make([]SkippedMove, 0),
	}
	
	// Save initial execution log
	if err := e.store.SaveExecutionLog(execLog); err != nil {
		return nil, fmt.Errorf("failed to save initial execution log: %w", err)
	}
	
	// Execute moves in order
	for _, move := range plan.Moves {
		// Log operation to WAL before executing
		opData, err := json.Marshal(move)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal move data: %w", err)
		}
		
		operation := &Operation{
			ID:        fmt.Sprintf("%s-%s", planID, move.ID),
			Type:      "move",
			Data:      opData,
			Timestamp: time.Now(),
		}
		
		if err := e.store.LogOperation(operation); err != nil {
			return nil, fmt.Errorf("failed to log operation to WAL: %w", err)
		}
		
		// Execute the move
		err = e.executeMove(move)
		if err != nil {
			// Check if this is a conflict (file doesn't exist or destination exists)
			if isConflictError(err) {
				// Skip this move and continue
				execLog.Skipped = append(execLog.Skipped, SkippedMove{
					MoveID:    move.ID,
					Timestamp: time.Now(),
					Reason:    fmt.Sprintf("Conflict: %s", err.Error()),
				})
			} else {
				// Real error - mark as failed
				execLog.Failed = append(execLog.Failed, FailedMove{
					MoveID:    move.ID,
					Timestamp: time.Now(),
					Error:     err.Error(),
				})
				
				if failFast {
					execLog.Status = StatusFailed
					e.store.SaveExecutionLog(execLog)
					return execLog, fmt.Errorf("execution failed (fail-fast enabled): %w", err)
				}
			}
		} else {
			// Move succeeded - mark as completed
			execLog.Completed = append(execLog.Completed, CompletedMove{
				MoveID:    move.ID,
				Timestamp: time.Now(),
			})
		}
		
		// Mark operation as complete in WAL
		if err := e.store.MarkOperationComplete(operation.ID); err != nil {
			return nil, fmt.Errorf("failed to mark operation complete: %w", err)
		}
		
		// Update execution log
		if err := e.store.SaveExecutionLog(execLog); err != nil {
			return nil, fmt.Errorf("failed to update execution log: %w", err)
		}
	}
	
	// Determine final status
	if len(execLog.Failed) > 0 {
		if len(execLog.Completed) > 0 {
			execLog.Status = StatusPartial
		} else {
			execLog.Status = StatusFailed
		}
	} else if len(execLog.Skipped) > 0 {
		// If no failures but some operations were skipped, it's partial
		execLog.Status = StatusPartial
	} else {
		execLog.Status = StatusCompleted
	}
	
	// Save final execution log
	if err := e.store.SaveExecutionLog(execLog); err != nil {
		return nil, fmt.Errorf("failed to save final execution log: %w", err)
	}
	
	return execLog, nil
}

// executeMove executes a single move operation
func (e *ExecutionEngine) executeMove(move Move) error {
	switch move.Type {
	case CreateFolder:
		return e.fs.CreateFolder(move.Destination)
		
	case FileMove:
		// Check if source still exists
		exists, err := e.fs.Exists(move.Source)
		if err != nil {
			return fmt.Errorf("failed to check if source exists: %w", err)
		}
		if !exists {
			return &ConflictError{Message: fmt.Sprintf("source file no longer exists: %s", move.Source)}
		}
		
		// Check if destination already exists
		destExists, err := e.fs.Exists(move.Destination)
		if err != nil {
			return fmt.Errorf("failed to check if destination exists: %w", err)
		}
		if destExists {
			return &ConflictError{Message: fmt.Sprintf("destination already exists: %s", move.Destination)}
		}
		
		// Ensure destination directory exists
		destDir := filepath.Dir(move.Destination)
		if err := e.fs.CreateFolder(destDir); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
		
		// Execute the move
		return e.fs.Move(move.Source, move.Destination)
		
	case FolderMove:
		// Check if source folder still exists
		exists, err := e.fs.Exists(move.Source)
		if err != nil {
			return fmt.Errorf("failed to check if source folder exists: %w", err)
		}
		if !exists {
			return &ConflictError{Message: fmt.Sprintf("source folder no longer exists: %s", move.Source)}
		}
		
		// Check if destination already exists
		destExists, err := e.fs.Exists(move.Destination)
		if err != nil {
			return fmt.Errorf("failed to check if destination exists: %w", err)
		}
		if destExists {
			return &ConflictError{Message: fmt.Sprintf("destination already exists: %s", move.Destination)}
		}
		
		// Ensure parent of destination directory exists
		destParent := filepath.Dir(move.Destination)
		if err := e.fs.CreateFolder(destParent); err != nil {
			return fmt.Errorf("failed to create destination parent directory: %w", err)
		}
		
		// Execute the folder move
		return e.fs.Move(move.Source, move.Destination)
		
	default:
		return fmt.Errorf("unknown move type: %s", move.Type)
	}
}

// ResumePendingOperations resumes any pending operations from WAL after a crash
func (e *ExecutionEngine) ResumePendingOperations() error {
	pending, err := e.store.GetPendingOperations()
	if err != nil {
		return fmt.Errorf("failed to get pending operations: %w", err)
	}
	
	if len(pending) == 0 {
		return nil // Nothing to resume
	}
	
	fmt.Printf("Found %d pending operations to resume\n", len(pending))
	
	for _, op := range pending {
		if op.Type == "move" {
			var move Move
			if err := json.Unmarshal(op.Data, &move); err != nil {
				fmt.Printf("Failed to unmarshal move data for operation %s: %v\n", op.ID, err)
				continue
			}
			
			// Try to execute the move
			err := e.executeMove(move)
			if err != nil {
				fmt.Printf("Failed to resume move operation %s: %v\n", op.ID, err)
			} else {
				fmt.Printf("Successfully resumed move operation %s\n", op.ID)
			}
			
			// Mark as complete regardless of success/failure
			if err := e.store.MarkOperationComplete(op.ID); err != nil {
				fmt.Printf("Failed to mark operation %s as complete: %v\n", op.ID, err)
			}
		}
	}
	
	return nil
}

// ConflictError represents a conflict during execution
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

// isConflictError checks if an error is a conflict error
func isConflictError(err error) bool {
	_, ok := err.(*ConflictError)
	return ok
}

// GetExecutionStatus returns the current status of a plan execution
func (e *ExecutionEngine) GetExecutionStatus(planID string) (*ExecutionLog, error) {
	history, err := e.store.GetExecutionHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}
	
	// Find the most recent execution log for this plan
	// History is sorted newest first, so find the first match with final status
	for _, log := range history {
		if log.PlanID == planID {
			// Return logs that are not in progress (final state)
			if log.Status != StatusInProgress {
				return log, nil
			}
		}
	}
	
	// If no completed logs found, return the most recent one (could be in progress)
	for _, log := range history {
		if log.PlanID == planID {
			return log, nil
		}
	}
	
	return nil, fmt.Errorf("no execution found for plan: %s", planID)
}