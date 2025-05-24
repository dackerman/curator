package main

import (
	"fmt"
	"os"

	"github.com/dackerman/curator"
	"github.com/spf13/cobra"
)

// Global components
var (
	fs       curator.FileSystem
	store    curator.OperationStore
	analyzer curator.AIAnalyzer
	engine   *curator.ExecutionEngine
	reporter *curator.Reporter
	config   *curator.Config
)


func setupSampleFiles(mfs *curator.MemoryFileSystem) {
	
	// Add some sample files to demonstrate the functionality
	mfs.AddFile("/document1.pdf", []byte("Sample PDF content"), "application/pdf")
	mfs.AddFile("/image1.jpg", []byte("Sample image content"), "image/jpeg")
	mfs.AddFile("/video1.mp4", []byte("Sample video content"), "video/mp4")
	mfs.AddFile("/code.go", []byte("package main\n\nfunc main() {}"), "text/plain")
	mfs.AddFile("/temp_file.tmp", []byte("temporary"), "text/plain")
	mfs.AddFile("/My Document.pdf", []byte("Another document"), "application/pdf")
	mfs.AddFile("/Photo With Spaces.jpg", []byte("Photo content"), "image/jpeg")
	mfs.AddFile("/empty_file.txt", []byte(""), "text/plain")
	mfs.AddFile("/backup.bak", []byte("backup content"), "text/plain")
	mfs.AddFile("/Downloads/random_download.zip", []byte("zip content"), "application/zip")
}

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
		_, _ = cmd.Flags().GetString("exclude") // exclude not yet implemented
		
		// For now, always operate on root "/"
		// In the future, this could be configurable
		path := "/"
		
		// Get filesystem
		fs, err := getFileSystem(cmd)
		if err != nil {
			return fmt.Errorf("failed to create filesystem: %w", err)
		}
		
		fmt.Printf("Scanning filesystem at %s...\n", path)
		fmt.Printf("Using %s filesystem...\n", config.FileSystem.Type)
		
		// List all files
		_, err = fs.List(path)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}
		
		// Get all files recursively
		allFiles, err := getAllFilesRecursively(fs, path)
		if err != nil {
			return fmt.Errorf("failed to get all files: %w", err)
		}
		
		fmt.Printf("Found %d files to analyze...\n", len(allFiles))
		
		// Get analyzer
		analyzer, err := getAnalyzer(cmd)
		if err != nil {
			return fmt.Errorf("failed to create analyzer: %w", err)
		}
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Using %s AI provider...\n", config.AI.Provider)
		
		// Generate reorganization plan
		plan, err := analyzer.AnalyzeForReorganization(allFiles)
		if err != nil {
			return fmt.Errorf("failed to analyze files: %w", err)
		}
		
		// Save the plan
		if err := store.SavePlan(plan); err != nil {
			return fmt.Errorf("failed to save plan: %w", err)
		}
		
		// Display the plan
		fmt.Println()
		fmt.Print(reporter.FormatReorganizationPlan(plan))
		
		if !dryRun {
			fmt.Printf("\nPlan saved with ID: %s\n", plan.ID)
		}
		
		return nil
	},
}

// Helper function to get all files recursively
func getAllFilesRecursively(fs curator.FileSystem, root string) ([]curator.FileInfo, error) {
	var allFiles []curator.FileInfo
	
	var traverse func(string) error
	traverse = func(path string) error {
		files, err := fs.List(path)
		if err != nil {
			return err
		}
		
		for _, file := range files {
			allFiles = append(allFiles, file)
			if file.IsDir() {
				if err := traverse(file.Path()); err != nil {
					return err
				}
			}
		}
		return nil
	}
	
	return allFiles, traverse(root)
}

// getAnalyzer creates an analyzer based on configuration and command flags
func getAnalyzer(cmd *cobra.Command) (curator.AIAnalyzer, error) {
	// Check if provider is overridden via flag
	if provider, _ := cmd.Flags().GetString("ai-provider"); provider != "" {
		config.AI.Provider = provider
		if provider == "gemini" {
			config.AI.Gemini = curator.DefaultGeminiConfig()
			if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
				config.AI.Gemini.APIKey = apiKey
			}
		}
	}
	
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	// Create analyzer
	return config.CreateAnalyzer()
}

// getFileSystem creates a filesystem based on configuration and command flags
func getFileSystem(cmd *cobra.Command) (curator.FileSystem, error) {
	// Check if filesystem type is overridden via flag
	if fsType, _ := cmd.Flags().GetString("filesystem"); fsType != "" {
		config.FileSystem.Type = fsType
	}
	
	// Check if root path is overridden via flag
	if root, _ := cmd.Flags().GetString("root"); root != "" {
		config.FileSystem.Root = root
	}
	
	// Create filesystem
	fs, err := config.CreateFileSystem()
	if err != nil {
		return nil, err
	}
	
	// If using memory filesystem, add sample files for testing
	if config.FileSystem.Type == "memory" {
		setupSampleFiles(fs.(*curator.MemoryFileSystem))
	}
	
	return fs, nil
}

var listPlansCmd = &cobra.Command{
	Use:   "list-plans",
	Short: "List all reorganization plans",
	RunE: func(cmd *cobra.Command, args []string) error {
		summaries, err := store.ListPlans()
		if err != nil {
			return fmt.Errorf("failed to list plans: %w", err)
		}
		
		fmt.Print(reporter.FormatPlanSummaries(summaries))
		return nil
	},
}

var showPlanCmd = &cobra.Command{
	Use:   "show-plan [plan-id]",
	Short: "Show details of a specific plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
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
		
		fmt.Printf("Executing plan %s...\n", planID)
		
		// Resume any pending operations first
		if err := engine.ResumePendingOperations(); err != nil {
			fmt.Printf("Warning: failed to resume pending operations: %v\n", err)
		}
		
		// Execute the plan
		execLog, err := engine.ExecutePlan(planID, failFast)
		if err != nil {
			return fmt.Errorf("failed to execute plan: %w", err)
		}
		
		// Display execution results
		fmt.Println()
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
		fmt.Printf("Rollback functionality not yet implemented for plan %s\n", planID)
		fmt.Println("This will be implemented in a future phase.")
		return nil
	},
}

var deduplicateCmd = &cobra.Command{
	Use:   "deduplicate",
	Short: "Find and remove duplicate files",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = cmd.Flags().GetBool("dry-run") // dry-run not yet implemented for duplicates
		
		fmt.Println("Scanning for duplicate files...")
		
		// Get filesystem
		fs, err := getFileSystem(cmd)
		if err != nil {
			return fmt.Errorf("failed to create filesystem: %w", err)
		}
		
		fmt.Printf("Using %s filesystem...\n", config.FileSystem.Type)
		
		allFiles, err := getAllFilesRecursively(fs, "/")
		if err != nil {
			return fmt.Errorf("failed to get all files: %w", err)
		}
		
		// Get analyzer
		analyzer, err := getAnalyzer(cmd)
		if err != nil {
			return fmt.Errorf("failed to create analyzer: %w", err)
		}
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Using %s AI provider...\n", config.AI.Provider)
		
		report, err := analyzer.AnalyzeForDuplicates(allFiles)
		if err != nil {
			return fmt.Errorf("failed to analyze duplicates: %w", err)
		}
		
		fmt.Print(reporter.FormatDuplicationReport(report))
		return nil
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up junk and unnecessary files",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = cmd.Flags().GetBool("dry-run") // dry-run not yet implemented for cleanup
		
		fmt.Println("Scanning for junk files...")
		
		// Get filesystem
		fs, err := getFileSystem(cmd)
		if err != nil {
			return fmt.Errorf("failed to create filesystem: %w", err)
		}
		
		fmt.Printf("Using %s filesystem...\n", config.FileSystem.Type)
		
		allFiles, err := getAllFilesRecursively(fs, "/")
		if err != nil {
			return fmt.Errorf("failed to get all files: %w", err)
		}
		
		// Get analyzer
		analyzer, err := getAnalyzer(cmd)
		if err != nil {
			return fmt.Errorf("failed to create analyzer: %w", err)
		}
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Using %s AI provider...\n", config.AI.Provider)
		
		plan, err := analyzer.AnalyzeForCleanup(allFiles)
		if err != nil {
			return fmt.Errorf("failed to analyze cleanup: %w", err)
		}
		
		fmt.Print(reporter.FormatCleanupPlan(plan))
		return nil
	},
}

var renameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Standardize file naming conventions",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = cmd.Flags().GetBool("dry-run") // dry-run not yet implemented for rename
		_, _ = cmd.Flags().GetString("pattern") // pattern not yet implemented
		
		fmt.Println("Scanning for files to rename...")
		
		// Get filesystem
		fs, err := getFileSystem(cmd)
		if err != nil {
			return fmt.Errorf("failed to create filesystem: %w", err)
		}
		
		fmt.Printf("Using %s filesystem...\n", config.FileSystem.Type)
		
		allFiles, err := getAllFilesRecursively(fs, "/")
		if err != nil {
			return fmt.Errorf("failed to get all files: %w", err)
		}
		
		// Get analyzer
		analyzer, err := getAnalyzer(cmd)
		if err != nil {
			return fmt.Errorf("failed to create analyzer: %w", err)
		}
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Using %s AI provider...\n", config.AI.Provider)
		
		plan, err := analyzer.AnalyzeForRenaming(allFiles)
		if err != nil {
			return fmt.Errorf("failed to analyze renaming: %w", err)
		}
		
		// Simple text output for renaming plan
		fmt.Printf("\nRENAMING PLAN\n")
		fmt.Printf("=============\n")
		fmt.Printf("Plan ID: %s\n", plan.ID)
		fmt.Printf("Generated: %s\n\n", plan.Timestamp.Format("2006-01-02 15:04:05"))
		
		if len(plan.Renames) == 0 {
			fmt.Println("🎉 All files already follow consistent naming conventions!")
		} else {
			fmt.Printf("Files to rename: %d\n", len(plan.Renames))
			fmt.Printf("Pattern: %s\n\n", plan.Summary.Pattern)
			
			for i, rename := range plan.Renames {
				if i >= 10 {
					fmt.Printf("\n[... %d more files to rename ...]\n", len(plan.Renames)-10)
					break
				}
				fmt.Printf("• %s → %s\n", rename.OldName, rename.NewName)
			}
		}
		
		return nil
	},
}

func init() {
	// Load configuration
	config = curator.LoadConfig()
	
	// Initialize components (filesystem will be created per-command based on config)
	store = curator.NewMemoryOperationStore()
	reporter = curator.NewReporter()
	
	// Add global flags
	rootCmd.PersistentFlags().String("ai-provider", "", "AI provider to use (mock, gemini) - overrides CURATOR_AI_PROVIDER")
	rootCmd.PersistentFlags().String("filesystem", "", "Filesystem type to use (memory, local, googledrive) - overrides CURATOR_FILESYSTEM_TYPE")
	rootCmd.PersistentFlags().String("root", "", "Root path for local filesystem - overrides CURATOR_FILESYSTEM_ROOT")
	
	// Global flags
	reorganizeCmd.Flags().Bool("dry-run", false, "Generate plan without executing")
	reorganizeCmd.Flags().String("exclude", "", "Comma-separated list of patterns to exclude")
	
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}