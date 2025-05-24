package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dackerman/curator"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "curator",
	Short: "AI-powered file system organizer",
	Long: `Curator uses AI to intelligently reorganize file systems by analyzing
file structures, proposing reorganization plans with explanations,
and executing approved changes while maintaining a complete audit trail.`,
}

var reorganizeCmd = &cobra.Command{
	Use:   "reorganize",
	Short: "Analyze and generate a reorganization plan",
	Long:  `Scans the filesystem, analyzes structure, and generates a reorganization plan`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		exclude, _ := cmd.Flags().GetString("exclude")
		path, _ := cmd.Flags().GetString("path")
		
		if path == "" {
			path = "."
		}
		
		// Create components
		filesystem := createSampleFilesystem()
		store := curator.NewMemoryOperationStore()
		analyzer := curator.NewMockAIAnalyzer()
		reporter := curator.NewReporter()
		
		// List files in the filesystem
		files, err := filesystem.List(path)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}
		
		fmt.Printf("Analyzing %d files in %s...\n", len(files), path)
		
		// Generate reorganization plan
		plan, err := analyzer.AnalyzeForReorganization(files)
		if err != nil {
			return fmt.Errorf("failed to analyze for reorganization: %w", err)
		}
		
		// Save the plan
		if err := store.SavePlan(plan); err != nil {
			return fmt.Errorf("failed to save plan: %w", err)
		}
		
		// Display the plan
		fmt.Println(reporter.FormatReorganizationPlan(plan))
		
		if !dryRun {
			fmt.Printf("Plan saved with ID: %s\n", plan.ID)
		}
		
		_ = exclude // TODO: implement exclude patterns
		return nil
	},
}

var listPlansCmd = &cobra.Command{
	Use:   "list-plans",
	Short: "List all reorganization plans",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := getStore()
		reporter := curator.NewReporter()
		
		summaries, err := store.ListPlans()
		if err != nil {
			return fmt.Errorf("failed to list plans: %w", err)
		}
		
		fmt.Print(reporter.FormatPlanSummary(summaries))
		return nil
	},
}

var showPlanCmd = &cobra.Command{
	Use:   "show-plan [plan-id]",
	Short: "Show details of a specific plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		store := getStore()
		reporter := curator.NewReporter()
		
		plan, err := store.GetPlan(planID)
		if err != nil {
			return fmt.Errorf("failed to get plan: %w", err)
		}
		
		fmt.Print(reporter.FormatReorganizationPlan(plan))
		return nil
	},
}

var applyCmd = &cobra.Command{
	Use:   "apply [plan-id]",
	Short: "Execute a reorganization plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		failFast, _ := cmd.Flags().GetBool("fail-fast")
		
		filesystem := createSampleFilesystem()
		store := getStore()
		engine := curator.NewExecutionEngine(filesystem, store)
		reporter := curator.NewReporter()
		
		fmt.Printf("Executing plan %s...\n", planID)
		
		execLog, err := engine.ExecutePlan(planID, failFast)
		if err != nil {
			return fmt.Errorf("failed to execute plan: %w", err)
		}
		
		fmt.Print(reporter.FormatExecutionLog(execLog))
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [plan-id]",
	Short: "Check the status of a plan execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		
		filesystem := createSampleFilesystem()
		store := getStore()
		engine := curator.NewExecutionEngine(filesystem, store)
		reporter := curator.NewReporter()
		
		execLog, err := engine.GetExecutionStatus(planID)
		if err != nil {
			return fmt.Errorf("failed to get execution status: %w", err)
		}
		
		fmt.Print(reporter.FormatExecutionLog(execLog))
		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show execution history",
	RunE: func(cmd *cobra.Command, args []string) error {
		store := getStore()
		reporter := curator.NewReporter()
		
		logs, err := store.GetExecutionHistory()
		if err != nil {
			return fmt.Errorf("failed to get execution history: %w", err)
		}
		
		fmt.Print(reporter.FormatExecutionHistory(logs))
		return nil
	},
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback [plan-id]",
	Short: "Rollback a previously executed plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		fmt.Printf("Rolling back plan %s - not yet implemented\n", planID)
		return nil
	},
}

var deduplicateCmd = &cobra.Command{
	Use:   "deduplicate",
	Short: "Find and remove duplicate files",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		fmt.Printf("Finding duplicates (dry-run: %v) - not yet implemented\n", dryRun)
		return nil
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up junk and unnecessary files",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		fmt.Printf("Cleaning up files (dry-run: %v) - not yet implemented\n", dryRun)
		return nil
	},
}

var renameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Standardize file naming conventions",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		pattern, _ := cmd.Flags().GetString("pattern")
		fmt.Printf("Renaming files (dry-run: %v, pattern: %s) - not yet implemented\n", dryRun, pattern)
		return nil
	},
}

func init() {
	// Global flags
	reorganizeCmd.Flags().Bool("dry-run", false, "Generate plan without executing")
	reorganizeCmd.Flags().String("exclude", "", "Comma-separated list of patterns to exclude")
	reorganizeCmd.Flags().String("path", "", "Path to analyze (defaults to current directory)")
	
	applyCmd.Flags().Bool("fail-fast", false, "Stop on first error")
	
	deduplicateCmd.Flags().Bool("dry-run", false, "Show duplicates without removing")
	cleanupCmd.Flags().Bool("dry-run", false, "Show cleanup plan without executing")
	renameCmd.Flags().Bool("dry-run", false, "Show rename plan without executing")
	renameCmd.Flags().String("pattern", "consistent-naming", "Naming pattern to use")
	
	// Add commands to root
	rootCmd.AddCommand(reorganizeCmd)
	rootCmd.AddCommand(listPlansCmd)
	rootCmd.AddCommand(showPlanCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.AddCommand(rollbackCmd)
	rootCmd.AddCommand(deduplicateCmd)
	rootCmd.AddCommand(cleanupCmd)
	rootCmd.AddCommand(renameCmd)
}

// Global store instance for CLI
var globalStore *curator.MemoryOperationStore

// getStore returns a global store instance
func getStore() *curator.MemoryOperationStore {
	if globalStore == nil {
		globalStore = curator.NewMemoryOperationStore()
	}
	return globalStore
}

// createSampleFilesystem creates a sample filesystem for demonstration
func createSampleFilesystem() *curator.MemoryFileSystem {
	fs := curator.NewMemoryFileSystem()
	
	// Add sample files to demonstrate the functionality
	fs.AddFile("/document1.pdf", []byte("Sample PDF content"), "application/pdf")
	fs.AddFile("/photo1.jpg", []byte("JPEG data"), "image/jpeg")
	fs.AddFile("/video1.mp4", []byte("MP4 data"), "video/mp4")
	fs.AddFile("/random/document2.docx", []byte("Word document"), "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	fs.AddFile("/downloads/photo2.png", []byte("PNG data"), "image/png")
	fs.AddFile("/temp/cache.tmp", []byte("temporary file"), "text/plain")
	fs.AddFile("/desktop/My Document.txt", []byte("text content"), "text/plain")
	fs.AddFile("/desktop/Another File.pdf", []byte("another PDF"), "application/pdf")
	
	return fs
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}