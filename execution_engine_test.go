package curator

import (
	"testing"
	"time"
)

func TestExecutionEngine_ExecutePlan(t *testing.T) {
	// Setup
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Add test files
	fs.AddFile("/source1.txt", []byte("content1"), "text/plain")
	fs.AddFile("/source2.pdf", []byte("content2"), "application/pdf")
	
	// Create a test plan
	plan := &ReorganizationPlan{
		ID:        "test-plan-123",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "/Documents",
				Reason:      "Create Documents folder",
				Type:        CreateFolder,
				FileCount:   0,
			},
			{
				ID:          "move-2",
				Source:      "/source1.txt",
				Destination: "/Documents/source1.txt",
				Reason:      "Move text file to Documents",
				Type:        FileMove,
				FileCount:   1,
			},
			{
				ID:          "move-3",
				Source:      "/source2.pdf",
				Destination: "/Documents/source2.pdf",
				Reason:      "Move PDF to Documents",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FoldersCreated: 1,
			FilesMoved:     2,
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
	
	if len(execLog.Completed) != 3 {
		t.Errorf("Expected 3 completed moves, got %d", len(execLog.Completed))
	}
	
	if len(execLog.Failed) != 0 {
		t.Errorf("Expected 0 failed moves, got %d", len(execLog.Failed))
	}
	
	// Verify filesystem changes
	exists, err := fs.Exists("/Documents")
	if err != nil {
		t.Fatalf("Failed to check if Documents folder exists: %v", err)
	}
	if !exists {
		t.Error("Documents folder should exist")
	}
	
	exists, err = fs.Exists("/Documents/source1.txt")
	if err != nil {
		t.Fatalf("Failed to check if moved file exists: %v", err)
	}
	if !exists {
		t.Error("Moved file should exist in Documents folder")
	}
	
	exists, err = fs.Exists("/source1.txt")
	if err != nil {
		t.Fatalf("Failed to check if original file exists: %v", err)
	}
	if exists {
		t.Error("Original file should no longer exist")
	}
}

func TestExecutionEngine_ExecutePlanWithConflicts(t *testing.T) {
	// Setup
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Add test file
	fs.AddFile("/source.txt", []byte("content"), "text/plain")
	
	// Create a plan with a conflict (source doesn't exist)
	plan := &ReorganizationPlan{
		ID:        "test-plan-conflict",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "/nonexistent.txt",
				Destination: "/Documents/nonexistent.txt",
				Reason:      "Move nonexistent file",
				Type:        FileMove,
				FileCount:   1,
			},
			{
				ID:          "move-2",
				Source:      "/source.txt",
				Destination: "/Documents/source.txt",
				Reason:      "Move existing file",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FilesMoved: 2,
		},
		Rationale: "Test conflict handling",
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
	if execLog.Status != StatusPartial {
		t.Errorf("Expected status %s, got %s", StatusPartial, execLog.Status)
	}
	
	// Should have some skipped operations due to conflicts
	if len(execLog.Skipped) == 0 {
		t.Error("Expected some skipped operations due to conflicts")
	}
	
	// Should still complete valid operations
	if len(execLog.Completed) == 0 {
		t.Error("Expected some completed operations")
	}
}

func TestExecutionEngine_ExecutePlanWithFailFast(t *testing.T) {
	// Setup
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Create a plan that will fail
	plan := &ReorganizationPlan{
		ID:        "test-plan-failfast",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "/nonexistent.txt",
				Destination: "/Documents/nonexistent.txt",
				Reason:      "This will fail",
				Type:        FileMove,
				FileCount:   1,
			},
			{
				ID:          "move-2",
				Source:      "/another.txt",
				Destination: "/Documents/another.txt",
				Reason:      "This should not execute with fail-fast",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FilesMoved: 2,
		},
		Rationale: "Test fail-fast behavior",
	}
	
	// Save the plan
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	// Execute the plan with fail-fast
	execLog, err := engine.ExecutePlan(plan.ID, true)
	
	// With fail-fast, should return error and partial status
	if err == nil {
		t.Error("Expected error with fail-fast mode")
	}
	
	if execLog.Status != StatusFailed {
		t.Errorf("Expected status %s, got %s", StatusFailed, execLog.Status)
	}
}

func TestExecutionEngine_ResumeExecution(t *testing.T) {
	// Setup
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Add test files
	fs.AddFile("/source1.txt", []byte("content1"), "text/plain")
	fs.AddFile("/source2.txt", []byte("content2"), "text/plain")
	
	// Create a plan
	plan := &ReorganizationPlan{
		ID:        "test-plan-resume",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "/Documents",
				Reason:      "Create Documents folder",
				Type:        CreateFolder,
				FileCount:   0,
			},
			{
				ID:          "move-2",
				Source:      "/source1.txt",
				Destination: "/Documents/source1.txt",
				Reason:      "Move first file",
				Type:        FileMove,
				FileCount:   1,
			},
			{
				ID:          "move-3",
				Source:      "/source2.txt",
				Destination: "/Documents/source2.txt",
				Reason:      "Move second file",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FoldersCreated: 1,
			FilesMoved:     2,
		},
		Rationale: "Test resume capability",
	}
	
	// Save the plan
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	// Simulate partially completed execution by manually logging some operations
	// and marking one as complete
	move1 := plan.Moves[0]
	move2 := plan.Moves[1]
	move3 := plan.Moves[2]
	
	// Log all operations first
	for _, move := range plan.Moves {
		op := &Operation{
			ID:        plan.ID + "-" + move.ID,
			Type:      string(move.Type),
			Data:      []byte(`{"id":"` + move.ID + `"}`),
			Timestamp: time.Now(),
		}
		err := store.LogOperation(op)
		if err != nil {
			t.Fatalf("Failed to log operation: %v", err)
		}
	}
	
	// Mark first operation as complete (simulate partial execution)
	err = store.MarkOperationComplete(plan.ID + "-" + move1.ID)
	if err != nil {
		t.Fatalf("Failed to mark operation complete: %v", err)
	}
	
	// Create the Documents folder manually (since move-1 was "completed")
	err = fs.CreateFolder("/Documents")
	if err != nil {
		t.Fatalf("Failed to create Documents folder: %v", err)
	}
	
	// Now execute the plan (should resume from pending operations)
	execLog, err := engine.ExecutePlan(plan.ID, false)
	if err != nil {
		t.Fatalf("Failed to resume execution: %v", err)
	}
	
	// Verify that execution completed successfully
	if execLog.Status != StatusCompleted {
		t.Errorf("Expected status %s, got %s", StatusCompleted, execLog.Status)
	}
	
	// Should have completed the remaining moves
	if len(execLog.Completed) == 0 {
		t.Error("Expected some completed moves")
	}
	
	// Verify files were moved
	exists, err := fs.Exists("/Documents/source1.txt")
	if err != nil || !exists {
		t.Error("First file should be moved to Documents")
	}
	
	exists, err = fs.Exists("/Documents/source2.txt")
	if err != nil || !exists {
		t.Error("Second file should be moved to Documents")
	}
}

func TestExecutionEngine_GetExecutionStatus(t *testing.T) {
	// Setup
	fs := NewMemoryFileSystem()
	store := NewMemoryOperationStore()
	engine := NewExecutionEngine(fs, store)
	
	// Create and execute a plan first
	plan := &ReorganizationPlan{
		ID:        "test-plan-status",
		Timestamp: time.Now(),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "/Documents",
				Reason:      "Create Documents folder",
				Type:        CreateFolder,
				FileCount:   0,
			},
		},
		Summary: Summary{
			FoldersCreated: 1,
		},
		Rationale: "Test status retrieval",
	}
	
	err := store.SavePlan(plan)
	if err != nil {
		t.Fatalf("Failed to save plan: %v", err)
	}
	
	execLog, err := engine.ExecutePlan(plan.ID, false)
	if err != nil {
		t.Fatalf("Failed to execute plan: %v", err)
	}
	
	// Get execution status
	status, err := engine.GetExecutionStatus(plan.ID)
	if err != nil {
		t.Fatalf("Failed to get execution status: %v", err)
	}
	
	// Verify status matches the execution log
	if status.PlanID != execLog.PlanID {
		t.Errorf("Expected plan ID %s, got %s", execLog.PlanID, status.PlanID)
	}
	
	if status.Status != execLog.Status {
		t.Errorf("Expected status %s, got %s", execLog.Status, status.Status)
	}
	
	if len(status.Completed) != len(execLog.Completed) {
		t.Errorf("Expected %d completed moves, got %d", len(execLog.Completed), len(status.Completed))
	}
}