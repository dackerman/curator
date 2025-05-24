package main

import (
	"fmt"
	"os"

	"github.com/dackerman/curator"
	"github.com/spf13/cobra"
)

// Global configuration
var config curator.Configuration

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
		
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Scanning filesystem at /...\n")
		fmt.Printf("Using %s filesystem...\n", finalConfig.FileSystem.Type)
		
		fmt.Printf("Using %s AI provider...\n", finalConfig.AI.Provider)
		
		// Execute reorganize command
		reorganizeOpts := curator.ReorganizeOptions{
			DryRun:  dryRun,
			Exclude: exclude,
		}
		
		plan, err := curator.ExecuteReorganize(opts, reorganizeOpts)
		if err != nil {
			return err
		}
		
		// Display the plan
		fmt.Println()
		fmt.Print(opts.Reporter.FormatReorganizationPlan(plan))
		
		if !dryRun {
			fmt.Printf("\nPlan saved with ID: %s\n", plan.ID)
		}
		
		return nil
	},
}


var listPlansCmd = &cobra.Command{
	Use:   "list-plans",
	Short: "List all reorganization plans",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Execute list-plans command
		summaries, err := curator.ExecuteListPlans(opts)
		if err != nil {
			return err
		}
		
		fmt.Print(opts.Reporter.FormatPlanSummaries(summaries))
		return nil
	},
}

var showPlanCmd = &cobra.Command{
	Use:   "show-plan [plan-id]",
	Short: "Show details of a specific plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Execute show-plan command
		plan, err := curator.ExecuteShowPlan(opts, planID)
		if err != nil {
			return err
		}
		
		fmt.Print(opts.Reporter.FormatReorganizationPlan(plan))
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
		
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Execute apply command
		applyOpts := curator.ApplyOptions{
			FailFast: failFast,
		}
		
		execLog, err := curator.ExecuteApply(opts, planID, applyOpts)
		if err != nil {
			return err
		}
		
		// Display execution results
		fmt.Println()
		fmt.Print(opts.Reporter.FormatExecutionLog(execLog))
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status [plan-id]",
	Short: "Check the status of a plan execution",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planID := args[0]
		
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Execute status command
		execLog, err := curator.ExecuteStatus(opts, planID)
		if err != nil {
			return err
		}
		
		fmt.Print(opts.Reporter.FormatExecutionLog(execLog))
		return nil
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "Show execution history",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Execute history command
		logs, err := curator.ExecuteHistory(opts)
		if err != nil {
			return err
		}
		
		fmt.Print(opts.Reporter.FormatExecutionHistory(logs))
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
		
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Using %s filesystem...\n", finalConfig.FileSystem.Type)
		fmt.Printf("Using %s AI provider...\n", finalConfig.AI.Provider)
		
		// Execute deduplicate command
		report, err := curator.ExecuteDeduplicate(opts)
		if err != nil {
			return err
		}
		
		fmt.Print(opts.Reporter.FormatDuplicationReport(report))
		return nil
	},
}

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up junk and unnecessary files",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = cmd.Flags().GetBool("dry-run") // dry-run not yet implemented for cleanup
		
		fmt.Println("Scanning for junk files...")
		
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Using %s filesystem...\n", finalConfig.FileSystem.Type)
		fmt.Printf("Using %s AI provider...\n", finalConfig.AI.Provider)
		
		// Execute cleanup command
		plan, err := curator.ExecuteCleanup(opts)
		if err != nil {
			return err
		}
		
		fmt.Print(opts.Reporter.FormatCleanupPlan(plan))
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
		
		// Apply command-line flag overrides to configuration
		aiProvider, _ := cmd.Flags().GetString("ai-provider")
		filesystem, _ := cmd.Flags().GetString("filesystem")
		root, _ := cmd.Flags().GetString("root")
		verbose, _ := cmd.Flags().GetBool("verbose")
		
		finalConfig := curator.OverrideConfiguration(config, aiProvider, filesystem, root)
		finalConfig = curator.PopulateConfigurationFromEnvironment(finalConfig)
		
		// Create command options
		opts, err := curator.CreateCommandOptions(finalConfig)
		if err != nil {
			return fmt.Errorf("failed to create command options: %w", err)
		}
		opts.Verbose = verbose
		
		// Close analyzer if it supports it (for Gemini)
		if closer, ok := opts.Analyzer.(interface{ Close() error }); ok {
			defer closer.Close()
		}
		
		fmt.Printf("Using %s filesystem...\n", finalConfig.FileSystem.Type)
		fmt.Printf("Using %s AI provider...\n", finalConfig.AI.Provider)
		
		// Execute rename command
		plan, err := curator.ExecuteRename(opts)
		if err != nil {
			return err
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
	config = curator.LoadConfigurationFromEnvironment()
	
	// Add global flags
	rootCmd.PersistentFlags().String("ai-provider", "", "AI provider to use (mock, gemini) - overrides CURATOR_AI_PROVIDER")
	rootCmd.PersistentFlags().String("filesystem", "", "Filesystem type to use (memory, local, googledrive) - overrides CURATOR_FILESYSTEM_TYPE")
	rootCmd.PersistentFlags().String("root", "", "Root path for local filesystem - overrides CURATOR_FILESYSTEM_ROOT")
	rootCmd.PersistentFlags().Bool("verbose", false, "Enable debug logging (shows files found, AI prompts/responses, planned actions)")
	
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