# Tests

## Overview

| Package | File | What it tests |
|---------|------|---------------|
| `internal/config` | `config_test.go` | Config parsing, validation, prompts, mixed package formats |
| `internal/engine` | `engine_test.go` | Template rendering, `renderName` (plain, conditional true/false, empty) |
| `internal/manifest` | `manifest_test.go` | Manifest load/save, atomic writes, state file paths |
| `internal/prefs` | `prefs_test.go` | State load/save, prompt resolve (bool, string, cached skip), coerceValue, BuildData, FormatPromptValue, askBool retry |

## Running

```bash
# All tests (no root)
make test

# Individual package
go test ./internal/config/ -v
go test ./internal/engine/ -v
go test ./internal/manifest/ -v
go test ./internal/prefs/ -v
```

## How they work

### Unit tests
All tests use Go's standard `testing` package with `t.TempDir()` for filesystem isolation. No external test frameworks.

## Test environment
- All tests create temporary directories via `t.TempDir()`, cleaned up automatically
- No root privileges required
- No real home directories or system files are touched
