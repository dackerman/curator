package curator

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_FullWorkflow tests the complete CLI workflow:
// reorganize -> list-plans -> show-plan -> apply -> status -> history
func TestCLI_FullWorkflow(t *testing.T) {
	// Build the CLI binary for testing
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	// Create temporary directory for testing
	tempDir := t.TempDir()
	
	// Set up test environment with memory filesystem
	env := []string{
		"CURATOR_FILESYSTEM_TYPE=memory",
		"CURATOR_AI_PROVIDER=mock",
	}
	
	t.Run("reorganize command", func(t *testing.T) {
		// Run reorganize command
		output, err := runCLICommand(t, binaryPath, tempDir, env, "reorganize")
		if err != nil {
			t.Fatalf("reorganize command failed: %v\nOutput: %s", err, output)
		}
		
		// Verify reorganize output
		if !strings.Contains(output, "REORGANIZATION PLAN") {
			t.Error("Expected reorganize output to contain 'REORGANIZATION PLAN'")
		}
		if !strings.Contains(output, "Plan saved with ID:") {
			t.Error("Expected reorganize output to contain 'Plan saved with ID:'")
		}
		// Note: File count output was removed during refactoring to business logic
		// The business logic now handles file counting internally
		
		// Extract plan ID from output
		planID := extractPlanID(output)
		if planID == "" {
			t.Fatal("Could not extract plan ID from reorganize output")
		}
		t.Logf("Generated plan ID: %s", planID)
		
		t.Run("list-plans command", func(t *testing.T) {
			// Run list-plans command
			output, err := runCLICommand(t, binaryPath, tempDir, env, "list-plans")
			if err != nil {
				t.Fatalf("list-plans command failed: %v\nOutput: %s", err, output)
			}
			
			// Verify list-plans output
			if !strings.Contains(output, "REORGANIZATION PLANS") {
				t.Error("Expected list-plans output to contain 'REORGANIZATION PLANS'")
			}
			if !strings.Contains(output, planID) {
				t.Errorf("Expected list-plans output to contain plan ID '%s'", planID)
			}
			if !strings.Contains(output, "pending") {
				t.Error("Expected list-plans output to show 'pending' status")
			}
		})
		
		t.Run("show-plan command", func(t *testing.T) {
			// Run show-plan command
			output, err := runCLICommand(t, binaryPath, tempDir, env, "show-plan", planID)
			if err != nil {
				t.Fatalf("show-plan command failed: %v\nOutput: %s", err, output)
			}
			
			// Verify show-plan output
			if !strings.Contains(output, "REORGANIZATION PLAN") {
				t.Error("Expected show-plan output to contain 'REORGANIZATION PLAN'")
			}
			if !strings.Contains(output, planID) {
				t.Errorf("Expected show-plan output to contain plan ID '%s'", planID)
			}
			if !strings.Contains(output, "DETAILED OPERATIONS") {
				t.Error("Expected show-plan output to contain 'DETAILED OPERATIONS'")
			}
		})
		
		t.Run("apply command", func(t *testing.T) {
			// Run apply command
			output, err := runCLICommand(t, binaryPath, tempDir, env, "apply", planID)
			if err != nil {
				t.Fatalf("apply command failed: %v\nOutput: %s", err, output)
			}
			
			// Verify apply output
			if !strings.Contains(output, "Executing plan "+planID) {
				t.Errorf("Expected apply output to contain 'Executing plan %s'", planID)
			}
			if !strings.Contains(output, "EXECUTION REPORT") {
				t.Error("Expected apply output to contain 'EXECUTION REPORT'")
			}
			// Should be partial completion due to memory filesystem reset between commands
			if !strings.Contains(output, "PARTIALLY COMPLETED") && !strings.Contains(output, "COMPLETED") {
				t.Error("Expected apply output to show completion status")
			}
		})
		
		t.Run("status command", func(t *testing.T) {
			// Run status command
			output, err := runCLICommand(t, binaryPath, tempDir, env, "status", planID)
			if err != nil {
				t.Fatalf("status command failed: %v\nOutput: %s", err, output)
			}
			
			// Verify status output
			if !strings.Contains(output, "EXECUTION REPORT") {
				t.Error("Expected status output to contain 'EXECUTION REPORT'")
			}
			if !strings.Contains(output, planID) {
				t.Errorf("Expected status output to contain plan ID '%s'", planID)
			}
		})
		
		t.Run("history command", func(t *testing.T) {
			// Run history command
			output, err := runCLICommand(t, binaryPath, tempDir, env, "history")
			if err != nil {
				t.Fatalf("history command failed: %v\nOutput: %s", err, output)
			}
			
			// Verify history output
			if !strings.Contains(output, "EXECUTION HISTORY") {
				t.Error("Expected history output to contain 'EXECUTION HISTORY'")
			}
			if !strings.Contains(output, planID) {
				t.Errorf("Expected history output to contain plan ID '%s'", planID)
			}
		})
	})
}

// TestCLI_ErrorHandling tests error conditions in CLI commands
func TestCLI_ErrorHandling(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	tempDir := t.TempDir()
	env := []string{
		"CURATOR_FILESYSTEM_TYPE=memory",
		"CURATOR_AI_PROVIDER=mock",
	}
	
	t.Run("apply with invalid plan ID", func(t *testing.T) {
		output, err := runCLICommand(t, binaryPath, tempDir, env, "apply", "nonexistent-plan")
		if err == nil {
			t.Error("Expected apply with invalid plan ID to fail")
		}
		
		if !strings.Contains(output, "plan not found") {
			t.Error("Expected error message about plan not found")
		}
	})
	
	t.Run("show-plan with invalid plan ID", func(t *testing.T) {
		output, err := runCLICommand(t, binaryPath, tempDir, env, "show-plan", "nonexistent-plan")
		if err == nil {
			t.Error("Expected show-plan with invalid plan ID to fail")
		}
		
		if !strings.Contains(output, "plan not found") {
			t.Error("Expected error message about plan not found")
		}
	})
	
	t.Run("status with invalid plan ID", func(t *testing.T) {
		output, err := runCLICommand(t, binaryPath, tempDir, env, "status", "nonexistent-plan")
		if err == nil {
			t.Error("Expected status with invalid plan ID to fail")
		}
		
		if !strings.Contains(output, "no execution found") {
			t.Error("Expected error message about no execution found")
		}
	})
	
	t.Run("invalid filesystem type", func(t *testing.T) {
		invalidEnv := []string{
			"CURATOR_FILESYSTEM_TYPE=invalid-type",
			"CURATOR_AI_PROVIDER=mock",
		}
		
		output, err := runCLICommand(t, binaryPath, tempDir, invalidEnv, "reorganize")
		if err == nil {
			t.Error("Expected reorganize with invalid filesystem type to fail")
		}
		
		if !strings.Contains(output, "unknown filesystem type") {
			t.Error("Expected error message about unknown filesystem type")
		}
	})
}

// TestCLI_GoogleDriveConfiguration tests Google Drive OAuth2 configuration errors
func TestCLI_GoogleDriveConfiguration(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	tempDir := t.TempDir()
	
	t.Run("googledrive without credentials", func(t *testing.T) {
		env := []string{
			"CURATOR_FILESYSTEM_TYPE=googledrive",
			"CURATOR_AI_PROVIDER=mock",
		}
		
		output, err := runCLICommand(t, binaryPath, tempDir, env, "reorganize")
		if err == nil {
			t.Error("Expected googledrive without credentials to fail")
		}
		
		if !strings.Contains(output, "OAuth2 credentials file path is required") {
			t.Errorf("Expected error message about OAuth2 credentials file, got: %s", output)
		}
	})
	
	t.Run("googledrive with invalid credentials file", func(t *testing.T) {
		env := []string{
			"CURATOR_FILESYSTEM_TYPE=googledrive",
			"CURATOR_AI_PROVIDER=mock",
			"GOOGLE_DRIVE_OAUTH_CREDENTIALS=/nonexistent/file.json",
		}
		
		output, err := runCLICommand(t, binaryPath, tempDir, env, "reorganize")
		if err == nil {
			t.Error("Expected googledrive with invalid credentials file to fail")
		}
		
		if !strings.Contains(output, "failed to create command options") {
			t.Error("Expected error message about failed to create command options")
		}
	})
}

// TestCLI_OtherCommands tests the remaining CLI commands
func TestCLI_OtherCommands(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	tempDir := t.TempDir()
	env := []string{
		"CURATOR_FILESYSTEM_TYPE=memory",
		"CURATOR_AI_PROVIDER=mock",
	}
	
	t.Run("deduplicate command", func(t *testing.T) {
		output, err := runCLICommand(t, binaryPath, tempDir, env, "deduplicate")
		if err != nil {
			t.Fatalf("deduplicate command failed: %v\nOutput: %s", err, output)
		}
		
		if !strings.Contains(output, "Scanning for duplicate files") {
			t.Error("Expected deduplicate output to contain scanning message")
		}
		if !strings.Contains(output, "DUPLICATION REPORT") {
			t.Error("Expected deduplicate output to contain 'DUPLICATION REPORT'")
		}
	})
	
	t.Run("cleanup command", func(t *testing.T) {
		output, err := runCLICommand(t, binaryPath, tempDir, env, "cleanup")
		if err != nil {
			t.Fatalf("cleanup command failed: %v\nOutput: %s", err, output)
		}
		
		if !strings.Contains(output, "Scanning for junk files") {
			t.Error("Expected cleanup output to contain scanning message")
		}
		if !strings.Contains(output, "CLEANUP PLAN") {
			t.Error("Expected cleanup output to contain 'CLEANUP PLAN'")
		}
	})
	
	t.Run("rename command", func(t *testing.T) {
		output, err := runCLICommand(t, binaryPath, tempDir, env, "rename")
		if err != nil {
			t.Fatalf("rename command failed: %v\nOutput: %s", err, output)
		}
		
		if !strings.Contains(output, "Scanning for files to rename") {
			t.Error("Expected rename output to contain scanning message")
		}
		if !strings.Contains(output, "RENAMING PLAN") {
			t.Error("Expected rename output to contain 'RENAMING PLAN'")
		}
	})
}

// Helper functions

// buildCLIBinary builds the curator CLI binary for testing
func buildCLIBinary(t *testing.T) string {
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "curator-test")
	
	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/curator")
	cmd.Dir = "." // Run from the project root
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build CLI binary: %v\nStderr: %s", err, stderr.String())
	}
	
	return binaryPath
}

// runCLICommand runs a CLI command and returns its output
func runCLICommand(t *testing.T, binaryPath, workDir string, env []string, args ...string) (string, error) {
	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), env...)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	// Combine stdout and stderr for complete output
	output := stdout.String() + stderr.String()
	
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}
	
	return output, nil
}

// extractPlanID extracts the plan ID from reorganize command output
func extractPlanID(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Plan saved with ID:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
		if strings.Contains(line, "Plan ID:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

// TestCLI_PersistentStore tests that plans persist across different command invocations
func TestCLI_PersistentStore(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	// Use a specific temporary directory for the store
	storeDir := t.TempDir()
	
	env := []string{
		"CURATOR_FILESYSTEM_TYPE=memory",
		"CURATOR_AI_PROVIDER=mock",
		"CURATOR_STORE_DIR=" + storeDir,
	}
	
	// Step 1: Create a plan
	output1, err := runCLICommand(t, binaryPath, storeDir, env, "reorganize")
	if err != nil {
		t.Fatalf("First reorganize command failed: %v", err)
	}
	
	planID := extractPlanID(output1)
	if planID == "" {
		t.Fatal("Could not extract plan ID from first reorganize")
	}
	
	// Step 2: Verify plan persists in a separate command invocation
	output2, err := runCLICommand(t, binaryPath, storeDir, env, "list-plans")
	if err != nil {
		t.Fatalf("list-plans command failed: %v", err)
	}
	
	if !strings.Contains(output2, planID) {
		t.Errorf("Plan %s should persist between command invocations", planID)
	}
	
	// Step 3: Show the plan in another separate invocation
	output3, err := runCLICommand(t, binaryPath, storeDir, env, "show-plan", planID)
	if err != nil {
		t.Fatalf("show-plan command failed: %v", err)
	}
	
	if !strings.Contains(output3, planID) {
		t.Errorf("Should be able to show plan %s from persistent store", planID)
	}
	
	// Step 4: Verify store directory structure was created
	expectedPaths := []string{
		filepath.Join(storeDir, "plans"),
		filepath.Join(storeDir, "operations"),
		filepath.Join(storeDir, "execution_logs"),
		filepath.Join(storeDir, "plans", planID+".json"),
	}
	
	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected path to exist: %s", path)
		}
	}
}

// TestCLI_ApplyBugFix tests that the apply command no longer panics with nil pointer
// This is a regression test for the bug where ExecutionEngine was not initialized
func TestCLI_ApplyBugFix(t *testing.T) {
	binaryPath := buildCLIBinary(t)
	defer os.Remove(binaryPath)
	
	storeDir := t.TempDir()
	
	env := []string{
		"CURATOR_FILESYSTEM_TYPE=memory",
		"CURATOR_AI_PROVIDER=mock",
		"CURATOR_STORE_DIR=" + storeDir,
	}
	
	// Step 1: Create a plan
	output1, err := runCLICommand(t, binaryPath, storeDir, env, "reorganize")
	if err != nil {
		t.Fatalf("reorganize command failed: %v", err)
	}
	
	planID := extractPlanID(output1)
	if planID == "" {
		t.Fatal("Could not extract plan ID")
	}
	
	// Step 2: This used to panic with "invalid memory address or nil pointer dereference"
	// Now it should work without panic
	output2, err := runCLICommand(t, binaryPath, storeDir, env, "apply", planID)
	// We don't check err here because apply might fail due to file conflicts,
	// but it should NOT panic
	
	// Verify apply command ran and produced output (didn't crash)
	if !strings.Contains(output2, "Executing plan") {
		t.Errorf("Apply command should have run without panic, got output: %s", output2)
	}
	
	// Also test status command (which also used the nil engine)
	output3, err := runCLICommand(t, binaryPath, storeDir, env, "status", planID)
	if err != nil {
		t.Fatalf("status command failed: %v", err)
	}
	
	if !strings.Contains(output3, "EXECUTION REPORT") {
		t.Errorf("Status command should work without panic, got output: %s", output3)
	}
}