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
	Hash() string   // For deduplication
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
	ID        string
	Timestamp time.Time
	Moves     []Move
	Summary   Summary
	Rationale string
}

type Move struct {
	ID          string
	Source      string
	Destination string
	Reason      string
	Type        MoveType
	FileCount   int // For folder moves
}

type MoveType string

const (
	FileMove     MoveType = "FILE_MOVE"
	FolderMove   MoveType = "FOLDER_MOVE"
	CreateFolder MoveType = "CREATE_FOLDER"
)

type Summary struct {
	FoldersCreated int
	FilesMoved     int
	FoldersMovedDeduplicated int
	DepthReduction string
	OrganizationImprovement string
}

// Execution tracking
type ExecutionLog struct {
	PlanID    string
	Timestamp time.Time
	Status    ExecutionStatus
	Completed []CompletedMove
	Failed    []FailedMove
	Skipped   []SkippedMove
}

type ExecutionStatus string

const (
	StatusInProgress ExecutionStatus = "IN_PROGRESS"
	StatusCompleted  ExecutionStatus = "COMPLETED"
	StatusPartial    ExecutionStatus = "PARTIAL"
	StatusFailed     ExecutionStatus = "FAILED"
)

type CompletedMove struct {
	MoveID    string
	Timestamp time.Time
}

type FailedMove struct {
	MoveID    string
	Timestamp time.Time
	Error     string
}

type SkippedMove struct {
	MoveID    string
	Timestamp time.Time
	Reason    string
}

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

type PlanSummary struct {
	ID        string
	Timestamp time.Time
	Status    string
	FileCount int
	MoveCount int
}

type Operation struct {
	ID        string
	Type      string
	Data      []byte
	Timestamp time.Time
}

// Additional plan types for different operations
type DuplicationReport struct {
	ID         string
	Timestamp  time.Time
	Duplicates []DuplicateGroup
	Summary    DuplicationSummary
}

type DuplicateGroup struct {
	Hash  string
	Files []string
	Size  int64
}

type DuplicationSummary struct {
	TotalDuplicates int
	SpaceSaved      int64
}

type CleanupPlan struct {
	ID        string
	Timestamp time.Time
	Deletions []Deletion
	Summary   CleanupSummary
}

type Deletion struct {
	ID     string
	Path   string
	Reason string
	Size   int64
}

type CleanupSummary struct {
	FilesDeleted int
	SpaceFreed   int64
}

type RenamingPlan struct {
	ID        string
	Timestamp time.Time
	Renames   []Rename
	Summary   RenamingSummary
}

type Rename struct {
	ID      string
	OldName string
	NewName string
	Reason  string
}

type RenamingSummary struct {
	FilesRenamed int
	Pattern      string
}