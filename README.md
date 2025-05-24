# Curator

AI-powered document organization system with intelligent categorization and management.

## Overview

Curator is a Go-based tool that uses AI to intelligently reorganize file systems, starting with Google Drive. It analyzes file structures, proposes reorganization plans with detailed explanations, and executes approved changes while maintaining a complete audit trail.

## Key Features

- **AI-driven analysis**: Intelligent categorization and organization suggestions
- **Safe operations**: Every action requires explicit approval with detailed explanations
- **Complete audit trail**: Full logging and rollback capability
- **Multiple operation types**: Reorganize, deduplicate, cleanup, and rename
- **Filesystem agnostic**: Pluggable backend support (Google Drive, local filesystem, etc.)
- **Graceful failure handling**: Resume interrupted operations, handle conflicts

## Core Principles

1. **No surprises**: Every action is explained and requires explicit approval
2. **Filesystem agnostic**: Works with any filesystem-like backend
3. **AI provider agnostic**: Pluggable AI providers (starting with Gemini)
4. **Safe and reversible**: Complete audit trail and rollback capability
5. **Graceful failure**: Handle interruptions and conflicts without data loss

## Quick Start

```bash
# Install curator
go install ./cmd/curator

# Analyze your filesystem and generate a reorganization plan
curator reorganize --dry-run

# Review the generated plan
curator list-plans
curator show-plan <plan-id>

# Execute the plan
curator apply <plan-id>

# Check execution status
curator status <plan-id>
```

## Available Operations

- **Reorganize**: Move files/folders to create logical structure
- **Deduplicate**: Identify and remove duplicate files  
- **Cleanup**: Remove junk/unnecessary files
- **Rename**: Standardize file naming conventions

## Development Status

Currently in Phase 1 development - building core infrastructure:

- ✅ Core interfaces and data structures
- ✅ In-memory filesystem for testing
- ✅ Operation store with in-memory implementation
- ✅ Basic CLI structure
- ✅ Unit tests for core functionality


next, phase 2

See [docs/spec.md](docs/spec.md) for full project specification and roadmap.

## Contributing

This project is in early development. See the project specification for architecture details and development guidelines.

## License

MIT License