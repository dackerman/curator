package curator

import (
	"fmt"
	"strings"
)

// Reporter handles generating text-based reports for plans and execution results
type Reporter struct{}

// NewReporter creates a new reporter
func NewReporter() *Reporter {
	return &Reporter{}
}

// FormatReorganizationPlan formats a reorganization plan as text
func (r *Reporter) FormatReorganizationPlan(plan *ReorganizationPlan) string {
	var b strings.Builder
	
	// Header
	b.WriteString("REORGANIZATION PLAN\n")
	b.WriteString("==================\n")
	b.WriteString(fmt.Sprintf("Plan ID: %s\n", plan.ID))
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", plan.Timestamp.Format("2006-01-02 15:04:05")))
	
	// Summary
	b.WriteString("SUMMARY\n")
	b.WriteString("-------\n")
	b.WriteString(fmt.Sprintf("✓ Create %d new folders\n", plan.Summary.FoldersCreated))
	b.WriteString(fmt.Sprintf("✓ Move %d files\n", plan.Summary.FilesMoved))
	if plan.Summary.FoldersMovedDeduplicated > 0 {
		b.WriteString(fmt.Sprintf("✓ Consolidate %d folders\n", plan.Summary.FoldersMovedDeduplicated))
	}
	if plan.Summary.DepthReduction != "" {
		b.WriteString(fmt.Sprintf("✓ %s reduction in folder depth\n", plan.Summary.DepthReduction))
	}
	b.WriteString(fmt.Sprintf("✓ %s\n\n", plan.Summary.OrganizationImprovement))
	
	// Rationale
	if plan.Rationale != "" {
		b.WriteString("RATIONALE\n")
		b.WriteString("---------\n")
		b.WriteString(fmt.Sprintf("%s\n\n", plan.Rationale))
	}
	
	// Operations
	b.WriteString(fmt.Sprintf("DETAILED OPERATIONS (%d total)\n", len(plan.Moves)))
	b.WriteString(strings.Repeat("-", 40) + "\n")
	
	createFolderCount := 0
	moveCount := 0
	
	for i, move := range plan.Moves {
		if i >= 10 && len(plan.Moves) > 10 {
			b.WriteString(fmt.Sprintf("\n[... %d more operations ...]\n", len(plan.Moves)-10))
			break
		}
		
		switch move.Type {
		case CreateFolder:
			createFolderCount++
			b.WriteString(fmt.Sprintf("%d. CREATE FOLDER: %s\n", i+1, move.Destination))
			b.WriteString(fmt.Sprintf("   → %s\n\n", move.Reason))
			
		case FileMove, FolderMove:
			moveCount++
			actionType := "MOVE"
			if move.Type == FolderMove {
				actionType = "MOVE FOLDER"
			}
			
			b.WriteString(fmt.Sprintf("%d. %s: %s → %s\n", i+1, actionType, move.Source, move.Destination))
			if move.FileCount > 1 {
				b.WriteString(fmt.Sprintf("   → Affects: %d files\n", move.FileCount))
			}
			b.WriteString(fmt.Sprintf("   → %s\n\n", move.Reason))
		}
	}
	
	// Instructions
	b.WriteString(fmt.Sprintf("Type 'curator apply %s' to execute this plan\n", plan.ID))
	b.WriteString(fmt.Sprintf("Type 'curator show-plan %s' to view this plan again\n", plan.ID))
	
	return b.String()
}

// FormatExecutionLog formats an execution log as text
func (r *Reporter) FormatExecutionLog(log *ExecutionLog) string {
	var b strings.Builder
	
	// Header
	b.WriteString("EXECUTION REPORT\n")
	b.WriteString("================\n")
	b.WriteString(fmt.Sprintf("Plan ID: %s\n", log.PlanID))
	b.WriteString(fmt.Sprintf("Executed: %s\n", log.Timestamp.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("Status: %s\n\n", formatStatus(log.Status)))
	
	// Summary
	total := len(log.Completed) + len(log.Failed) + len(log.Skipped)
	b.WriteString("SUMMARY\n")
	b.WriteString("-------\n")
	b.WriteString(fmt.Sprintf("✓ Completed: %d operations\n", len(log.Completed)))
	if len(log.Failed) > 0 {
		b.WriteString(fmt.Sprintf("✗ Failed: %d operations\n", len(log.Failed)))
	}
	if len(log.Skipped) > 0 {
		b.WriteString(fmt.Sprintf("⚠ Skipped: %d operations\n", len(log.Skipped)))
	}
	b.WriteString(fmt.Sprintf("📊 Total: %d operations\n\n", total))
	
	// Details for failed operations
	if len(log.Failed) > 0 {
		b.WriteString("FAILED OPERATIONS\n")
		b.WriteString("-----------------\n")
		for _, failed := range log.Failed {
			b.WriteString(fmt.Sprintf("• %s: %s\n", failed.MoveID, failed.Error))
		}
		b.WriteString("\n")
	}
	
	// Details for skipped operations
	if len(log.Skipped) > 0 {
		b.WriteString("SKIPPED OPERATIONS\n")
		b.WriteString("------------------\n")
		for _, skipped := range log.Skipped {
			b.WriteString(fmt.Sprintf("• %s: %s\n", skipped.MoveID, skipped.Reason))
		}
		b.WriteString("\n")
	}
	
	// Success message or next steps
	switch log.Status {
	case StatusCompleted:
		b.WriteString("🎉 Plan executed successfully! All operations completed.\n")
	case StatusPartial:
		b.WriteString("⚠️  Plan partially executed. Some operations failed or were skipped.\n")
		b.WriteString("You may want to review the failed operations and retry manually.\n")
	case StatusFailed:
		b.WriteString("❌ Plan execution failed. No operations were completed successfully.\n")
		b.WriteString("Please review the errors and fix any issues before retrying.\n")
	case StatusInProgress:
		b.WriteString("🔄 Plan execution is still in progress.\n")
	}
	
	return b.String()
}

// FormatPlanSummaries formats a list of plan summaries
func (r *Reporter) FormatPlanSummaries(summaries []*PlanSummary) string {
	var b strings.Builder
	
	if len(summaries) == 0 {
		return "No reorganization plans found.\n"
	}
	
	b.WriteString("REORGANIZATION PLANS\n")
	b.WriteString("====================\n\n")
	
	for _, summary := range summaries {
		b.WriteString(fmt.Sprintf("Plan ID: %s\n", summary.ID))
		b.WriteString(fmt.Sprintf("Created: %s\n", summary.Timestamp.Format("2006-01-02 15:04:05")))
		b.WriteString(fmt.Sprintf("Status:  %s\n", summary.Status))
		b.WriteString(fmt.Sprintf("Files:   %d files, %d operations\n", summary.FileCount, summary.MoveCount))
		b.WriteString(strings.Repeat("-", 40) + "\n\n")
	}
	
	b.WriteString(fmt.Sprintf("Use 'curator show-plan <plan-id>' to view details\n"))
	b.WriteString(fmt.Sprintf("Use 'curator apply <plan-id>' to execute a plan\n"))
	
	return b.String()
}

// FormatExecutionHistory formats execution history
func (r *Reporter) FormatExecutionHistory(logs []*ExecutionLog) string {
	var b strings.Builder
	
	if len(logs) == 0 {
		return "No execution history found.\n"
	}
	
	b.WriteString("EXECUTION HISTORY\n")
	b.WriteString("=================\n\n")
	
	for _, log := range logs {
		b.WriteString(fmt.Sprintf("Plan: %s\n", log.PlanID))
		b.WriteString(fmt.Sprintf("Executed: %s\n", log.Timestamp.Format("2006-01-02 15:04:05")))
		b.WriteString(fmt.Sprintf("Status: %s\n", formatStatus(log.Status)))
		
		total := len(log.Completed) + len(log.Failed) + len(log.Skipped)
		successRate := float64(len(log.Completed)) / float64(total) * 100
		b.WriteString(fmt.Sprintf("Success: %d/%d (%.1f%%)\n", len(log.Completed), total, successRate))
		
		b.WriteString(strings.Repeat("-", 40) + "\n\n")
	}
	
	return b.String()
}

// FormatDuplicationReport formats a duplication report
func (r *Reporter) FormatDuplicationReport(report *DuplicationReport) string {
	var b strings.Builder
	
	b.WriteString("DUPLICATION REPORT\n")
	b.WriteString("==================\n")
	b.WriteString(fmt.Sprintf("Report ID: %s\n", report.ID))
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))
	
	// Summary
	b.WriteString("SUMMARY\n")
	b.WriteString("-------\n")
	b.WriteString(fmt.Sprintf("• Total duplicate files: %d\n", report.Summary.TotalDuplicates))
	b.WriteString(fmt.Sprintf("• Space that could be saved: %s\n\n", formatBytes(report.Summary.SpaceSaved)))
	
	if len(report.Duplicates) == 0 {
		b.WriteString("🎉 No duplicate files found!\n")
		return b.String()
	}
	
	// Duplicate groups
	b.WriteString("DUPLICATE GROUPS\n")
	b.WriteString("----------------\n")
	
	for i, group := range report.Duplicates {
		if i >= 10 {
			b.WriteString(fmt.Sprintf("\n[... %d more duplicate groups ...]\n", len(report.Duplicates)-10))
			break
		}
		
		b.WriteString(fmt.Sprintf("Group %d (Size: %s each):\n", i+1, formatBytes(group.Size)))
		for _, file := range group.Files {
			b.WriteString(fmt.Sprintf("  • %s\n", file))
		}
		b.WriteString("\n")
	}
	
	return b.String()
}

// FormatCleanupPlan formats a cleanup plan
func (r *Reporter) FormatCleanupPlan(plan *CleanupPlan) string {
	var b strings.Builder
	
	b.WriteString("CLEANUP PLAN\n")
	b.WriteString("============\n")
	b.WriteString(fmt.Sprintf("Plan ID: %s\n", plan.ID))
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", plan.Timestamp.Format("2006-01-02 15:04:05")))
	
	// Summary
	b.WriteString("SUMMARY\n")
	b.WriteString("-------\n")
	b.WriteString(fmt.Sprintf("• Files to delete: %d\n", plan.Summary.FilesDeleted))
	b.WriteString(fmt.Sprintf("• Space to free: %s\n\n", formatBytes(plan.Summary.SpaceFreed)))
	
	if len(plan.Deletions) == 0 {
		b.WriteString("🎉 No junk files found!\n")
		return b.String()
	}
	
	// Deletions
	b.WriteString("FILES TO DELETE\n")
	b.WriteString("---------------\n")
	
	for i, deletion := range plan.Deletions {
		if i >= 20 {
			b.WriteString(fmt.Sprintf("\n[... %d more files to delete ...]\n", len(plan.Deletions)-20))
			break
		}
		
		b.WriteString(fmt.Sprintf("• %s (%s) - %s\n", deletion.Path, formatBytes(deletion.Size), deletion.Reason))
	}
	
	return b.String()
}

// Helper functions

func formatStatus(status ExecutionStatus) string {
	switch status {
	case StatusCompleted:
		return "✅ COMPLETED"
	case StatusFailed:
		return "❌ FAILED"
	case StatusPartial:
		return "⚠️ PARTIALLY COMPLETED"
	case StatusInProgress:
		return "🔄 IN PROGRESS"
	default:
		return string(status)
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}