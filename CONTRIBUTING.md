# Contributing to PicoLM Server

Thank you for your interest in contributing to PicoLM Server! This document provides guidelines and workflows for contributing to this project.

## Repository

- **URL**: https://github.com/wmik/picolm-server
- **Issues**: https://github.com/wmik/picolm-server/issues

## Development Setup

### Prerequisites

- **Go** 1.21 or higher
- **PicoLM** binary (see [PicoLM repository](https://github.com/RightNow-AI/picolm))
- Git

### Getting Started

1. **Fork and clone the repository**

   ```bash
   git clone <your-fork-url>
   cd picolm-server
   ```

2. **Download dependencies**

   ```bash
   go mod tidy
   ```

3. **Build the project**

   ```bash
   go build -o picolm-server ./cmd/server/
   ```

4. **Configure PicoLM**

   ```bash
   cp config.example.yaml config.yaml
   # Edit config.yaml with your PicoLM binary and model paths
   ```

5. **Run the server**

   ```bash
   ./picolm-server -config config.yaml
   ```

## Git Workflow

### Branch Strategy

This project follows a feature branch workflow:

- **`main`**: Production-ready code
- **Feature branches**: Create descriptive branches for new features or fixes
  - `feat/feature-name` for new features
  - `fix/issue-description` for bug fixes
  - `refactor/component-name` for code refactoring
  - `chore/task-description` for maintenance tasks

### Commit Message Format

Follow the established commit message format with emoji prefixes and detailed descriptions:

```
📚 <type>(<optional scope>): <brief description>

✨ Features Added:
- Feature 1 description
- Feature 2 description

🐛 Issues Fixed:
- Bug 1 description
- Bug 2 description

🔧 Technical improvements:
- Technical change 1
- Technical change 2
```

#### Common Emoji Prefixes

- ✨ `feat`: New features
- 🐛 `fix`: Bug fixes
- ♻️ `refactor`: Code refactoring
- 🔧 `chore`: Maintenance tasks
- 📚 `docs`: Documentation updates
- 💄 `style`: Code style changes (formatting, no logic changes)
- ⚡ `perf`: Performance improvements
- ✅ `test`: Test additions or updates

#### Examples

```
✨ feat: add streaming support for chat completions

✨ Features Added:
- Implemented SSE streaming for real-time token output
- Added StreamChat method to PicoLM client
- Created handleStreamingChat HTTP handler

🔧 Technical improvements:
- Added bufio scanner for stdout streaming
- Implemented callback-based token handler
```
```
📚 docs: add CONTRIBUTING.md with development guidelines

📚 Documentation:
- Created comprehensive contributing guide
- Documented development setup and commands
- Added git workflow and commit message format
- Included code quality standards
```

## Development Commands

```bash
# Build
go build -o picolm-server ./cmd/server/   # Build binary
go build ./...                            # Build all packages

# Run
go run ./cmd/server/                      # Run server (needs config.yaml)

# Test
go test ./...                             # Run all tests

# Code Quality
go vet ./...                              # Run go vet
gofmt -s -w .                             # Format code
gofmt -l .                                # Check formatting

# Dependencies
go mod tidy                               # Update dependencies
go mod verify                             # Verify dependencies
```

## Code Quality Standards

### Before Committing

Always run these commands:

```bash
go build ./...
go vet ./...
gofmt -l .
go test ./...
```

### Code Style

- **Go**: Follow standard Go conventions
- **File naming**: lowercase with underscores (snake_case) for Go files
- **Package names**: short, lowercase, no underscores
- **Imports**: Use `go fmt` for automatic formatting
- **Error handling**: Always handle errors, don't ignore with `_`

### Testing

- Write tests for new features and bug fixes
- Use table-driven tests where appropriate
- Name test files with `_test.go` suffix

## Project Structure

```
picolm-server/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── pkg/
│   ├── config/
│   │   └── config.go       # Configuration loading
│   ├── handlers/
│   │   └── chat.go         # HTTP request handlers
│   ├── picolm/
│   │   └── client.go       # PicoLM subprocess client
│   └── types/
│       └── openai.go       # OpenAI API types
├── config.example.yaml      # Example configuration
├── config.yaml             # Local configuration (gitignored)
├── go.mod                  # Module definition
├── go.sum                  # Dependency checksums
└── CONTRIBUTING.md         # This file
```

## Submitting Changes

1. **Create a feature branch** from `main`
2. **Make your changes** following the code style guidelines
3. **Run all quality checks** (build, vet, format, test)
4. **Commit with proper message** following the established format
5. **Push to your fork** and create a pull request

### Pull Request Process

- Provide a clear description of changes
- Link any related issues
- Ensure all checks pass (`go build`, `go vet`, `gofmt`)
- Request code review from maintainers

## Releasing

Releases are automated via GitHub Actions using [GoReleaser](https://goreleaser.com). Pushing a semantic version tag (without a `v` prefix) triggers the release workflow.

### Creating a Release

1. **Ensure `main` is up to date and CI is green**

   ```bash
   git checkout main
   git pull
   ```

2. **Tag the release with a semantic version** (e.g. `1.2.3`)

   ```bash
   git tag 1.2.3
   git push origin 1.2.3
   ```

3. **Wait for the `Release` workflow to finish.** It runs tests, then builds and publishes:

   - Binaries for `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64` (`.exe`, zipped)
   - `checksums.txt` for artifact verification
   - Release notes auto-generated from commits since the previous tag

4. **Verify the release** on the [Releases page](https://github.com/wmik/picolm-server/releases).

### Local Snapshot Build

To verify the release artifacts locally without publishing:

```bash
make snapshot
```

Artifacts are written to `dist/` (gitignored).

### Notes

- The release only ships the server binary. The [PicoLM](https://github.com/RightNow-AI/picolm) C binary must be built separately for each target platform (on Windows, use `picolm.exe`).
- Pre-release tags (e.g. `1.2.3-rc1`) are not matched by the current workflow pattern. Extend the pattern in `.github/workflows/release.yml` if needed.

## Getting Help

- Check existing issues for similar problems
- Read the [PicoLM repository](https://github.com/RightNow-AI/picolm) for PicoLM specifics
- Ask questions in issues or discussions

Thank you for contributing to PicoLM Server! 🚀
