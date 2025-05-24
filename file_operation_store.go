package curator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileOperationStore implements OperationStore interface using JSON files
type FileOperationStore struct {
	mu       sync.RWMutex
	storeDir string
}

// NewFileOperationStore creates a new file-based operation store
func NewFileOperationStore(storeDir string) (*FileOperationStore, error) {
	// Create store directory if it doesn't exist
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{"plans", "operations", "execution_logs"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(storeDir, subdir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create subdirectory %s: %w", subdir, err)
		}
	}

	return &FileOperationStore{
		storeDir: storeDir,
	}, nil
}

// SavePlan implements OperationStore.SavePlan
func (f *FileOperationStore) SavePlan(plan *ReorganizationPlan) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	planPath := filepath.Join(f.storeDir, "plans", plan.ID+".json")
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal plan: %w", err)
	}

	if err := os.WriteFile(planPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	return nil
}

// GetPlan implements OperationStore.GetPlan
func (f *FileOperationStore) GetPlan(id string) (*ReorganizationPlan, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	planPath := filepath.Join(f.storeDir, "plans", id+".json")
	data, err := os.ReadFile(planPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("plan not found: %s", id)
		}
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var plan ReorganizationPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan: %w", err)
	}

	return &plan, nil
}

// ListPlans implements OperationStore.ListPlans
func (f *FileOperationStore) ListPlans() ([]*PlanSummary, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	plansDir := filepath.Join(f.storeDir, "plans")
	entries, err := os.ReadDir(plansDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read plans directory: %w", err)
	}

	var summaries []*PlanSummary
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			planID := strings.TrimSuffix(entry.Name(), ".json")
			
			// Read the plan to get details
			plan, err := f.GetPlan(planID)
			if err != nil {
				continue // Skip corrupted files
			}

			summary := &PlanSummary{
				ID:        plan.ID,
				Timestamp: plan.Timestamp,
				Status:    "pending", // Default status
				FileCount: len(plan.Moves),
				MoveCount: len(plan.Moves),
			}
			summaries = append(summaries, summary)
		}
	}

	// Sort by timestamp descending (newest first)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Timestamp.After(summaries[j].Timestamp)
	})

	return summaries, nil
}

// LogOperation implements OperationStore.LogOperation
func (f *FileOperationStore) LogOperation(op *Operation) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	opPath := filepath.Join(f.storeDir, "operations", op.ID+".json")
	data, err := json.MarshalIndent(op, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}

	if err := os.WriteFile(opPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write operation file: %w", err)
	}

	return nil
}

// GetPendingOperations implements OperationStore.GetPendingOperations
func (f *FileOperationStore) GetPendingOperations() ([]*Operation, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	opsDir := filepath.Join(f.storeDir, "operations")
	entries, err := os.ReadDir(opsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read operations directory: %w", err)
	}

	var pending []*Operation
	completed := make(map[string]bool)

	// First pass: identify completed operations
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			opID := strings.TrimSuffix(entry.Name(), ".json")
			if strings.HasSuffix(opID, "_completed") {
				originalID := strings.TrimSuffix(opID, "_completed")
				completed[originalID] = true
			}
		}
	}

	// Second pass: collect pending operations
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			opID := strings.TrimSuffix(entry.Name(), ".json")
			
			// Skip completion markers and already completed operations
			if strings.HasSuffix(opID, "_completed") || completed[opID] {
				continue
			}

			opPath := filepath.Join(opsDir, entry.Name())
			data, err := os.ReadFile(opPath)
			if err != nil {
				continue // Skip corrupted files
			}

			var op Operation
			if err := json.Unmarshal(data, &op); err != nil {
				continue // Skip corrupted files
			}

			pending = append(pending, &op)
		}
	}

	// Sort by timestamp
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Timestamp.Before(pending[j].Timestamp)
	})

	return pending, nil
}

// MarkOperationComplete implements OperationStore.MarkOperationComplete
func (f *FileOperationStore) MarkOperationComplete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if operation exists
	opPath := filepath.Join(f.storeDir, "operations", id+".json")
	if _, err := os.Stat(opPath); os.IsNotExist(err) {
		return fmt.Errorf("operation not found: %s", id)
	}

	// Create completion marker
	completionOp := &Operation{
		ID:        id + "_completed",
		Type:      "completion_marker",
		Timestamp: time.Now(),
	}

	completionPath := filepath.Join(f.storeDir, "operations", completionOp.ID+".json")
	data, err := json.MarshalIndent(completionOp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal completion marker: %w", err)
	}

	if err := os.WriteFile(completionPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write completion marker: %w", err)
	}

	return nil
}

// SaveExecutionLog implements OperationStore.SaveExecutionLog
func (f *FileOperationStore) SaveExecutionLog(log *ExecutionLog) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	logPath := filepath.Join(f.storeDir, "execution_logs", log.PlanID+".json")
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal execution log: %w", err)
	}

	if err := os.WriteFile(logPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write execution log file: %w", err)
	}

	return nil
}

// GetExecutionHistory implements OperationStore.GetExecutionHistory
func (f *FileOperationStore) GetExecutionHistory() ([]*ExecutionLog, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	logsDir := filepath.Join(f.storeDir, "execution_logs")
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read execution logs directory: %w", err)
	}

	var logs []*ExecutionLog
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			logPath := filepath.Join(logsDir, entry.Name())
			data, err := os.ReadFile(logPath)
			if err != nil {
				continue // Skip corrupted files
			}

			var log ExecutionLog
			if err := json.Unmarshal(data, &log); err != nil {
				continue // Skip corrupted files
			}

			logs = append(logs, &log)
		}
	}

	// Sort by timestamp descending (newest first)
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.After(logs[j].Timestamp)
	})

	return logs, nil
}

// Clear removes all stored data (useful for testing)
func (f *FileOperationStore) Clear() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	subdirs := []string{"plans", "operations", "execution_logs"}
	for _, subdir := range subdirs {
		dirPath := filepath.Join(f.storeDir, subdir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				if err := os.Remove(filepath.Join(dirPath, entry.Name())); err != nil {
					return fmt.Errorf("failed to remove file %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return nil
}