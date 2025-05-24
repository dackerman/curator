package curator

import (
	"testing"
	"time"
)

func TestExecutionEngine_ExecutePlan_Success(t *testing.T) {
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Add test files
	fs.AddFile("/document.pdf", []byte("content"), "application/pdf")
	fs.AddFile("/image.jpg", []byte("content"), "image/jpeg")
	
	// Create a simple reorganization plan
	plan := &ReorganizationPlan{
		ID:        "test-plan-1",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "Documents",
				Reason:      "Create Documents folder",
				Type:        CreateFolder,
				FileCount:   0,
			},
			{
				ID:          "move-2",
				Source:      "/document.pdf",
				Destination: "Documents/document.pdf",
				Reason:      "Move document to Documents folder",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FoldersCreated: 1,
			FilesMoved:     1,
		},
		Rationale: "Test reorganization",
	}
	
	// Save the plan
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	// Execute the plan
	execLog, err := engine.ExecutePlan(plan.ID, false)
	if err != nil {
		t.Fatalf("Failed to execute plan: %v", err)
	}
	
	// Verify execution log
	if execLog.PlanID != plan.ID {
		t.Errorf("Expected plan ID %s, got %s", plan.ID, execLog.PlanID)
	}
	
	if execLog.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, execLog.Status)
	}
	
	if len(execLog.Completed) != 2 {
		t.Errorf("Expected 2 completed moves, got %d", len(execLog.Completed))
	}
	
	if len(execLog.Failed) != 0 {
		t.Errorf("Expected 0 failed moves, got %d", len(execLog.Failed))
	}
	
	// Verify files were actually moved
	exists, err := fs.Exists("Documents/document.pdf")
	if err != nil {
		t.Fatalf("Failed to check if file exists: %v", err)
	}
	if !exists {
		t.Error("File should have been moved to Documents folder")
	}
	
	// Original file should no longer exist
	exists, err = fs.Exists("/document.pdf")
	if err != nil {
		t.Fatalf("Failed to check if original file exists: %v", err)
	}
	if exists {
		t.Error("Original file should have been moved away")
	}
}

func TestExecutionEngine_ExecutePlan_WithConflicts(t *testing.T) {
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Add test file
	fs.AddFile("/document.pdf", []byte("content"), "application/pdf")
	
	// Create Documents folder and destination file to cause conflict
	fs.CreateFolder("Documents")
	fs.AddFile("Documents/document.pdf", []byte("existing"), "application/pdf")
	
	// Create a plan that will have conflicts
	plan := &ReorganizationPlan{
		ID:        "test-plan-conflict",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "Documents",
				Reason:      "Create Documents folder",
				Type:        CreateFolder,
				FileCount:   0,
			},
			{
				ID:          "move-2",
				Source:      "/document.pdf",
				Destination: "Documents/document.pdf",
				Reason:      "Move document to Documents folder",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FoldersCreated: 1,
			FilesMoved:     1,
		},
		Rationale: "Test reorganization with conflicts",
	}
	
	// Save the plan
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	// Execute the plan
	execLog, err := engine.ExecutePlan(plan.ID, false)
	if err != nil {
		t.Fatalf("Failed to execute plan: %v", err)
	}
	
	// Verify execution log shows conflict handling
	if execLog.Status != StatusPartial {
		t.Errorf("Expected status %s, got %s", StatusPartial, execLog.Status)
	}
	
	// Should have one completed (folder creation) and one skipped (file move due to conflict)
	if len(execLog.Completed) != 1 {
		t.Errorf("Expected 1 completed move, got %d", len(execLog.Completed))
	}
	
	if len(execLog.Skipped) != 1 {
		t.Errorf("Expected 1 skipped move, got %d", len(execLog.Skipped))
	}
	
	// Verify the conflict reason
	if len(execLog.Skipped) > 0 {
		skipped := execLog.Skipped[0]
		if skipped.MoveID != "move-2" {
			t.Errorf("Expected skipped move ID 'move-2', got %s", skipped.MoveID)
		}
	}
}

func TestExecutionEngine_ExecutePlan_FailFast(t *testing.T) {
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Create a plan that will have conflicts (non-existent file results in conflict, not error)
	plan := &ReorganizationPlan{
		ID:        "test-plan-failfast",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "/nonexistent.txt",
				Destination: "Documents/nonexistent.txt",
				Reason:      "Move non-existent file",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FilesMoved: 1,
		},
		Rationale: "Test conflict handling with fail-fast",
	}
	
	// Save the plan
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	// Execute the plan with fail-fast enabled
	execLog, err := engine.ExecutePlan(plan.ID, true)
	
	// Should not return an error for conflicts (they are skipped)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Verify execution log shows the conflict was skipped
	if execLog.Status != StatusPartial {
		t.Errorf("Expected status %s, got %s", StatusPartial, execLog.Status)
	}
	
	if len(execLog.Skipped) != 1 {
		t.Errorf("Expected 1 skipped operation, got %d", len(execLog.Skipped))
	}
}

func TestExecutionEngine_GetExecutionStatus(t *testing.T) {
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Create and execute a simple plan
	plan := &ReorganizationPlan{
		ID:        "test-plan-status",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "TestFolder",
				Reason:      "Create test folder",
				Type:        CreateFolder,
				FileCount:   0,
			},
		},
		Summary: Summary{
			FoldersCreated: 1,
		},
		Rationale: "Test status retrieval",
	}
	
	// Save and execute the plan
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	_, err = engine.ExecutePlan(plan.ID, false)
	if err != nil {
		t.Fatalf("Failed to execute plan: %v", err)
	}
	
	// Get execution status
	status, err := engine.GetExecutionStatus(plan.ID)
	if err != nil {
		t.Fatalf("Failed to get execution status: %v", err)
	}
	
	if status.PlanID != plan.ID {
		t.Errorf("Expected plan ID %s, got %s", plan.ID, status.PlanID)
	}
	
	if status.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, status.Status)
	}
}

func TestExecutionEngine_ResumePendingOperations(t *testing.T) {
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Add test file
	fs.AddFile("/test.txt", []byte("content"), "text/plain")
	
	// Manually log an operation without completing it
	_ = Move{
		ID:          "pending-move",
		Source:      "/test.txt",
		Destination: "Documents/test.txt",
		Reason:      "Test pending operation",
		Type:        FileMove,
		FileCount:   1,
	}
	
	// Create operation data
	operation := &Operation{
		ID:        "test-op-1",
		Type:      "move",
		Data:      []byte(`{"ID":"pending-move","Source":"/test.txt","Destination":"Documents/test.txt","Reason":"Test pending operation","Type":"FILE_MOVE","FileCount":1}`),
		Timestamp: time.Now(),
	}
	
	// Log the operation but don't mark it complete
	err := store.LogOperation(operation)
	if err != nil {
		t.Fatalf("Failed to log operation: %v", err)
	}
	
	// Verify there's a pending operation
	pending, err := store.GetPendingOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations: %v", err)
	}
	
	if len(pending) != 1 {
		t.Fatalf("Expected 1 pending operation, got %d", len(pending))
	}
	
	// Resume pending operations
	err = engine.ResumePendingOperations()
	if err != nil {
		t.Fatalf("Failed to resume pending operations: %v", err)
	}
	
	// Verify the operation was completed
	pending, err = store.GetPendingOperations()
	if err != nil {
		t.Fatalf("Failed to get pending operations after resume: %v", err)
	}
	
	if len(pending) != 0 {
		t.Errorf("Expected 0 pending operations after resume, got %d", len(pending))
	}
}