# Built with Claude Code

This project was built using [Claude Code](https://claude.com/claude-code), Anthropic's official CLI for Claude.

## Implementation Details

- **Date**: 2025-11-17
- **Model**: Claude Sonnet 4.5 (claude-sonnet-4-5-20250929)
- **Architecture**: Based on specifications in `architecture.md`

## What was built

A generic parallel web crawler package with the following features:

### Core Components
- **Parallel execution** with configurable worker pools (default: 10)
- **Iterator-based URL generation** using Go 1.23+ `iter.Seq[string]`
- **Flexible callback system** for request building and response handling
- **Context support** for cancellation and timeouts
- **Automatic User-Agent management** from Sansec API

### Default Helpers
- `FileURLs(path)` - Read URLs from file (one per line)
- `ResponseFileDumper(dir)` - Save response bodies to files
- `ErrorLoggerStdout()` - Log errors to stdout
- `NoopErrorHandler()` - Silent error handler (default)

### Demo Application
Located in `cmd/demo/main.go` with 5 examples demonstrating:
1. Basic crawl with defaults
2. Reading URLs from file
3. Custom POST request builder
4. Saving responses to files
5. All features combined

## Project Structure

```
crawl/
├── types.go           # Type definitions and interfaces
├── crawler.go         # Main Crawler implementation
├── useragent.go       # User agent fetching from API
├── defaults.go        # Default handlers and helpers
├── crawler_test.go    # Unit tests (all passing)
├── cmd/demo/main.go   # Demo application
├── README.md          # Documentation
└── architecture.md    # Original specification
```

## Test Coverage

All tests passing:
- Crawler initialization with default/custom config
- Basic crawling with worker pools
- Context cancellation
- Error handling
- Default handlers
- File-based URL reading
- Response file dumping
- User agent management

## Design Decisions

1. **Iterator pattern**: Chose `iter.Seq[string]` for URL generation (requires Go 1.23+)
2. **Error handling**: Optional error callback, defaults to noop handler
3. **User agent**: Fetched once on crawler creation with fallback
4. **Worker pool**: Channel-based distribution with configurable concurrency

## Running the Project

```bash
# Run tests
go test -v

# Run demo
go run ./cmd/demo 1  # Basic example
go run ./cmd/demo 2  # File URLs
go run ./cmd/demo 3  # POST requests
go run ./cmd/demo 4  # Save responses
go run ./cmd/demo 5  # All features
```
