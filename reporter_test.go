package curator

import (
	"strings"
	"testing"
	"time"
)

func TestReporter_FormatReorganizationPlan(t *testing.T) {
	reporter := NewReporter()
	
	// Create test plan
	plan := &ReorganizationPlan{
		ID:        "test-plan-123",
		Timestamp: time.Date(2024, 1, 15, 14, 30, 52, 0, time.UTC),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "/Documents",
				Reason:      "Create organized folder for documents",
				Type:        CreateFolder,
				FileCount:   0,
			},
			{
				ID:          "move-2",
				Source:      "/random/file1.pdf",
				Destination: "/Documents/file1.pdf",
				Reason:      "Move PDF to Documents folder",
				Type:        FileMove,
				FileCount:   1,
			},
		},
		Summary: Summary{
			FoldersCreated:          1,
			FilesMoved:              1,
			FoldersMovedDeduplicated: 0,
			DepthReduction:          "15%",
			OrganizationImprovement: "90% of files will be organized",
		},
		Rationale: "Organize files by type for better management",
	}
	
	// Format the plan
	output := reporter.FormatReorganizationPlan(plan)
	
	// Verify output contains expected sections
	expectedSections := []string{
		"REORGANIZATION PLAN",
		"Plan ID: test-plan-123",
		"Generated: 2024-01-15 14:30:52",
		"SUMMARY",
		"✓ Create 1 new folders",
		"✓ Move 1 files",
		"✓ 15% reduction in folder depth",
		"✓ 90% of files will be organized",
		"RATIONALE",
		"Organize files by type for better management",
		"DETAILED OPERATIONS",
		"1. CREATE FOLDER: /Documents",
		"2. MOVE: /random/file1.pdf → /Documents/file1.pdf",
		"curator apply test-plan-123",
	}
	
	for _, expected := range expectedSections {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain: %s", expected)
		}
	}
}

func TestReporter_FormatExecutionLog(t *testing.T) {
	reporter := NewReporter()
	
	// Create test execution log
	execLog := &ExecutionLog{
		PlanID:    "test-plan-456",
		Timestamp: time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
		Status:    StatusPartial,
		Completed: []CompletedMove{
			{
				MoveID:    "move-1",
				Timestamp: time.Date(2024, 1, 15, 15, 0, 10, 0, time.UTC),
			},
		},
		Failed: []FailedMove{
			{
				MoveID:    "move-2",
				Timestamp: time.Date(2024, 1, 15, 15, 0, 20, 0, time.UTC),
				Error:     "destination already exists",
			},
		},
		Skipped: []SkippedMove{
			{
				MoveID:    "move-3",
				Timestamp: time.Date(2024, 1, 15, 15, 0, 30, 0, time.UTC),
				Reason:    "source file no longer exists",
			},
		},
	}
	
	// Format the execution log
	output := reporter.FormatExecutionLog(execLog)
	
	// Verify output contains expected sections
	expectedSections := []string{
		"EXECUTION REPORT",
		"Plan ID: test-plan-456",
		"Status: PARTIAL",
		"Started: 2024-01-15 15:00:00",
		"SUMMARY",
		"Total operations: 3",
		"✓ Completed: 1",
		"✗ Failed: 1",
		"⚠ Skipped: 1",
		"COMPLETED OPERATIONS",
		"✓ move-1 (completed at 15:00:10)",
		"FAILED OPERATIONS",
		"✗ move-2",
		"Error: destination already exists",
		"SKIPPED OPERATIONS",
		"⚠ move-3",
		"Reason: source file no longer exists",
	}
	
	for _, expected := range expectedSections {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain: %s", expected)
		}
	}
}

func TestReporter_FormatPlanSummary(t *testing.T) {
	reporter := NewReporter()
	
	// Test empty list
	emptyOutput := reporter.FormatPlanSummary([]*PlanSummary{})
	if !strings.Contains(emptyOutput, "No plans found") {
		t.Error("Should indicate no plans found for empty list")
	}
	
	// Create test plan summaries
	summaries := []*PlanSummary{
		{
			ID:        "plan-1",
			Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Status:    "completed",
			FileCount: 50,
			MoveCount: 25,
		},
		{
			ID:        "plan-2", 
			Timestamp: time.Date(2024, 1, 14, 9, 0, 0, 0, time.UTC),
			Status:    "pending",
			FileCount: 100,
			MoveCount: 75,
		},
	}
	
	// Format the summaries
	output := reporter.FormatPlanSummary(summaries)
	
	// Verify output contains expected content
	expectedContent := []string{
		"SAVED PLANS",
		"Plan ID",
		"Created",
		"Status", 
		"Files",
		"Moves",
		"plan-1",
		"2024-01-15",
		"completed",
		"50",
		"25",
		"plan-2",
		"2024-01-14", 
		"pending",
		"100",
		"75",
		"curator show-plan",
		"curator apply",
	}
	
	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain: %s", expected)
		}
	}
}

func TestReporter_FormatExecutionHistory(t *testing.T) {
	reporter := NewReporter()
	
	// Test empty list
	emptyOutput := reporter.FormatExecutionHistory([]*ExecutionLog{})
	if !strings.Contains(emptyOutput, "No execution history found") {
		t.Error("Should indicate no execution history for empty list")
	}
	
	// Create test execution logs
	logs := []*ExecutionLog{
		{
			PlanID:    "plan-1",
			Timestamp: time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
			Status:    StatusCompleted,
			Completed: []CompletedMove{{MoveID: "move-1"}, {MoveID: "move-2"}},
			Failed:    []FailedMove{},
			Skipped:   []SkippedMove{},
		},
		{
			PlanID:    "plan-2",
			Timestamp: time.Date(2024, 1, 14, 14, 0, 0, 0, time.UTC),
			Status:    StatusPartial,
			Completed: []CompletedMove{{MoveID: "move-1"}},
			Failed:    []FailedMove{{MoveID: "move-2"}},
			Skipped:   []SkippedMove{{MoveID: "move-3"}},
		},
	}
	
	// Format the execution history
	output := reporter.FormatExecutionHistory(logs)
	
	// Verify output contains expected content
	expectedContent := []string{
		"EXECUTION HISTORY",
		"Plan ID",
		"Started",
		"Status",
		"Success", 
		"Failed",
		"Skipped",
		"plan-1",
		"2024-01-15",
		"COMPLETED",
		"2",
		"0",
		"0",
		"plan-2",
		"2024-01-14",
		"PARTIAL",
		"1",
		"1",
		"1",
		"curator status",
	}
	
	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain: %s", expected)
		}
	}
}

func TestReporter_FormatDuplicationReport(t *testing.T) {
	reporter := NewReporter()
	
	// Create test duplication report
	report := &DuplicationReport{
		ID:        "dup-report-123",
		Timestamp: time.Date(2024, 1, 15, 16, 0, 0, 0, time.UTC),
		Duplicates: []DuplicateGroup{
			{
				Hash:  "abc123",
				Files: []string{"/file1.txt", "/copy/file1_copy.txt"},
				Size:  1024,
			},
			{
				Hash:  "def456",
				Files: []string{"/image.jpg", "/backup/image.jpg", "/archive/image.jpg"},
				Size:  2048000,
			},
		},
		Summary: DuplicationSummary{
			TotalDuplicates: 3,
			SpaceSaved:      2049024,
		},
	}
	
	// Format the duplication report
	output := reporter.FormatDuplicationReport(report)
	
	// Verify output contains expected content
	expectedContent := []string{
		"DUPLICATE FILES REPORT",
		"Report ID: dup-report-123",
		"Generated: 2024-01-15 16:00:00",
		"SUMMARY",
		"Duplicate files found: 3",
		"Space that can be saved:",
		"DUPLICATE GROUPS",
		"1. Files (1.0 KB each):",
		"- /file1.txt",
		"- /copy/file1_copy.txt",
		"2. Files (2.0 MB each):",
		"- /image.jpg",
		"- /backup/image.jpg",
		"- /archive/image.jpg",
	}
	
	for _, expected := range expectedContent {
		if !strings.Contains(output, expected) {
			t.Errorf("Output should contain: %s", expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{2147483648, "2.0 GB"},
	}
	
	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestReporter_FormatMove(t *testing.T) {
	reporter := NewReporter()
	
	tests := []struct {
		move     Move
		contains []string
	}{
		{
			move: Move{
				ID:          "move-1",
				Source:      "",
				Destination: "/Documents",
				Reason:      "Create organized folder",
				Type:        CreateFolder,
				FileCount:   0,
			},
			contains: []string{"CREATE FOLDER: /Documents", "Create organized folder"},
		},
		{
			move: Move{
				ID:          "move-2",
				Source:      "/file.txt",
				Destination: "/Documents/file.txt",
				Reason:      "Move to organized location",
				Type:        FileMove,
				FileCount:   1,
			},
			contains: []string{"MOVE: /file.txt → /Documents/file.txt", "Move to organized location"},
		},
		{
			move: Move{
				ID:          "move-3",
				Source:      "/folder",
				Destination: "/Documents/folder",
				Reason:      "Consolidate folder",
				Type:        FolderMove,
				FileCount:   5,
			},
			contains: []string{"MOVE: /folder → /Documents/folder", "Affects: 5 files", "Consolidate folder"},
		},
	}
	
	for _, test := range tests {
		result := reporter.formatMove(test.move)
		for _, expected := range test.contains {
			if !strings.Contains(result, expected) {
				t.Errorf("formatMove result should contain %q, got: %s", expected, result)
			}
		}
	}
}