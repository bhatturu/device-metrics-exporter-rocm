---
name: Builder
description: This skill should be used when building project artifacts - gpuagent binaries, exporter binary, or docker images. Provides build orchestration with automatic workflow detection for the Device Metrics Exporter project.
agent: builder-agent
version: 2.0.0
---

# Builder Orchestrator

Comprehensive build orchestration for the AMD Device Metrics Exporter project. This skill coordinates three distinct build workflows and delegates to the builder-agent for complex orchestration.

## Overview

The Device Metrics Exporter requires building three types of artifacts:

1. **GPUAgent Binaries** (`gpuagent-build` sub-skill)
   - Four binaries: gpuagent, gpuagent_gim, gpuagent_mock, gpuctl
   - Built from the `gpuagent/` submodule
   - Uses RHEL-based build container
   - Outputs to `assets/` directory

2. **Exporter Binary** (`exporter-build` sub-skill)
   - Single binary: `amd-metrics-exporter`
   - Built from main repository
   - Uses Ubuntu-based build container
   - Outputs to `bin/` directory

3. **Docker Images** (`docker-build` sub-skill)
   - Multiple variants: standard, SR-IOV, AINIC, Ubuntu 22.04, mock
   - Requires completed gpuagent and exporter builds
   - Host-based build (no container)
   - Outputs tagged Docker images

## Workflow Selection

When you need to build artifacts, this skill determines which sub-workflow to invoke based on your request:

### Single Workflow Selection

- **"Build gpuagent"** / **"Build gpu-agent binaries"** → Invoke `gpuagent-build` sub-skill
- **"Build exporter"** / **"Build amd-metrics-exporter"** → Invoke `exporter-build` sub-skill
- **"Build docker"** / **"Build container image"** → Invoke `docker-build` sub-skill

### Full Build Sequence

- **"Build all"** / **"Do a full build"** / **"Build everything"** → Execute all three workflows **SEQUENTIALLY** (not in parallel):
  1. GPUAgent Build (assets to `assets/`) - **WAIT for completion**
  2. Exporter Build (binary to `bin/`) - **WAIT for completion**
  3. Docker Build (image tags) - **WAIT for completion**

**CRITICAL**: Each workflow MUST complete successfully before starting the next. Do NOT run build workflows in parallel.

## Prerequisites

Before starting any build, ensure:

- **Docker daemon running**: All builds use Docker containers
- **Build containers available**: GPUAgent (RHEL), Exporter (Ubuntu)
- **GPUAgent submodule initialized** (for gpuagent builds)

## Sub-Skills Reference

### gpuagent-build
Build gpuagent binaries from the ROCm/gpu-agent submodule using RHEL container.

### exporter-build
Build the amd-metrics-exporter binary from main repository using Ubuntu container.

### docker-build
Build Docker container images for deployment (5 variants available).

## Sequential Execution Rules

**CRITICAL - NO PARALLEL BUILDS**: All build workflows MUST execute sequentially.

### Execution Requirements

1. **One workflow at a time**: Never start a new build workflow until the current one completes
2. **Wait for completion**: Use background tasks and wait for completion notifications
3. **Verify success**: Check that each step succeeded before proceeding to the next
4. **Dependencies matter**: 
   - Docker build depends on exporter binary existing
   - Exporter may depend on gpuagent assets for some variants
5. **Handle failures**: If any workflow fails, STOP and report - do not continue to dependent steps

### Why Sequential?

- Build containers can conflict if run simultaneously
- File system locks on shared directories
- Docker daemon resource contention
- Artifact dependencies between steps

## Agent Delegation

For complex builds requiring orchestration, validation, and error handling, delegate to the **builder-agent**.

The builder-agent provides:
- Pre-build validation
- Intelligent retry logic
- **Sequential orchestration** (enforces one-at-a-time execution)
- Error recovery
- Post-build validation

## Summary

The Builder skill provides a unified entry point for all build workflows. It intelligently routes build requests to the appropriate sub-skill and delegates complex orchestration to the builder-agent when needed.
