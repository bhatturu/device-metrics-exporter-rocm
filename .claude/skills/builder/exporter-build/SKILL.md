---
name: Exporter Build
description: Build the amd-metrics-exporter binary using Ubuntu-based build container with automatic proto change detection
version: 2.0.0
---

# Exporter Binary Build Workflow

Builds the `amd-metrics-exporter` binary from the main repository using an Ubuntu-based build container.

## Overview

**Container**: `${USER}_exporter-bld`
**Output**: `bin/amd-metrics-exporter` (40-50 MB)

## Build Process

### 1. Proto Change Detection
```bash
# Check for proto changes
git diff --name-only HEAD | grep "\.proto$"
# OR
find pkg/*/proto -name "*.proto" -newer bin/amd-metrics-exporter 2>/dev/null
```

### 2. Enter Container
```bash
make docker-shell
```

### 3. Build (Inside Container)

**If proto changed**:
```bash
make gen            # Generate protobuf code
make copyrights     # Update headers (may fail - benign)
make amd-metrics-exporter    # Build binary
```

**If no proto changes**:
```bash
make amd-metrics-exporter
```

### 4. Verify Success
```bash
ls -lh bin/amd-metrics-exporter
file bin/amd-metrics-exporter
./bin/amd-metrics-exporter --help
```

## Proto Change Handling

The build automatically detects proto file changes and runs the full sequence:
- `make gen` → Regenerates proto code
- `make copyrights` → Updates headers (failure is benign)
- `make amd-metrics-exporter` → Compiles binary

## Common Issues

**Proto generation fails**: Check proto syntax errors
**Copyright check fails**: Benign - continue to `make amd-metrics-exporter`
**CGO errors**: Verify `CGO_ENABLED=0` in Makefile

## Success Criteria

- ✅ Binary exists at `bin/amd-metrics-exporter`
- ✅ Size > 10 MB (typically 40-50 MB)
- ✅ Type: ELF 64-bit LSB executable
- ✅ No compilation errors

## Build Time

- With proto generation: 2-3 minutes
- Without proto generation: 30-60 seconds
