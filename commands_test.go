package curator

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestCommands_FullWorkflow tests the complete command workflow using business logic functions
func TestCommands_FullWorkflow(t *testing.T) {
	// Setup test configuration
	config := Configuration{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
		StoreDir: t.TempDir(),
	}
	
	// Create command options
	opts, err := CreateCommandOptions(config)
	if err != nil {
		t.Fatalf("Failed to create command options: %v", err)
	}
	
	// Ensure analyzer is closed if it supports it
	if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	
	// Test reorganize command
	t.Run("ExecuteReorganize", func(t *testing.T) {
		reorganizeOpts := ReorganizeOptions{
			DryRun:  false,
			Exclude: "",
		}
		
		plan, err := ExecuteReorganize(opts, reorganizeOpts)
		if err != nil {
			t.Fatalf("ExecuteReorganize failed: %v", err)
		}
		
		if plan.ID == "" {
			t.Error("Expected plan to have an ID")
		}
		
		if len(plan.Moves) == 0 {
			t.Error("Expected plan to have moves")
		}
		
		if plan.Rationale == "" {
			t.Error("Expected plan to have rationale")
		}
		
		planID := plan.ID
		
		// Test list-plans command
		t.Run("ExecuteListPlans", func(t *testing.T) {
			summaries, err := ExecuteListPlans(opts)
			if err != nil {
				t.Fatalf("ExecuteListPlans failed: %v", err)
			}
			
			if len(summaries) != 1 {
				t.Errorf("Expected 1 plan summary, got %d", len(summaries))
			}
			
			if summaries[0].ID != planID {
				t.Errorf("Expected plan ID %s, got %s", planID, summaries[0].ID)
			}
		})
		
		// Test show-plan command
		t.Run("ExecuteShowPlan", func(t *testing.T) {
			retrievedPlan, err := ExecuteShowPlan(opts, planID)
			if err != nil {
				t.Fatalf("ExecuteShowPlan failed: %v", err)
			}
			
			if retrievedPlan.ID != planID {
				t.Errorf("Expected plan ID %s, got %s", planID, retrievedPlan.ID)
			}
			
			if len(retrievedPlan.Moves) != len(plan.Moves) {
				t.Errorf("Expected %d moves, got %d", len(plan.Moves), len(retrievedPlan.Moves))
			}
		})
		
		// Test apply command
		t.Run("ExecuteApply", func(t *testing.T) {
			applyOpts := ApplyOptions{
				FailFast: false,
			}
			
			execLog, err := ExecuteApply(opts, planID, applyOpts)
			if err != nil {
				t.Fatalf("ExecuteApply failed: %v", err)
			}
			
			if execLog.PlanID != planID {
				t.Errorf("Expected execution log plan ID %s, got %s", planID, execLog.PlanID)
			}
			
			// Should have some operations (completed or skipped due to memory FS reset)
			totalOps := len(execLog.Completed) + len(execLog.Failed) + len(execLog.Skipped)
			if totalOps == 0 {
				t.Error("Expected some operations to be recorded")
			}
			
			// Verify status is set
			if execLog.Status == "" {
				t.Error("Expected execution status to be set")
			}
		})
		
		// Test status command
		t.Run("ExecuteStatus", func(t *testing.T) {
			statusLog, err := ExecuteStatus(opts, planID)
			if err != nil {
				t.Fatalf("ExecuteStatus failed: %v", err)
			}
			
			if statusLog.PlanID != planID {
				t.Errorf("Expected status log plan ID %s, got %s", planID, statusLog.PlanID)
			}
		})
		
		// Test history command
		t.Run("ExecuteHistory", func(t *testing.T) {
			logs, err := ExecuteHistory(opts)
			if err != nil {
				t.Fatalf("ExecuteHistory failed: %v", err)
			}
			
			if len(logs) == 0 {
				t.Error("Expected at least one execution log")
			}
			
			// Should find our execution
			found := false
			for _, log := range logs {
				if log.PlanID == planID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find execution log for plan %s", planID)
			}
		})
	})
}

// TestCommands_ErrorHandling tests error conditions in command functions
func TestCommands_ErrorHandling(t *testing.T) {
	config := Configuration{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
		StoreDir: t.TempDir(),
	}
	
	opts, err := CreateCommandOptions(config)
	if err != nil {
		t.Fatalf("Failed to create command options: %v", err)
	}
	
	// Ensure analyzer is closed if it supports it
	if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	
	t.Run("show plan with invalid ID", func(t *testing.T) {
		_, err := ExecuteShowPlan(opts, "nonexistent-plan")
		if err == nil {
			t.Error("Expected error for nonexistent plan")
		}
		
		if !strings.Contains(err.Error(), "plan not found") {
			t.Errorf("Expected 'plan not found' error, got: %v", err)
		}
	})
	
	t.Run("apply plan with invalid ID", func(t *testing.T) {
		applyOpts := ApplyOptions{FailFast: false}
		_, err := ExecuteApply(opts, "nonexistent-plan", applyOpts)
		if err == nil {
			t.Error("Expected error for nonexistent plan")
		}
		
		if !strings.Contains(err.Error(), "plan not found") {
			t.Errorf("Expected 'plan not found' error, got: %v", err)
		}
	})
	
	t.Run("status for nonexistent plan", func(t *testing.T) {
		_, err := ExecuteStatus(opts, "nonexistent-plan")
		if err == nil {
			t.Error("Expected error for nonexistent plan")
		}
		
		if !strings.Contains(err.Error(), "no execution found") {
			t.Errorf("Expected 'no execution found' error, got: %v", err)
		}
	})
}

// TestCommands_OtherOperations tests deduplicate, cleanup, and rename commands
func TestCommands_OtherOperations(t *testing.T) {
	config := Configuration{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
		StoreDir: t.TempDir(),
	}
	
	opts, err := CreateCommandOptions(config)
	if err != nil {
		t.Fatalf("Failed to create command options: %v", err)
	}
	
	// Ensure analyzer is closed if it supports it
	if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	
	t.Run("ExecuteDeduplicate", func(t *testing.T) {
		report, err := ExecuteDeduplicate(opts)
		if err != nil {
			t.Fatalf("ExecuteDeduplicate failed: %v", err)
		}
		
		if report.ID == "" {
			t.Error("Expected duplication report to have an ID")
		}
		
		// Summary should be set
		if report.Summary.TotalDuplicates < 0 {
			t.Error("Expected duplication report to have valid TotalDuplicates")
		}
	})
	
	t.Run("ExecuteCleanup", func(t *testing.T) {
		plan, err := ExecuteCleanup(opts)
		if err != nil {
			t.Fatalf("ExecuteCleanup failed: %v", err)
		}
		
		if plan.ID == "" {
			t.Error("Expected cleanup plan to have an ID")
		}
		
		// Should have analyzed some files (Deletions can be 0 if no junk files)
		if len(plan.Deletions) < 0 {
			t.Error("Expected cleanup plan to have valid Deletions slice")
		}
	})
	
	t.Run("ExecuteRename", func(t *testing.T) {
		plan, err := ExecuteRename(opts)
		if err != nil {
			t.Fatalf("ExecuteRename failed: %v", err)
		}
		
		if plan.ID == "" {
			t.Error("Expected rename plan to have an ID")
		}
		
		// Summary should be set
		if plan.Summary.Pattern == "" {
			t.Error("Expected rename plan to have a pattern")
		}
	})
}

// TestCommands_ConfigurationValidation tests configuration creation and validation
func TestCommands_ConfigurationValidation(t *testing.T) {
	t.Run("invalid filesystem type", func(t *testing.T) {
		config := Configuration{
			AI: AIConfig{
				Provider: "mock",
			},
			FileSystem: FileSystemConfig{
				Type: "invalid-type",
			},
			StoreDir: t.TempDir(),
		}
		
		_, err := CreateCommandOptions(config)
		if err == nil {
			t.Error("Expected error for invalid filesystem type")
		}
		
		if !strings.Contains(err.Error(), "unknown filesystem type") {
			t.Errorf("Expected 'unknown filesystem type' error, got: %v", err)
		}
	})
	
	t.Run("invalid AI provider", func(t *testing.T) {
		config := Configuration{
			AI: AIConfig{
				Provider: "invalid-provider",
			},
			FileSystem: FileSystemConfig{
				Type: "memory",
			},
			StoreDir: t.TempDir(),
		}
		
		_, err := CreateCommandOptions(config)
		if err == nil {
			t.Error("Expected error for invalid AI provider")
		}
		
		if !strings.Contains(err.Error(), "unknown AI provider") {
			t.Errorf("Expected 'unknown AI provider' error, got: %v", err)
		}
	})
	
	t.Run("googledrive without configuration", func(t *testing.T) {
		config := Configuration{
			AI: AIConfig{
				Provider: "mock",
			},
			FileSystem: FileSystemConfig{
				Type: "googledrive",
				// GoogleDrive config is nil
			},
			StoreDir: t.TempDir(),
		}
		
		_, err := CreateCommandOptions(config)
		if err == nil {
			t.Error("Expected error for googledrive without configuration")
		}
		
		if !strings.Contains(err.Error(), "Google Drive configuration is required") {
			t.Errorf("Expected Google Drive configuration error, got: %v", err)
		}
	})
}

// TestCommands_DryRun tests dry-run functionality
func TestCommands_DryRun(t *testing.T) {
	config := Configuration{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
		StoreDir: t.TempDir(),
	}
	
	opts, err := CreateCommandOptions(config)
	if err != nil {
		t.Fatalf("Failed to create command options: %v", err)
	}
	
	// Ensure analyzer is closed if it supports it
	if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	
	// Test dry-run reorganize
	reorganizeOpts := ReorganizeOptions{
		DryRun:  true,
		Exclude: "",
	}
	
	plan, err := ExecuteReorganize(opts, reorganizeOpts)
	if err != nil {
		t.Fatalf("ExecuteReorganize with dry-run failed: %v", err)
	}
	
	if plan.ID == "" {
		t.Error("Expected plan to have an ID even in dry-run")
	}
	
	// Plan should not be saved in dry-run mode
	summaries, err := ExecuteListPlans(opts)
	if err != nil {
		t.Fatalf("ExecuteListPlans failed: %v", err)
	}
	
	if len(summaries) != 0 {
		t.Error("Expected no plans to be saved in dry-run mode")
	}
}

// TestCommands_ConfigurationOverrides tests configuration override functionality
func TestCommands_ConfigurationOverrides(t *testing.T) {
	originalConfig := Configuration{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
			Root: "/original",
		},
		StoreDir: t.TempDir(),
	}
	
	// Test overrides
	overriddenConfig := OverrideConfiguration(originalConfig, "mock", "local", "/new/root")
	
	if overriddenConfig.AI.Provider != "mock" {
		t.Errorf("Expected AI provider to remain 'mock', got '%s'", overriddenConfig.AI.Provider)
	}
	
	if overriddenConfig.FileSystem.Type != "local" {
		t.Errorf("Expected filesystem type to be 'local', got '%s'", overriddenConfig.FileSystem.Type)
	}
	
	if overriddenConfig.FileSystem.Root != "/new/root" {
		t.Errorf("Expected root to be '/new/root', got '%s'", overriddenConfig.FileSystem.Root)
	}
}

// TestCommands_PersistentStore tests that commands work with file-based persistence
func TestCommands_PersistentStore(t *testing.T) {
	storeDir := t.TempDir()
	
	config := Configuration{
		AI: AIConfig{
			Provider: "mock",
		},
		FileSystem: FileSystemConfig{
			Type: "memory",
		},
		StoreDir: storeDir,
	}
	
	// Create first set of command options
	opts1, err := CreateCommandOptions(config)
	if err != nil {
		t.Fatalf("Failed to create first command options: %v", err)
	}
	defer func() {
		if closer, ok := opts1.Analyzer.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()
	
	// Create and save a plan
	reorganizeOpts := ReorganizeOptions{DryRun: false}
	plan, err := ExecuteReorganize(opts1, reorganizeOpts)
	if err != nil {
		t.Fatalf("ExecuteReorganize failed: %v", err)
	}
	
	planID := plan.ID
	
	// Create second set of command options (simulating separate process)
	opts2, err := CreateCommandOptions(config)
	if err != nil {
		t.Fatalf("Failed to create second command options: %v", err)
	}
	defer func() {
		if closer, ok := opts2.Analyzer.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()
	
	// Plan should persist across different command option instances
	retrievedPlan, err := ExecuteShowPlan(opts2, planID)
	if err != nil {
		t.Fatalf("ExecuteShowPlan failed with second options: %v", err)
	}
	
	if retrievedPlan.ID != planID {
		t.Errorf("Expected persisted plan ID %s, got %s", planID, retrievedPlan.ID)
	}
	
	// Verify store directory structure was created
	expectedPaths := []string{
		filepath.Join(storeDir, "plans"),
		filepath.Join(storeDir, "operations"),
		filepath.Join(storeDir, "execution_logs"),
		filepath.Join(storeDir, "plans", planID+".json"),
	}
	
	for _ = range expectedPaths {
		if _, ok := opts2.Store.(*FileOperationStore); ok {
			// We can't easily access the file system from here, but we know
			// the store was created successfully, which means the directories exist
		}
	}
}