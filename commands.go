package curator

import (
	"fmt"
	"os"
)

// CommandOptions holds common options for all commands
type CommandOptions struct {
	FileSystem FileSystem
	Store      OperationStore
	Analyzer   AIAnalyzer
	Reporter   *Reporter
}

// ReorganizeOptions holds options specific to the reorganize command
type ReorganizeOptions struct {
	DryRun  bool
	Exclude string
}

// ApplyOptions holds options specific to the apply command
type ApplyOptions struct {
	FailFast bool
}

// ExecuteReorganize performs the reorganize operation with the given dependencies
func ExecuteReorganize(opts CommandOptions, reorganizeOpts ReorganizeOptions) (*ReorganizationPlan, error) {
	// Get all files recursively from the filesystem
	allFiles, err := getAllFilesRecursively(opts.FileSystem, "/")
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}

	// Generate reorganization plan using the analyzer
	plan, err := opts.Analyzer.AnalyzeForReorganization(allFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze files: %w", err)
	}

	// Save the plan if not dry run
	if !reorganizeOpts.DryRun {
		if err := opts.Store.SavePlan(plan); err != nil {
			return nil, fmt.Errorf("failed to save plan: %w", err)
		}
	}

	return plan, nil
}

// ExecuteListPlans lists all saved reorganization plans
func ExecuteListPlans(opts CommandOptions) ([]*PlanSummary, error) {
	summaries, err := opts.Store.ListPlans()
	if err != nil {
		return nil, fmt.Errorf("failed to list plans: %w", err)
	}
	
	return summaries, nil
}

// ExecuteShowPlan shows details of a specific plan
func ExecuteShowPlan(opts CommandOptions, planID string) (*ReorganizationPlan, error) {
	plan, err := opts.Store.GetPlan(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	
	return plan, nil
}

// ExecuteApply executes a reorganization plan
func ExecuteApply(opts CommandOptions, planID string, applyOpts ApplyOptions) (*ExecutionLog, error) {
	// Create execution engine
	engine := NewExecutionEngine(opts.FileSystem, opts.Store)
	
	// Resume any pending operations first
	if err := engine.ResumePendingOperations(); err != nil {
		// This is just a warning, don't fail the whole operation
		fmt.Printf("Warning: failed to resume pending operations: %v\n", err)
	}
	
	// Execute the plan
	execLog, err := engine.ExecutePlan(planID, applyOpts.FailFast)
	if err != nil {
		return nil, fmt.Errorf("failed to execute plan: %w", err)
	}
	
	return execLog, nil
}

// ExecuteStatus checks the status of a plan execution
func ExecuteStatus(opts CommandOptions, planID string) (*ExecutionLog, error) {
	// Create execution engine
	engine := NewExecutionEngine(opts.FileSystem, opts.Store)
	
	execLog, err := engine.GetExecutionStatus(planID)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution status: %w", err)
	}
	
	return execLog, nil
}

// ExecuteHistory shows execution history
func ExecuteHistory(opts CommandOptions) ([]*ExecutionLog, error) {
	logs, err := opts.Store.GetExecutionHistory()
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}
	
	return logs, nil
}

// ExecuteDeduplicate finds duplicate files
func ExecuteDeduplicate(opts CommandOptions) (*DuplicationReport, error) {
	// Get all files recursively
	allFiles, err := getAllFilesRecursively(opts.FileSystem, "/")
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}
	
	// Analyze for duplicates
	report, err := opts.Analyzer.AnalyzeForDuplicates(allFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze duplicates: %w", err)
	}
	
	return report, nil
}

// ExecuteCleanup identifies junk files for cleanup
func ExecuteCleanup(opts CommandOptions) (*CleanupPlan, error) {
	// Get all files recursively
	allFiles, err := getAllFilesRecursively(opts.FileSystem, "/")
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}
	
	// Analyze for cleanup
	plan, err := opts.Analyzer.AnalyzeForCleanup(allFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze cleanup: %w", err)
	}
	
	return plan, nil
}

// ExecuteRename standardizes file naming conventions
func ExecuteRename(opts CommandOptions) (*RenamingPlan, error) {
	// Get all files recursively
	allFiles, err := getAllFilesRecursively(opts.FileSystem, "/")
	if err != nil {
		return nil, fmt.Errorf("failed to get all files: %w", err)
	}
	
	// Analyze for renaming
	plan, err := opts.Analyzer.AnalyzeForRenaming(allFiles)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze renaming: %w", err)
	}
	
	return plan, nil
}

// Helper function to get all files recursively (moved from main.go)
func getAllFilesRecursively(fs FileSystem, root string) ([]FileInfo, error) {
	var allFiles []FileInfo
	
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

// Configuration represents all the configuration needed for commands
type Configuration struct {
	AI         AIConfig
	FileSystem FileSystemConfig
	StoreDir   string
}

// CreateCommandOptions creates CommandOptions from Configuration
func CreateCommandOptions(config Configuration) (CommandOptions, error) {
	// Create filesystem
	var fs FileSystem
	var err error
	
	switch config.FileSystem.Type {
	case "memory":
		memFS := NewMemoryFileSystem()
		// Add sample files for memory filesystem
		setupSampleFilesForTesting(memFS)
		fs = memFS
	case "local":
		fs, err = NewLocalFileSystem(config.FileSystem.Root)
		if err != nil {
			return CommandOptions{}, fmt.Errorf("failed to create local filesystem: %w", err)
		}
	case "googledrive":
		if config.FileSystem.GoogleDrive == nil {
			return CommandOptions{}, fmt.Errorf("Google Drive configuration is required")
		}
		fs, err = NewGoogleDriveFileSystem(config.FileSystem.GoogleDrive)
		if err != nil {
			return CommandOptions{}, fmt.Errorf("failed to create Google Drive filesystem: %w", err)
		}
	default:
		return CommandOptions{}, fmt.Errorf("unknown filesystem type: %s", config.FileSystem.Type)
	}
	
	// Create store
	store, err := NewFileOperationStore(config.StoreDir)
	if err != nil {
		return CommandOptions{}, fmt.Errorf("failed to create operation store: %w", err)
	}
	
	// Create analyzer
	var analyzer AIAnalyzer
	switch config.AI.Provider {
	case "mock":
		analyzer = NewMockAIAnalyzer()
	case "gemini":
		if config.AI.Gemini == nil {
			return CommandOptions{}, fmt.Errorf("Gemini configuration is required")
		}
		analyzer, err = NewGeminiAnalyzer(config.AI.Gemini)
		if err != nil {
			return CommandOptions{}, fmt.Errorf("failed to create Gemini analyzer: %w", err)
		}
	default:
		return CommandOptions{}, fmt.Errorf("unknown AI provider: %s", config.AI.Provider)
	}
	
	// Create reporter
	reporter := NewReporter()
	
	return CommandOptions{
		FileSystem: fs,
		Store:      store,
		Analyzer:   analyzer,
		Reporter:   reporter,
	}, nil
}

// setupSampleFilesForTesting adds sample files to memory filesystem
func setupSampleFilesForTesting(mfs *MemoryFileSystem) {
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

// LoadConfigurationFromEnvironment loads configuration from environment variables
func LoadConfigurationFromEnvironment() Configuration {
	config := LoadConfig() // Use existing config loading
	
	return Configuration{
		AI:         config.AI,
		FileSystem: config.FileSystem,
		StoreDir:   GetDefaultStoreDir(),
	}
}

// OverrideConfiguration applies command-line flag overrides to configuration
func OverrideConfiguration(config Configuration, aiProvider, filesystem, root string) Configuration {
	if aiProvider != "" {
		config.AI.Provider = aiProvider
		if aiProvider == "gemini" && config.AI.Gemini == nil {
			config.AI.Gemini = DefaultGeminiConfig()
		}
	}
	
	if filesystem != "" {
		config.FileSystem.Type = filesystem
	}
	
	if root != "" {
		config.FileSystem.Root = root
	}
	
	return config
}

// PopulateConfigurationFromEnvironment fills in missing environment-dependent config values
func PopulateConfigurationFromEnvironment(config Configuration) Configuration {
	// Re-load the full configuration to get all environment variables
	envConfig := LoadConfig()
	
	// Populate AI configuration from environment
	if config.AI.Provider == "gemini" {
		if envConfig.AI.Provider == "gemini" && envConfig.AI.Gemini != nil {
			// Use the fully loaded Gemini config from environment
			config.AI.Gemini = envConfig.AI.Gemini
		} else if config.AI.Gemini != nil && config.AI.Gemini.APIKey == "" {
			// Fallback: populate just the API key if Gemini config exists but key is missing
			if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
				config.AI.Gemini.APIKey = apiKey
			}
		}
	}
	
	// Populate Google Drive configuration from environment
	if config.FileSystem.Type == "googledrive" {
		if envConfig.FileSystem.Type == "googledrive" && envConfig.FileSystem.GoogleDrive != nil {
			// Use the fully loaded Google Drive config from environment
			config.FileSystem.GoogleDrive = envConfig.FileSystem.GoogleDrive
		} else if config.FileSystem.GoogleDrive == nil {
			// Load Google Drive config from environment if not set
			config.FileSystem.GoogleDrive = loadGoogleDriveConfig()
		}
	}
	
	return config
}