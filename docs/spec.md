# Curator - Project Specification

## Project Overview

A Go-based tool that uses AI to intelligently reorganize file systems (starting with Google Drive). The tool analyzes file structures, proposes reorganization plans with explanations, and executes approved changes while maintaining a complete audit trail.

## Core Principles

1. **No surprises**: Every action is explained and requires explicit approval
2. **Filesystem agnostic**: Works with any filesystem-like backend
3. **AI provider agnostic**: Pluggable AI providers (starting with Gemini)
4. **Safe and reversible**: Complete audit trail and rollback capability
5. **Graceful failure**: Handle interruptions and conflicts without data loss

## Key Requirements

### Functional Requirements

1. **Multiple Operation Types** (each as separate commands):
   - **Reorganize**: Move files/folders to create logical structure
   - **Deduplicate**: Identify and remove duplicate files
   - **Cleanup**: Remove junk/unnecessary files
   - **Rename**: Standardize file naming conventions

2. **Workflow**:
   - Scan filesystem
   - AI analyzes structure
   - Generate reorganization plan with explanations
   - Present plan to user for review
   - Execute approved changes
   - Log all operations

3. **Safety Features**:
   - Write-ahead log (WAL) for crash recovery
   - Timestamp all operations
   - Persist entire plan before execution
   - Support partial execution and resume
   - Rollback capability

4. **Conflict Handling**:
   - Gracefully skip files that have moved since analysis
   - Log conflicts but continue execution
   - Optional `--fail-fast` flag to stop on first conflict

### Non-Functional Requirements

1. **Performance**: Batch API calls, handle rate limits
2. **Privacy**: Support excluding folders from analysis
3. **Usability**: Clear, concise reporting of proposed changes
4. **Extensibility**: Easy to add new filesystems and AI providers

## Architecture

### Core Interfaces

```go
package curator

import (
    "io"
    "time"
)

// FileSystem abstraction for any filesystem-like backend
type FileSystem interface {
    List(path string) ([]FileInfo, error)
    Read(path string) (io.Reader, error)
    Move(source, destination string) error
    CreateFolder(path string) error
    Delete(path string) error
    Exists(path string) (bool, error)
}

// FileInfo represents a file or folder
type FileInfo interface {
    Name() string
    Path() string
    IsDir() bool
    Size() int64
    ModTime() time.Time
    Hash() string // For deduplication
    MimeType() string
}

// AIAnalyzer for generating reorganization plans
type AIAnalyzer interface {
    AnalyzeForReorganization(files []FileInfo) (*ReorganizationPlan, error)
    AnalyzeForDuplicates(files []FileInfo) (*DuplicationReport, error)
    AnalyzeForCleanup(files []FileInfo) (*CleanupPlan, error)
    AnalyzeForRenaming(files []FileInfo) (*RenamingPlan, error)
}

// Core data structures
type ReorganizationPlan struct {
    ID          string
    Timestamp   time.Time
    Moves       []Move
    Summary     Summary
    Rationale   string
}

type Move struct {
    ID          string
    Source      string
    Destination string
    Reason      string
    Type        MoveType // FILE_MOVE, FOLDER_MOVE, CREATE_FOLDER
    FileCount   int      // For folder moves
}

type MoveType string

const (
    FileMove     MoveType = "FILE_MOVE"
    FolderMove   MoveType = "FOLDER_MOVE"
    CreateFolder MoveType = "CREATE_FOLDER"
)

// Execution tracking
type ExecutionLog struct {
    PlanID      string
    Timestamp   time.Time
    Status      ExecutionStatus
    Completed   []CompletedMove
    Failed      []FailedMove
    Skipped     []SkippedMove
}

type ExecutionStatus string

const (
    StatusInProgress ExecutionStatus = "IN_PROGRESS"
    StatusCompleted  ExecutionStatus = "COMPLETED"
    StatusPartial    ExecutionStatus = "PARTIAL"
    StatusFailed     ExecutionStatus = "FAILED"
)
```

### Storage Layer

```go
// OperationStore persists plans and execution logs
type OperationStore interface {
    SavePlan(plan *ReorganizationPlan) error
    GetPlan(id string) (*ReorganizationPlan, error)
    ListPlans() ([]*PlanSummary, error)
    
    // Write-ahead log
    LogOperation(op *Operation) error
    GetPendingOperations() ([]*Operation, error)
    MarkOperationComplete(id string) error
    
    // Execution history
    SaveExecutionLog(log *ExecutionLog) error
    GetExecutionHistory() ([]*ExecutionLog, error)
}
```

## Report Format

### Reorganization Plan Output

```
REORGANIZATION PLAN
==================
Plan ID: reorg-2024-01-15-143052
Analyzed: 1,247 files across 89 folders
Generated: 2024-01-15 14:30:52

SUMMARY
-------
✓ Create 5 new folders
✓ Move 234 files  
✓ Consolidate 12 duplicate folders
✓ 15% reduction in folder depth
✓ 87% of files will be in semantically organized folders

FOLDER STRUCTURE CHANGES
------------------------
My Drive/
├── Documents/
│   ├── [NEW] Work/
│   │   ├── Projects/ (← from /Random/ProjectStuff/)
│   │   └── Reports/ (← from /Desktop/OldReports/)
│   └── [NEW] Personal/
│       ├── Taxes/
│       │   ├── 2023/ (← consolidated from 3 locations)
│       │   └── 2024/
│       └── Notes/ (← from /random-thoughts/ and /ideas/)
└── [SHARED] Family Photos/
    └── [NEW] 2024/
        └── Vacation/ (← from /Downloads/phone-backup-aug/)

DETAILED OPERATIONS (showing first 10 of 234)
---------------------------------------------
1. CREATE FOLDER: /Documents/Work/
   → Organize work-related documents separately from personal files

2. MOVE: /Random/ProjectStuff/* → /Documents/Work/Projects/
   → Affects: 45 files
   → Consolidate scattered project files in dedicated work hierarchy

3. MOVE: /Desktop/TaxDocs2023.pdf → /Documents/Personal/Taxes/2023/
   → Group tax documents by year for easy retrieval

4. MOVE: /Downloads/phone-backup-aug/*.jpg → /Family Photos/2024/Vacation/
   → Organize photos chronologically in shared family folder

[... more operations ...]

Type 'curator apply reorg-2024-01-15-143052' to execute this plan
Type 'curator export reorg-2024-01-15-143052' to save as JSON
```

## CLI Interface

```bash
# Analyze and generate plan
curator reorganize --dry-run
curator reorganize --exclude="/Private/*,/Work Confidential/*"

# Review plans
curator list-plans
curator show-plan <plan-id>

# Execute plans
curator apply <plan-id>
curator apply <plan-id> --fail-fast

# Check status and history
curator status <plan-id>
curator history

# Rollback
curator rollback <plan-id>

# Other operations
curator deduplicate --dry-run
curator cleanup --dry-run
curator rename --pattern="consistent-naming" --dry-run
```

## Implementation Roadmap

### Phase 1: Core Infrastructure
1. Define interfaces and data structures
2. Implement in-memory filesystem for testing
3. Build operation store with SQLite
4. Create basic CLI structure

### Phase 2: Basic Functionality
1. Implement plan generation (mock AI)
2. Build execution engine with WAL
3. Add conflict handling
4. Create text-based reporting

### Phase 3: Google Drive Integration
1. Implement Google Drive filesystem adapter
2. Handle Drive-specific features (shared folders, permissions)
3. Add batch operations for performance
4. Implement rate limiting

### Phase 4: AI Integration
1. Integrate Gemini API
2. Build prompt engineering for each operation type
3. Add configuration for AI behavior
4. Test and refine AI suggestions

### Phase 5: Advanced Features
1. Implement rollback functionality
2. Add resume capability for interrupted executions
3. Build web UI (optional)
4. Add more filesystem adapters (S3, local FS)

## Key Considerations

### Google Drive Specific
- Handle shared folders and permissions appropriately
- Respect folder ownership when reorganizing
- Consider Google Drive's search when organizing (complement, not replace)

### AI Patterns to Recognize
- Yearly collections (taxes, statements, reports)
- Project-based groupings
- Personal vs work separation
- Photo organization by date/event
- Download cleanup and categorization
- Backup consolidation

### Error Handling
- Network failures during execution
- Files moved/deleted by user during execution
- Permission changes
- Storage quota issues
- Rate limit handling

### Privacy & Security
- Never log file contents
- Allow excluding sensitive folders
- Clear documentation about what data is sent to AI
- Local-first approach (plans stored locally)

## Development Guidelines

1. **Start simple**: Get basic move operations working first
2. **Test thoroughly**: Use in-memory filesystem for unit tests
3. **Fail safe**: When in doubt, skip the operation and log
4. **User first**: Clear explanations > clever algorithms
5. **Iterate**: Start with CLI, add features based on usage

## Next Steps

1. Set up Go project structure
2. Implement core interfaces
3. Build in-memory filesystem for testing
4. Create basic CLI with mock operations
5. Add execution engine with WAL
6. Integrate with Google Drive API
7. Add Gemini integration