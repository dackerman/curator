package curator

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryOperationStore implements OperationStore interface using in-memory storage
type MemoryOperationStore struct {
	mu         sync.RWMutex
	plans      map[string]*ReorganizationPlan
	operations map[string]*Operation
	execLogs   []*ExecutionLog
}

// NewMemoryOperationStore creates a new in-memory operation store
func NewMemoryOperationStore() *MemoryOperationStore {
	return &MemoryOperationStore{
		plans:      make(map[string]*ReorganizationPlan),
		operations: make(map[string]*Operation),
		execLogs:   make([]*ExecutionLog, 0),
	}
}

// SavePlan implements OperationStore.SavePlan
func (m *MemoryOperationStore) SavePlan(plan *ReorganizationPlan) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create a deep copy to avoid external modifications
	planCopy := *plan
	planCopy.Moves = make([]Move, len(plan.Moves))
	copy(planCopy.Moves, plan.Moves)
	
	m.plans[plan.ID] = &planCopy
	return nil
}

// GetPlan implements OperationStore.GetPlan
func (m *MemoryOperationStore) GetPlan(id string) (*ReorganizationPlan, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	plan, exists := m.plans[id]
	if !exists {
		return nil, fmt.Errorf("plan not found: %s", id)
	}
	
	// Return a copy to avoid external modifications
	planCopy := *plan
	planCopy.Moves = make([]Move, len(plan.Moves))
	copy(planCopy.Moves, plan.Moves)
	
	return &planCopy, nil
}

// ListPlans implements OperationStore.ListPlans
func (m *MemoryOperationStore) ListPlans() ([]*PlanSummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	summaries := make([]*PlanSummary, 0, len(m.plans))
	
	for _, plan := range m.plans {
		summary := &PlanSummary{
			ID:        plan.ID,
			Timestamp: plan.Timestamp,
			Status:    "pending", // Default status
			FileCount: len(plan.Moves),
			MoveCount: len(plan.Moves),
		}
		summaries = append(summaries, summary)
	}
	
	// Sort by timestamp descending (newest first)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Timestamp.After(summaries[j].Timestamp)
	})
	
	return summaries, nil
}

// LogOperation implements OperationStore.LogOperation
func (m *MemoryOperationStore) LogOperation(op *Operation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create a copy to avoid external modifications
	opCopy := *op
	opCopy.Data = make([]byte, len(op.Data))
	copy(opCopy.Data, op.Data)
	
	m.operations[op.ID] = &opCopy
	return nil
}

// GetPendingOperations implements OperationStore.GetPendingOperations
func (m *MemoryOperationStore) GetPendingOperations() ([]*Operation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var pending []*Operation
	for _, op := range m.operations {
		// Skip completion markers and check if operation is still pending
		if op.Type == "completion_marker" {
			continue
		}
		if _, exists := m.operations[op.ID+"_completed"]; !exists {
			// Return a copy
			opCopy := *op
			opCopy.Data = make([]byte, len(op.Data))
			copy(opCopy.Data, op.Data)
			pending = append(pending, &opCopy)
		}
	}
	
	// Sort by timestamp
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Timestamp.Before(pending[j].Timestamp)
	})
	
	return pending, nil
}

// MarkOperationComplete implements OperationStore.MarkOperationComplete
func (m *MemoryOperationStore) MarkOperationComplete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.operations[id]; !exists {
		return fmt.Errorf("operation not found: %s", id)
	}
	
	// Mark as completed by adding a completion marker
	m.operations[id+"_completed"] = &Operation{
		ID:        id + "_completed",
		Type:      "completion_marker",
		Timestamp: time.Now(),
	}
	
	return nil
}

// SaveExecutionLog implements OperationStore.SaveExecutionLog
func (m *MemoryOperationStore) SaveExecutionLog(log *ExecutionLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Create a deep copy
	logCopy := *log
	logCopy.Completed = make([]CompletedMove, len(log.Completed))
	copy(logCopy.Completed, log.Completed)
	logCopy.Failed = make([]FailedMove, len(log.Failed))
	copy(logCopy.Failed, log.Failed)
	logCopy.Skipped = make([]SkippedMove, len(log.Skipped))
	copy(logCopy.Skipped, log.Skipped)
	
	m.execLogs = append(m.execLogs, &logCopy)
	return nil
}

// GetExecutionHistory implements OperationStore.GetExecutionHistory
func (m *MemoryOperationStore) GetExecutionHistory() ([]*ExecutionLog, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Create copies and sort by timestamp descending
	logs := make([]*ExecutionLog, len(m.execLogs))
	for i, log := range m.execLogs {
		logCopy := *log
		logCopy.Completed = make([]CompletedMove, len(log.Completed))
		copy(logCopy.Completed, log.Completed)
		logCopy.Failed = make([]FailedMove, len(log.Failed))
		copy(logCopy.Failed, log.Failed)
		logCopy.Skipped = make([]SkippedMove, len(log.Skipped))
		copy(logCopy.Skipped, log.Skipped)
		logs[i] = &logCopy
	}
	
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].Timestamp.After(logs[j].Timestamp)
	})
	
	return logs, nil
}

// Clear removes all stored data (useful for testing)
func (m *MemoryOperationStore) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.plans = make(map[string]*ReorganizationPlan)
	m.operations = make(map[string]*Operation)
	m.execLogs = make([]*ExecutionLog, 0)
}