package curator

import (
	"testing"
	"time"
)

func TestMemoryOperationStore_SaveAndGetPlan(t *testing.T) {
	store := NewMemoryOperationStore()
	
	// Create a test plan
	plan := &ReorganizationPlan{
		ID:        "test-plan-001",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "/old/path.txt",
				Destination: "/new/path.txt",
				Reason:      "Better organization",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FoldersCreated:          1,
			FilesMoved:              1,
			FoldersMovedDeduplicated: 0,
			DepthReduction:          "10%",
			OrganizationImprovement: "Improved",
		},
		Rationale: "Test reorganization",
	}
	
	// Save the plan
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	// Retrieve the plan
	retrievedPlan, err := store.GetPlan("test-plan-001")
	if err != nil {
		t.Fatalf("Failed to get plan: %v", err)
	}
	
	// Verify the plan data
	if retrievedPlan.ID != plan.ID {
		t.Errorf("Expected ID %s, got %s", plan.ID, retrievedPlan.ID)
	}
	
	if len(retrievedPlan.Moves) != len(plan.Moves) {
		t.Errorf("Expected %d moves, got %d", len(plan.Moves), len(retrievedPlan.Moves))
	}
	
	if retrievedPlan.Moves[0].Source != plan.Moves[0].Source {
		t.Errorf("Expected source %s, got %s", plan.Moves[0].Source, retrievedPlan.Moves[0].Source)
	}
	
	// Test that modifications to retrieved plan don't affect stored plan
	retrievedPlan.Moves[0].Source = "modified"
	
	secondRetrieval, err := store.GetPlan("test-plan-001")
	if err != nil {
		t.Fatalf("Failed to get plan second time: %v", err)
	}
	
	if secondRetrieval.Moves[0].Source != "/old/path.txt" {
		t.Error("Stored plan should not be affected by modifications to retrieved copy")
	}
}

func TestMemoryOperationStore_ListPlans(t *testing.T) {
	store := NewMemoryOperationStore()
	
	// Create test plans with different timestamps
	now := time.Now()
	plan1 := &ReorganizationPlan{
		ID:        "plan-1",
		Timestamp: now.Add(-time.Hour),
		Moves:     []Move{{ID: "move-1", Type: FileMove}},
		Summary:   Summary{},
		Rationale: "First plan",
	}
	
	plan2 := &ReorganizationPlan{
		ID:        "plan-2",
		Timestamp: now,
		Moves:     []Move{{ID: "move-2", Type: FileMove}, {ID: "move-3", Type: FolderMove}},
		Summary:   Summary{},
		Rationale: "Second plan",
	}
	
	// Save plans
	err := store.SavePlan(plan1)
	if err != nil {
		t.Fatalf("Failed to save plan1: %v", err)
	}
	
	err = store.SavePlan(plan2)
	if err != nil {
		t.Fatalf("Failed to save plan2: %v", err)
	}
	
	// List plans
	summaries, err := store.ListPlans()
	if err != nil {
		t.Fatalf("Failed to list plans: %v", err)
	}
	
	if len(summaries) != 2 {
		t.Fatalf("Expected 2 plans, got %d", len(summaries))
	}
	
	// Plans should be ordered by timestamp DESC (newest first)
	if summaries[0].ID != "plan-2" {
		t.Errorf("Expected plan-2 first, got %s", summaries[0].ID)
	}
	
	if summaries[1].ID != "plan-1" {
		t.Errorf("Expected plan-1 second, got %s", summaries[1].ID)
	}
	
	// Check move counts
	if summaries[0].MoveCount != 2 {
		t.Errorf("Expected 2 moves for plan-2, got %d", summaries[0].MoveCount)
	}
	
	if summaries[1].MoveCount != 1 {
		t.Errorf("Expected 1 move for plan-1, got %d", summaries[1].MoveCount)
	}
}

func TestMemoryOperationStore_Operations(t *testing.T) {
	store := NewMemoryOperationStore()
	
	// Create test operations
	op1 := &Operation{
		ID:        "op-1",
		Type:      "move",
		Data:      []byte("test data 1"),
		Timestamp: time.Now().Add(-time.Minute),
	}
	
	op2 := &Operation{
		ID:        "op-2",
		Type:      "create",
		Data:      []byte("test data 2"),
		Timestamp: time.Now(),
	}
	
	// Log the operations
	err := store.LogOperation(op1)
	if err != nil {
		t.Fatalf("Failed to log operation 1: %v", err)
	}
	
	err = store.LogOperation(op2)
	if err != nil {
		t.Fatalf("Failed to log operation 2: %v", err)
	}
	
	// Get pending operations
	pending, err := store.GetPendingOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	
	if len(pending) != 2 {
		t.Fatalf("Expected 2 pending operations, got %d", len(pending))
	}
	
	// Should be ordered by timestamp (oldest first)
	if pending[0].ID != "op-1" {
		t.Errorf("Expected op-1 first, got %s", pending[0].ID)
	}
	
	if pending[1].ID != "op-2" {
		t.Errorf("Expected op-2 second, got %s", pending[1].ID)
	}
	
	// Mark first operation complete
	err = store.MarkOperationComplete("op-1")
	if err != nil {
		t.Fatalf("Failed to mark operation complete: %v", err)
	}
	
	// Check that only one operation is pending
	pending, err = store.GetPendingOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	
	if len(pending) != 1 {
		t.Fatalf("Expected 1 pending operation, got %d", len(pending))
	}
	
	if pending[0].ID != "op-2" {
		t.Errorf("Expected op-2 to still be pending, got %s", pending[0].ID)
	}
	
	// Mark second operation complete
	err = store.MarkOperationComplete("op-2")
	if err != nil {
		t.Fatalf("Failed to mark operation 2 complete: %v", err)
	}
	
	// Check that no operations are pending
	pending, err = store.GetPendingOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	
	if len(pending) != 0 {
		t.Fatalf("Expected 0 pending operations, got %d", len(pending))
	}
}

func TestMemoryOperationStore_ExecutionLog(t *testing.T) {
	store := NewMemoryOperationStore()
	
	// Create test execution logs
	log1 := &ExecutionLog{
		PlanID:    "plan-1",
		Timestamp: time.Now().Add(-time.Hour),
		Status:    StatusCompleted,
		Completed: []CompletedMove{
			{MoveID: "move-1", Timestamp: time.Now()},
		},
		Failed:  []FailedMove{},
		Skipped: []SkippedMove{},
	}
	
	log2 := &ExecutionLog{
		PlanID:    "plan-2",
		Timestamp: time.Now(),
		Status:    StatusPartial,
		Completed: []CompletedMove{
			{MoveID: "move-2", Timestamp: time.Now()},
		},
		Failed: []FailedMove{
			{MoveID: "move-3", Timestamp: time.Now(), Error: "File not found"},
		},
		Skipped: []SkippedMove{},
	}
	
	// Save the execution logs
	err := store.SaveExecutionLog(log1)
	if err != nil {
		t.Fatalf("Failed to save execution log 1: %v", err)
	}
	
	err = store.SaveExecutionLog(log2)
	if err != nil {
		t.Fatalf("Failed to save execution log 2: %v", err)
	}
	
	// Get execution history
	history, err := store.GetExecutionHistory()
	if err != nil {
		t.Fatalf("Failed to get execution history: %v", err)
	}
	
	if len(history) != 2 {
		t.Fatalf("Expected 2 execution logs, got %d", len(history))
	}
	
	// Should be ordered by timestamp DESC (newest first)
	if history[0].PlanID != "plan-2" {
		t.Errorf("Expected plan-2 first, got %s", history[0].PlanID)
	}
	
	if history[1].PlanID != "plan-1" {
		t.Errorf("Expected plan-1 second, got %s", history[1].PlanID)
	}
	
	// Verify data integrity
	if history[0].Status != StatusPartial {
		t.Errorf("Expected status %s, got %s", StatusPartial, history[0].Status)
	}
	
	if len(history[0].Failed) != 1 {
		t.Errorf("Expected 1 failed move, got %d", len(history[0].Failed))
	}
	
	if history[0].Failed[0].Error != "File not found" {
		t.Errorf("Expected error 'File not found', got %s", history[0].Failed[0].Error)
	}
}

func TestMemoryOperationStore_GetNonExistentPlan(t *testing.T) {
	store := NewMemoryOperationStore()
	
	_, err := store.GetPlan("non-existent")
	if err == nil {
		t.Error("Expected error when getting non-existent plan")
	}
}

func TestMemoryOperationStore_MarkNonExistentOperationComplete(t *testing.T) {
	store := NewMemoryOperationStore()
	
	err := store.MarkOperationComplete("non-existent")
	if err == nil {
		t.Error("Expected error when marking non-existent operation complete")
	}
}

func TestMemoryOperationStore_Clear(t *testing.T) {
	store := NewMemoryOperationStore()
	
	// Add some data
	plan := &ReorganizationPlan{ID: "test", Timestamp: time.Now()}
	store.SavePlan(plan)
	
	op := &Operation{ID: "test", Type: "test", Timestamp: time.Now()}
	store.LogOperation(op)
	
	log := &ExecutionLog{PlanID: "test", Timestamp: time.Now(), Status: StatusCompleted}
	store.SaveExecutionLog(log)
	
	// Clear the store
	store.Clear()
	
	// Verify everything is cleared
	plans, _ := store.ListPlans()
	if len(plans) != 0 {
		t.Error("Plans should be cleared")
	}
	
	pending, _ := store.GetPendingOperations()
	if len(pending) != 0 {
		t.Error("Operations should be cleared")
	}
	
	history, _ := store.GetExecutionHistory()
	if len(history) != 0 {
		t.Error("Execution history should be cleared")
	}
}