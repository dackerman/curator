package curator

import (
	"fmt"
	"strings"
	"time"
)

// Reporter handles formatting and displaying plans and execution results
type Reporter struct{}

// NewReporter creates a new reporter
func NewReporter() *Reporter {
	return &Reporter{}
}

// FormatReorganizationPlan formats a reorganization plan for display
func (r *Reporter) FormatReorganizationPlan(plan *ReorganizationPlan) string {
	var sb strings.Builder
	
	// Header
	sb.WriteString("REORGANIZATION PLAN\n")
	sb.WriteString("==================\n")
	sb.WriteString(fmt.Sprintf("Plan ID: %s\n", plan.ID))
	
	// Count analysis
	totalFiles := 0
	totalFolders := 0
	for _, move := range plan.Moves {
		if move.Type == FileMove {
			totalFiles++
		} else if move.Type == CreateFolder {
			totalFolders++
		}
	}
	
	sb.WriteString(fmt.Sprintf("Analyzed: %d moves planned\n", len(plan.Moves)))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", plan.Timestamp.Format("2006-01-02 15:04:05")))
	
	// Summary
	sb.WriteString("SUMMARY\n")
	sb.WriteString("-------\n")
	sb.WriteString(fmt.Sprintf("✓ Create %d new folders\n", plan.Summary.FoldersCreated))
	sb.WriteString(fmt.Sprintf("✓ Move %d files\n", plan.Summary.FilesMoved))
	if plan.Summary.FoldersMovedDeduplicated > 0 {
		sb.WriteString(fmt.Sprintf("✓ Consolidate %d duplicate folders\n", plan.Summary.FoldersMovedDeduplicated))
	}
	if plan.Summary.DepthReduction != "" {
		sb.WriteString(fmt.Sprintf("✓ %s reduction in folder depth\n", plan.Summary.DepthReduction))
	}
	if plan.Summary.OrganizationImprovement != "" {
		sb.WriteString(fmt.Sprintf("✓ %s\n", plan.Summary.OrganizationImprovement))
	}
	sb.WriteString("\n")
	
	// Rationale
	if plan.Rationale != "" {
		sb.WriteString("RATIONALE\n")
		sb.WriteString("---------\n")
		sb.WriteString(fmt.Sprintf("%s\n\n", plan.Rationale))
	}
	
	// Detailed operations
	sb.WriteString(fmt.Sprintf("DETAILED OPERATIONS (showing %d operations)\n", len(plan.Moves)))
	sb.WriteString(strings.Repeat("-", 50) + "\n")
	
	for i, move := range plan.Moves {
		sb.WriteString(fmt.Sprintf("%d. %s", i+1, r.formatMove(move)))
		if i < len(plan.Moves)-1 {
			sb.WriteString("\n")
		}
	}
	
	sb.WriteString("\n\n")
	sb.WriteString(fmt.Sprintf("Type 'curator apply %s' to execute this plan\n", plan.ID))
	sb.WriteString(fmt.Sprintf("Type 'curator show-plan %s' to view details again\n", plan.ID))
	
	return sb.String()
}

// formatMove formats a single move operation
func (r *Reporter) formatMove(move Move) string {
	switch move.Type {
	case CreateFolder:
		return fmt.Sprintf("CREATE FOLDER: %s\n   → %s\n", move.Destination, move.Reason)
	case FileMove:
		return fmt.Sprintf("MOVE: %s → %s\n   → %s\n", move.Source, move.Destination, move.Reason)
	case FolderMove:
		fileInfo := ""
		if move.FileCount > 0 {
			fileInfo = fmt.Sprintf("\n   → Affects: %d files", move.FileCount)
		}
		return fmt.Sprintf("MOVE: %s → %s%s\n   → %s\n", move.Source, move.Destination, fileInfo, move.Reason)
	default:
		return fmt.Sprintf("UNKNOWN: %s\n", move.Reason)
	}
}

// FormatExecutionLog formats an execution log for display
func (r *Reporter) FormatExecutionLog(log *ExecutionLog) string {
	var sb strings.Builder
	
	sb.WriteString("EXECUTION REPORT\n")
	sb.WriteString("================\n")
	sb.WriteString(fmt.Sprintf("Plan ID: %s\n", log.PlanID))
	sb.WriteString(fmt.Sprintf("Status: %s\n", log.Status))
	sb.WriteString(fmt.Sprintf("Started: %s\n\n", log.Timestamp.Format("2006-01-02 15:04:05")))
	
	// Summary
	total := len(log.Completed) + len(log.Failed) + len(log.Skipped)
	sb.WriteString("SUMMARY\n")
	sb.WriteString("-------\n")
	sb.WriteString(fmt.Sprintf("Total operations: %d\n", total))
	sb.WriteString(fmt.Sprintf("✓ Completed: %d\n", len(log.Completed)))
	if len(log.Failed) > 0 {
		sb.WriteString(fmt.Sprintf("✗ Failed: %d\n", len(log.Failed)))
	}
	if len(log.Skipped) > 0 {
		sb.WriteString(fmt.Sprintf("⚠ Skipped: %d\n", len(log.Skipped)))
	}
	sb.WriteString("\n")
	
	// Completed operations
	if len(log.Completed) > 0 {
		sb.WriteString("COMPLETED OPERATIONS\n")
		sb.WriteString("-------------------\n")
		for _, completed := range log.Completed {
			sb.WriteString(fmt.Sprintf("✓ %s (completed at %s)\n", 
				completed.MoveID, completed.Timestamp.Format("15:04:05")))
		}
		sb.WriteString("\n")
	}
	
	// Failed operations
	if len(log.Failed) > 0 {
		sb.WriteString("FAILED OPERATIONS\n")
		sb.WriteString("-----------------\n")
		for _, failed := range log.Failed {
			sb.WriteString(fmt.Sprintf("✗ %s\n", failed.MoveID))
			sb.WriteString(fmt.Sprintf("   Error: %s\n", failed.Error))
			sb.WriteString(fmt.Sprintf("   Time: %s\n", failed.Timestamp.Format("15:04:05")))
		}
		sb.WriteString("\n")
	}
	
	// Skipped operations
	if len(log.Skipped) > 0 {
		sb.WriteString("SKIPPED OPERATIONS\n")
		sb.WriteString("------------------\n")
		for _, skipped := range log.Skipped {
			sb.WriteString(fmt.Sprintf("⚠ %s\n", skipped.MoveID))
			sb.WriteString(fmt.Sprintf("   Reason: %s\n", skipped.Reason))
			sb.WriteString(fmt.Sprintf("   Time: %s\n", skipped.Timestamp.Format("15:04:05")))
		}
		sb.WriteString("\n")
	}
	
	return sb.String()
}

// FormatPlanSummary formats a list of plan summaries
func (r *Reporter) FormatPlanSummary(summaries []*PlanSummary) string {
	if len(summaries) == 0 {
		return "No plans found.\n"
	}
	
	var sb strings.Builder
	
	sb.WriteString("SAVED PLANS\n")
	sb.WriteString("===========\n\n")
	
	// Header
	sb.WriteString(fmt.Sprintf("%-25s %-12s %-8s %-8s %s\n", 
		"Plan ID", "Created", "Status", "Files", "Moves"))
	sb.WriteString(strings.Repeat("-", 70) + "\n")
	
	// Plans
	for _, summary := range summaries {
		createdStr := summary.Timestamp.Format("2006-01-02")
		sb.WriteString(fmt.Sprintf("%-25s %-12s %-8s %-8d %d\n",
			summary.ID, createdStr, summary.Status, 
			summary.FileCount, summary.MoveCount))
	}
	
	sb.WriteString("\n")
	sb.WriteString("Use 'curator show-plan <plan-id>' to view details\n")
	sb.WriteString("Use 'curator apply <plan-id>' to execute a plan\n")
	
	return sb.String()
}

// FormatExecutionHistory formats execution history
func (r *Reporter) FormatExecutionHistory(logs []*ExecutionLog) string {
	if len(logs) == 0 {
		return "No execution history found.\n"
	}
	
	var sb strings.Builder
	
	sb.WriteString("EXECUTION HISTORY\n")
	sb.WriteString("=================\n\n")
	
	// Header
	sb.WriteString(fmt.Sprintf("%-25s %-12s %-12s %-8s %-8s %s\n", 
		"Plan ID", "Started", "Status", "Success", "Failed", "Skipped"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	
	// History entries
	for _, log := range logs {
		startedStr := log.Timestamp.Format("2006-01-02")
		sb.WriteString(fmt.Sprintf("%-25s %-12s %-12s %-8d %-8d %d\n",
			log.PlanID, startedStr, log.Status,
			len(log.Completed), len(log.Failed), len(log.Skipped)))
	}
	
	sb.WriteString("\n")
	sb.WriteString("Use 'curator status <plan-id>' to view detailed execution report\n")
	
	return sb.String()
}

// FormatDuplicationReport formats a duplication report
func (r *Reporter) FormatDuplicationReport(report *DuplicationReport) string {
	var sb strings.Builder
	
	sb.WriteString("DUPLICATE FILES REPORT\n")
	sb.WriteString("======================\n")
	sb.WriteString(fmt.Sprintf("Report ID: %s\n", report.ID))
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))
	
	// Summary
	sb.WriteString("SUMMARY\n")
	sb.WriteString("-------\n")
	sb.WriteString(fmt.Sprintf("Duplicate files found: %d\n", report.Summary.TotalDuplicates))
	sb.WriteString(fmt.Sprintf("Space that can be saved: %s\n\n", formatBytes(report.Summary.SpaceSaved)))
	
	// Duplicate groups
	if len(report.Duplicates) > 0 {
		sb.WriteString("DUPLICATE GROUPS\n")
		sb.WriteString("----------------\n")
		for i, group := range report.Duplicates {
			sb.WriteString(fmt.Sprintf("%d. Files (%s each):\n", i+1, formatBytes(group.Size)))
			for _, file := range group.Files {
				sb.WriteString(fmt.Sprintf("   - %s\n", file))
			}
			sb.WriteString("\n")
		}
	}
	
	return sb.String()
}

// formatBytes formats byte count in human-readable format
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