---
name: GPUAgent Build
description: Build gpuagent binaries from the gpuagent submodule with automatic retry on known build issues
version: 2.0.0
---

# GPUAgent Binary Build Workflow

Builds gpuagent, gpuagent_gim, gpuagent_mock, and gpuctl binaries from the `gpuagent/` submodule using RHEL-based build container.

## Overview

**Container**: `gpuagent-builder-rhel:9`
**Output**: 4 binaries → `assets/` directory (compressed tar files)

## Build Process

### 0. Pre-Build Clean (Optional)

**Before starting the build**, ask user:
> "Would you like to clean the existing build directory? This removes stale build artifacts and cached files that can cause build failures."

**If user confirms**, run:
```bash
cd gpuagent && make -C sw/nic/gpuagent clean && rm -rf sw/nic/build
```

**When to recommend cleaning**:
- Build has failed previously with compilation errors
- Switching between different gpuagent versions/branches
- Path-related errors in build output (e.g., CMake cache mismatches)
- First time building after cloning repository

**When cleaning is NOT needed**:
- First build attempt in a fresh clone
- Previous build succeeded and you're rebuilding after minor changes
- User explicitly skips the clean step

### 1. Build Command

**Run from repository root** (use absolute paths to avoid `cd` issues in background tasks):

```bash
docker run --rm --privileged \
  --name gpuagent-ctr-$(whoami)_$(date +%Y-%m-%d_%H.%M.%S) \
  --network host \
  -e "USER_NAME=$(whoami)" \
  -e "USER_UID=$(id -u)" \
  -e "USER_GID=$(id -g)" \
  -e "GIT_COMMIT=$(git rev-list -1 HEAD --abbrev-commit)" \
  -e "GIT_VERSION=" \
  -e "BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S%z)" \
  -v /home/praveen/go/src/github.com/pensando/device-metrics-exporter/gpuagent:/usr/src/github.com/ROCm/gpu-agent \
  -w /usr/src/github.com/ROCm/gpu-agent \
  gpuagent-builder-rhel:9 \
  bash -c "cd /usr/src/github.com/ROCm/gpu-agent && source ~/.bashrc && make gopkglist && cd sw/nic/gpuagent/ && go mod vendor && cd /usr/src/github.com/ROCm/gpu-agent && make -C sw/nic/gpuagent all"
```

**Important Build Notes**:
- **No parallel build (`-j` option)**: Sequential compilation is more reliable and avoids race conditions in third-party library builds
- **Use absolute paths**: The volume mount uses absolute path to avoid `cd` command issues in background tasks
- **User-agnostic path**: Replace `/home/praveen/` with actual user home directory path or use `$(pwd)/gpuagent`

**Why not use `make gpuagent`?**
- The Makefile target includes git configuration commands that fail when the gpuagent submodule's `.git` file references `../.git/modules/gpuagent` (which doesn't exist in the container)
- This direct docker command bypasses that issue by running the build commands directly

**IMPORTANT**: The first build may stop without errors. This is a known issue - the build completes but doesn't create the binaries.

### 2. Auto-Retry Logic

**If first build stops without error** (completes successfully but no binaries created):
- Automatically retry with the same docker command
- Notify user: "First build completed without errors but no assets created (known issue). Retrying..."

**If first build fails with errors**:
- STOP and report errors to user
- Offer worst-case recovery option (see below)

### 2.1. Worst-Case Recovery (Third-Party Corruption)

**When build fails persistently** with third-party library errors (e.g., protobuf, boost, abseil), ask user:
> "The build is failing due to third-party library compilation errors. Would you like to reset the third-party dependencies? This will:
> 1. Remove corrupted third-party directory
> 2. Restore from git  
> 3. Re-initialize submodules
> 4. Clean build directories
> 5. Retry build
>
> **Warning**: This will discard any local changes in gpuagent/sw/nic/third-party/"

**If user confirms**, run recovery sequence:

```bash
# Step 1: Reset third-party directory
cd gpuagent/sw/nic/ && rm -rf third-party && git checkout third-party && cd -

# Step 2: Re-initialize submodules
git submodule update --init --recursive -f

# Step 3: Clean build directories  
cd gpuagent && make -C sw/nic/gpuagent clean && rm -rf sw/nic/build && cd ..

# Step 4: Retry build with the docker command (use absolute path)
docker run --rm --privileged \
  --name gpuagent-ctr-$(whoami)_$(date +%Y-%m-%d_%H.%M.%S) \
  --network host \
  -e "USER_NAME=$(whoami)" \
  -e "USER_UID=$(id -u)" \
  -e "USER_GID=$(id -g)" \
  -e "GIT_COMMIT=$(git rev-list -1 HEAD --abbrev-commit)" \
  -e "GIT_VERSION=" \
  -e "BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S%z)" \
  -v $(pwd)/gpuagent:/usr/src/github.com/ROCm/gpu-agent \
  -w /usr/src/github.com/ROCm/gpu-agent \
  gpuagent-builder-rhel:9 \
  bash -c "cd /usr/src/github.com/ROCm/gpu-agent && source ~/.bashrc && make gopkglist && cd sw/nic/gpuagent/ && go mod vendor && cd /usr/src/github.com/ROCm/gpu-agent && make -C sw/nic/gpuagent"
```

**Note**: No `-j` parallel build option - sequential compilation is more reliable for third-party libraries.

**When to use recovery**:
- Persistent protobuf build errors (`install-libLTLIBRARIES`)
- Boost compilation failures
- CMake cache corruption
- After multiple failed retry attempts
- When cleaning build directory alone doesn't help

### 3. Verify Build Success

Check for binaries in:
```bash
ls -la gpuagent/sw/nic/build/x86_64/sim/bin/
```

Should contain:
- `gpuagent`
- `gpuagent_gim`  
- `gpuagent_mock`
- `gpuctl`

Also check:
```bash
ls -la gpuagent/sw/nic/build/x86_64/sim/out/
```

Should contain:
- `gpuagent*` binaries

### 4. Asset Copy (Optional)

After successful build, ask user:
> "GPUAgent build completed successfully. Would you like to copy the assets to the assets/ directory?"

**If user confirms**, first try the Makefile target:
```bash
make gpuagent-asset-copy
```

**If make target fails due to missing binaries**:
1. Check which binaries exist:
   ```bash
   ls -la gpuagent/sw/nic/build/x86_64/sim/bin/
   ```
2. Ask user:
   > "The full asset copy requires all 4 binaries (gpuagent, gpuagent_gim, gpuagent_mock, gpuctl), but only [list found binaries] were built. Would you like to copy just the available assets?"

**If user confirms partial copy**, run manual copy for available binaries:

For `gpuagent` (if exists):
```bash
mkdir -p assets && \
cp -vf gpuagent/sw/nic/build/x86_64/sim/bin/gpuagent assets/gpuagent && \
strip assets/gpuagent && \
cd assets && tar czf gpuagent_static.bin.gz gpuagent && chmod +x gpuagent_static.bin.gz gpuagent && \
rm -f gpuagent && cd ..
```

For `gpuctl` (if exists):
```bash
cp -vf gpuagent/sw/nic/build/x86_64/sim/bin/gpuctl assets/gpuctl.gobin && \
strip assets/gpuctl.gobin
```

For `gpuagent_gim` (if exists - creates SR-IOV variant):
```bash
cp -vf gpuagent/sw/nic/build/x86_64/sim/bin/gpuagent_gim assets/gpuagent && \
strip assets/gpuagent && \
cd assets && tar czf gpuagent_sriov_static.bin.gz gpuagent && chmod +x gpuagent_sriov_static.bin.gz && \
rm -f gpuagent && cd ..
```

For `gpuagent_mock` (if exists):
```bash
cp -vf gpuagent/sw/nic/build/x86_64/sim/bin/gpuagent_mock assets/gpuagent && \
strip assets/gpuagent && \
cd assets && tar czf gpuagent_mock_static.bin.gz gpuagent && chmod +x gpuagent_mock_static.bin.gz && \
rm -f gpuagent && cd ..
```

**Full asset copy** (when all binaries exist):
1. Copy binaries from `gpuagent/sw/nic/build/x86_64/sim/{bin,out}/`
2. Strip and compress them
3. Place in `assets/` directory:
   - `gpuagent_static.bin.gz`
   - `gpuagent_sriov_static.bin.gz`
   - `gpuagent_mock_static.bin.gz`
   - `gpuctl.gobin`

## Build Workflow Summary

```
Step 0: Ask user if build directory should be cleaned (optional)
        If yes → cd gpuagent && make -C sw/nic/gpuagent clean && rm -rf sw/nic/build
Step 1: Run docker build command (from repository root)
Step 2: If stops without error → retry automatically
        If fails with error → offer worst-case recovery option
Step 2.1: (Recovery) If user confirms → reset third-party, clean, and retry
Step 3: Verify binaries created in gpuagent/sw/nic/build/x86_64/sim/{bin,out}/
Step 4: Ask user if assets should be copied
Step 5: If yes → try make gpuagent-asset-copy
        If fails (missing binaries) → check which binaries exist
        → ask user if partial copy is acceptable
        → if yes, run manual copy commands for available binaries only
```

## Success Criteria

**Build Success** (Minimum):
- ✅ At least these binaries exist in `gpuagent/sw/nic/build/x86_64/sim/bin/`:
  - gpuagent (required)
  - gpuctl (required)

**Full Build Success** (All variants):
- ✅ All binaries exist in `gpuagent/sw/nic/build/x86_64/sim/bin/`:
  - gpuagent
  - gpuagent_gim
  - gpuagent_mock
  - gpuctl

**Asset Copy Success** (Minimum):
- ✅ At least these assets exist in `assets/` directory:
  - gpuagent_static.bin.gz (~8-10 MB compressed)
  - gpuctl.gobin (~14-20 MB stripped)
- ✅ File sizes are reasonable (not 0 bytes)

**Full Asset Copy Success** (All variants):
- ✅ All assets exist in `assets/` directory:
  - gpuagent_static.bin.gz
  - gpuagent_sriov_static.bin.gz
  - gpuagent_mock_static.bin.gz
  - gpuctl.gobin
- ✅ File sizes are reasonable (not 0 bytes)
- ✅ Compressed files typically 8-10 MB each
- ✅ gpuctl.gobin typically 14-20 MB

## Common Issues

**Issue 1: Build stops without error, no binaries created**
- **Symptom**: `make gpuagent` completes with exit code 0 but no binaries in build directory
- **Solution**: Automatically retry once with same command
- **Agent behavior**: Detect this condition and retry automatically while notifying user

**Issue 2: Build fails with actual errors**
- **Symptom**: Compilation errors, linker errors, or other failures
- **Solution**: Report specific errors to user and await instruction
- **Agent behavior**: Do NOT retry automatically - user may need to fix something

**Issue 3: Missing binaries during asset copy**
- **Symptom**: `make gpuagent-asset-copy` fails with "Missing: gpuagent_gim" or other binaries
- **Cause**: Standard build only creates `gpuagent` and `gpuctl`, not the GIM and mock variants
- **Solution**: Ask user if they want to copy just the available assets using manual copy commands
- **Agent behavior**: Check which binaries exist, inform user, and offer partial copy option

**Issue 4: Assets directory not found during copy**
- **Symptom**: `make gpuagent-asset-copy` fails with path errors
- **Solution**: Ensure running from repository root, not gpuagent subdirectory
- **Agent behavior**: Always run from repository root for asset copy

**Issue 5: Stale build cache with path mismatches**
- **Symptom**: CMake cache errors, protobuf build failures, path not found errors (e.g., `/Rocm/` vs `/ROCm/`)
- **Solution**: Clean build directory before rebuilding
- **Agent behavior**: Ask user about cleaning at the start of the build workflow

**Issue 6: Git submodule configuration error with `make gpuagent`**
- **Symptom**: `fatal: not a git repository: /usr/src/github.com/ROCm/gpu-agent/../.git/modules/gpuagent`
- **Solution**: Use direct docker command instead of Makefile target
- **Why**: The gpuagent submodule's `.git` file references `../.git/modules/gpuagent` which doesn't exist in the mounted container path

**Issue 7: Persistent third-party library build failures**
- **Symptom**: Repeated errors like `install-libLTLIBRARIES Error 1`, protobuf/boost build failures, race conditions
- **Solution**: Use worst-case recovery to reset third-party directory, or try sequential build (remove `-j`)
- **Agent behavior**: After multiple failed attempts, offer the recovery option with user confirmation
- **Commands**: Reset third-party → re-init submodules → clean → rebuild (without `-j`)
- **Why sequential?**: Third-party library builds may have race conditions with parallel compilation

**Issue 8: Background task `cd` command failures**
- **Symptom**: `/bin/bash: line 12: cd: gpuagent: No such file or directory`
- **Solution**: Use absolute paths for volume mounts instead of relative paths with `cd`
- **Wrong**: `cd gpuagent && docker run ... -v $(pwd):/usr/src/...`
- **Correct**: `docker run ... -v /absolute/path/to/gpuagent:/usr/src/...` or `docker run ... -v $(pwd)/gpuagent:/usr/src/...`

## Build Time

- First build (sequential, no `-j`): 10-20 minutes (includes container setup and dependency installation)
- Retry build (incremental): 3-10 minutes depending on what needs rebuilding
- Asset copy: < 30 seconds

**Note**: Sequential builds (without `-j`) are slower but more reliable and avoid race conditions in third-party library compilation.

## Next Steps

After successful build, suggest to user:
- Build the exporter binary?
- Test the gpuagent binaries?
- Proceed to full docker build?
