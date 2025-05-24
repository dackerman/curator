package curator

import (
	"strings"
	"testing"
)

// TestApplyCommand_NilExecutionEngine tests the bug where apply command
// fails with nil pointer dereference because ExecutionEngine is not initialized
func TestApplyCommand_NilExecutionEngine(t *testing.T) {
	// Create a minimal setup to simulate the CLI behavior
	store := NewMemoryOperationStore()
	
	// Create a sample plan first
	plan := &ReorganizationPlan{
		ID:        "test-plan-123",
		Moves:     []Move{},
		Summary:   Summary{},
		Rationale: "Test plan",
	}
	
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save test plan: %v", err)
	}
	
	// This should reproduce the bug: creating ExecutionEngine with nil filesystem
	// The bug occurs when engine is nil and ResumePendingOperations is called
	var engine *ExecutionEngine // This simulates the nil engine in main.go
	
	// This should panic with nil pointer dereference, just like in the CLI
	defer func() {
		if r := recover(); r != nil {
			// We expect a panic - this confirms the bug exists
			if strings.Contains(r.(error).Error(), "invalid memory address") ||
			   strings.Contains(r.(error).Error(), "nil pointer") {
				// This is the expected panic - the bug is confirmed
				t.Logf("Bug confirmed: got expected nil pointer panic: %v", r)
				return
			}
			// If it's a different panic, re-panic
			panic(r)
		}
		// If we get here without a panic, the bug wasn't reproduced
		t.Error("Expected nil pointer panic but didn't get one")
	}()
	
	// This line should panic with nil pointer dereference
	_ = engine.ResumePendingOperations()
}

// TestApplyCommand_ProperInitialization tests that ExecutionEngine works correctly
// when properly initialized with filesystem and store
func TestApplyCommand_ProperInitialization(t *testing.T) {
	// Create proper components
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	
	// Create ExecutionEngine properly
	engine := NewExecutionEngine(fs, store)
	
	// This should NOT panic
	err := engine.ResumePendingOperations()
	if err != nil {
		t.Errorf("Expected no error from ResumePendingOperations, got: %v", err)
	}
	
	// Create a test plan
	plan := &ReorganizationPlan{
		ID:        "test-plan-456",
		Moves:     []Move{},
		Summary:   Summary{},
		Rationale: "Test plan",
	}
	
	err = store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save test plan: %v", err)
	}
	
	// This should also work without panic
	execLog, err := engine.ExecutePlan("test-plan-456", false)
	if err != nil {
		t.Errorf("Expected no error from ExecutePlan, got: %v", err)
	}
	
	if execLog == nil {
		t.Error("Expected execution log but got nil")
	}
	
	if execLog.PlanID != "test-plan-456" {
		t.Errorf("Expected plan ID 'test-plan-456', got '%s'", execLog.PlanID)
	}
}

// TestApplyCommand_WorkflowIntegration tests the complete workflow:
// create filesystem, create plan, execute plan, check status
func TestApplyCommand_WorkflowIntegration(t *testing.T) {
	// Setup components like the CLI does
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	
	// Add sample files
	fs.AddFile("/test.txt", []byte("test content"), "text/plain")
	fs.CreateFolder("/Documents")
	
	// Create a plan with actual moves
	plan := &ReorganizationPlan{
		ID:        "workflow-test-789",
		Moves: []Move{
			{
				ID:          "move-1",
				Type:        FileMove,
				Source:      "/test.txt",
				Destination: "/Documents/test.txt",
				Reason:      "Move to Documents folder",
			},
		},
		Summary:   Summary{FilesMoved: 1, FoldersCreated: 0},
		Rationale: "Test workflow",
	}
	
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save test plan: %v", err)
	}
	
	// Create execution engine (like the fixed apply command does)
	engine := NewExecutionEngine(fs, store)
	
	// Execute the plan
	execLog, err := engine.ExecutePlan("workflow-test-789", false)
	if err != nil {
		t.Fatalf("Failed to execute plan: %v", err)
	}
	
	// Verify execution results
	if execLog.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, execLog.Status)
	}
	
	if len(execLog.Completed) != 1 {
		t.Errorf("Expected 1 completed operation, got %d", len(execLog.Completed))
	}
	
	if len(execLog.Failed) != 0 {
		t.Errorf("Expected 0 failed operations, got %d", len(execLog.Failed))
	}
	
	// Verify the file was actually moved
	exists, err := fs.Exists("/test.txt")
	if err != nil {
		t.Fatalf("Error checking original file existence: %v", err)
	}
	if exists {
		t.Error("Original file should no longer exist after move")
	}
	
	exists, err = fs.Exists("/Documents/test.txt")
	if err != nil {
		t.Fatalf("Error checking moved file existence: %v", err)
	}
	if !exists {
		t.Error("Moved file should exist in new location")
	}
	
	// Test status retrieval (like the fixed status command does)
	statusLog, err := engine.GetExecutionStatus("workflow-test-789")
	if err != nil {
		t.Fatalf("Failed to get execution status: %v", err)
	}
	
	if statusLog.PlanID != "workflow-test-789" {
		t.Errorf("Expected plan ID 'workflow-test-789', got '%s'", statusLog.PlanID)
	}
	
	if statusLog.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, statusLog.Status)
	}
}