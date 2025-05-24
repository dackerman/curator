package curator

import (
	"strings"
	"testing"
	"time"
)

func TestReporter_FormatReorganizationPlan(t *testing.T) {
	reporter := NewReporter()
	
	plan := &ReorganizationPlan{
		ID:        "test-plan-123",
		Timestamp: time.Date(2024, 1, 15, 14, 30, 52, 0, time.UTC),
		Moves: []Move{
			{
				ID:          "move-1",
				Source:      "",
				Destination: "Documents",
				Reason:      "Create Documents folder for organization",
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
			FoldersCreated:          1,
			FilesMoved:              1,
			OrganizationImprovement: "100% of files will be organized",
		},
		Rationale: "Organize files by type for better structure",
	}
	
	output := reporter.FormatReorganizationPlan(plan)
	
	// Check that output contains expected sections
	if !strings.Contains(output, "REORGANIZATION PLAN") {
		t.Error("Output should contain plan header")
	}
	
	if !strings.Contains(output, "test-plan-123") {
		t.Error("Output should contain plan ID")
	}
	
	if !strings.Contains(output, "2024-01-15 14:30:52") {
		t.Error("Output should contain formatted timestamp")
	}
	
	if !strings.Contains(output, "SUMMARY") {
		t.Error("Output should contain summary section")
	}
	
	if !strings.Contains(output, "Create 1 new folders") {
		t.Error("Output should show folders created")
	}
	
	if !strings.Contains(output, "Move 1 files") {
		t.Error("Output should show files moved")
	}
	
	if !strings.Contains(output, "DETAILED OPERATIONS") {
		t.Error("Output should contain operations section")
	}
	
	if !strings.Contains(output, "CREATE FOLDER: Documents") {
		t.Error("Output should show folder creation")
	}
	
	if !strings.Contains(output, "MOVE: /document.pdf → Documents/document.pdf") {
		t.Error("Output should show file move")
	}
	
	if !strings.Contains(output, "curator apply test-plan-123") {
		t.Error("Output should contain apply command")
	}
}

func TestReporter_FormatExecutionLog(t *testing.T) {
	reporter := NewReporter()
	
	log := &ExecutionLog{
		PlanID:    "test-plan-456",
		Timestamp: time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
		Status:    StatusCompleted,
		Completed: []CompletedMove{
			{MoveID: "move-1", Timestamp: time.Now()},
			{MoveID: "move-2", Timestamp: time.Now()},
		},
		Failed:  []FailedMove{},
		Skipped: []SkippedMove{},
	}
	
	output := reporter.FormatExecutionLog(log)
	
	// Check that output contains expected sections
	if !strings.Contains(output, "EXECUTION REPORT") {
		t.Error("Output should contain execution header")
	}
	
	if !strings.Contains(output, "test-plan-456") {
		t.Error("Output should contain plan ID")
	}
	
	if !strings.Contains(output, "✅ COMPLETED") {
		t.Error("Output should show completed status")
	}
	
	if !strings.Contains(output, "Completed: 2 operations") {
		t.Error("Output should show completed count")
	}
	
	if !strings.Contains(output, "🎉 Plan executed successfully") {
		t.Error("Output should show success message")
	}
}

func TestReporter_FormatExecutionLog_WithFailures(t *testing.T) {
	reporter := NewReporter()
	
	log := &ExecutionLog{
		PlanID:    "test-plan-failed",
		Timestamp: time.Now(),
		Status:    StatusPartial,
		Completed: []CompletedMove{
			{MoveID: "move-1", Timestamp: time.Now()},
		},
		Failed: []FailedMove{
			{MoveID: "move-2", Timestamp: time.Now(), Error: "File not found"},
		},
		Skipped: []SkippedMove{
			{MoveID: "move-3", Timestamp: time.Now(), Reason: "Destination exists"},
		},
	}
	
	output := reporter.FormatExecutionLog(log)
	
	if !strings.Contains(output, "⚠️ PARTIALLY COMPLETED") {
		t.Error("Output should show partial status")
	}
	
	if !strings.Contains(output, "Completed: 1 operations") {
		t.Error("Output should show completed count")
	}
	
	if !strings.Contains(output, "Failed: 1 operations") {
		t.Error("Output should show failed count")
	}
	
	if !strings.Contains(output, "Skipped: 1 operations") {
		t.Error("Output should show skipped count")
	}
	
	if !strings.Contains(output, "FAILED OPERATIONS") {
		t.Error("Output should show failed operations section")
	}
	
	if !strings.Contains(output, "move-2: File not found") {
		t.Error("Output should show failed operation details")
	}
	
	if !strings.Contains(output, "SKIPPED OPERATIONS") {
		t.Error("Output should show skipped operations section")
	}
	
	if !strings.Contains(output, "move-3: Destination exists") {
		t.Error("Output should show skipped operation details")
	}
	
	if !strings.Contains(output, "⚠️  Plan partially executed") {
		t.Error("Output should show partial completion message")
	}
}

func TestReporter_FormatPlanSummaries(t *testing.T) {
	reporter := NewReporter()
	
	summaries := []*PlanSummary{
		{
			ID:        "plan-1",
			Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Status:    "completed",
			FileCount: 10,
			MoveCount: 15,
		},
		{
			ID:        "plan-2",
			Timestamp: time.Date(2024, 1, 14, 10, 0, 0, 0, time.UTC),
			Status:    "pending",
			FileCount: 5,
			MoveCount: 8,
		},
	}
	
	output := reporter.FormatPlanSummaries(summaries)
	
	if !strings.Contains(output, "REORGANIZATION PLANS") {
		t.Error("Output should contain plans header")
	}
	
	if !strings.Contains(output, "plan-1") {
		t.Error("Output should contain first plan ID")
	}
	
	if !strings.Contains(output, "plan-2") {
		t.Error("Output should contain second plan ID")
	}
	
	if !strings.Contains(output, "2024-01-15 10:00:00") {
		t.Error("Output should contain formatted timestamp")
	}
	
	if !strings.Contains(output, "Status:  completed") {
		t.Error("Output should show plan status")
	}
	
	if !strings.Contains(output, "Files:   10 files, 15 operations") {
		t.Error("Output should show file and operation counts")
	}
	
	if !strings.Contains(output, "curator show-plan") {
		t.Error("Output should contain usage instructions")
	}
}

func TestReporter_FormatPlanSummaries_Empty(t *testing.T) {
	reporter := NewReporter()
	
	output := reporter.FormatPlanSummaries([]*PlanSummary{})
	
	expected := "No reorganization plans found.\n"
	if output != expected {
		t.Errorf("Expected empty message, got: %s", output)
	}
}

func TestReporter_FormatDuplicationReport(t *testing.T) {
	reporter := NewReporter()
	
	report := &DuplicationReport{
		ID:        "dup-123",
		Timestamp: time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Duplicates: []DuplicateGroup{
			{
				Hash:  "abc123",
				Files: []string{"/file1.txt", "/copy/file1.txt"},
				Size:  1024,
			},
		},
		Summary: DuplicationSummary{
			TotalDuplicates: 1,
			SpaceSaved:      1024,
		},
	}
	
	output := reporter.FormatDuplicationReport(report)
	
	if !strings.Contains(output, "DUPLICATION REPORT") {
		t.Error("Output should contain duplication header")
	}
	
	if !strings.Contains(output, "dup-123") {
		t.Error("Output should contain report ID")
	}
	
	if !strings.Contains(output, "Total duplicate files: 1") {
		t.Error("Output should show total duplicates")
	}
	
	if !strings.Contains(output, "Space that could be saved: 1.0 KB") {
		t.Error("Output should show space savings")
	}
	
	if !strings.Contains(output, "DUPLICATE GROUPS") {
		t.Error("Output should contain groups section")
	}
	
	if !strings.Contains(output, "/file1.txt") {
		t.Error("Output should show duplicate files")
	}
}

func TestReporter_FormatDuplicationReport_NoDuplicates(t *testing.T) {
	reporter := NewReporter()
	
	report := &DuplicationReport{
		ID:         "dup-empty",
		Timestamp:  time.Now(),
		Duplicates: []DuplicateGroup{},
		Summary: DuplicationSummary{
			TotalDuplicates: 0,
			SpaceSaved:      0,
		},
	}
	
	output := reporter.FormatDuplicationReport(report)
	
	if !strings.Contains(output, "🎉 No duplicate files found!") {
		t.Error("Output should show no duplicates message")
	}
}

func TestReporter_FormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	
	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}