---
name: dev-test
description: This skill should be used when running tests, validating builds, or verifying deployments for the Device Metrics Exporter. Orchestrates unit testing, E2E testing, docker image management, and K8s deployment verification.
---

# Dev Test Orchestrator

Comprehensive test orchestration for the AMD Device Metrics Exporter project. This skill coordinates four distinct test workflows and delegates to the dev-test-agent for complex orchestration.

## Overview

The Device Metrics Exporter testing requires four types of validations:

1. **Unit Testing** (unit test workflow)
   - Runs Go unit test suite
   - Executed inside docker-shell (Ubuntu build container)
   - Validates code correctness and test coverage
   - Reports pass/fail with detailed failures

2. **E2E Testing** (mock mode workflow)
   - Runs end-to-end tests via `make e2e`
   - Uses mock gpuagent for integration testing
   - Executed inside docker-shell (Ubuntu build container)
   - Validates exporter-to-gpuagent communication
   - Reports E2E scenario results

3. **Docker Image Management** (docker push workflow)
   - Loads image from `docker/device-metrics-exporter-latest.tar.gz`
   - Retags to `$DOCKER_REGISTRY/$USER/device-metrics-exporter:dev`
   - Pushes to registry for deployment testing
   - Validates registry upload success

4. **Kubernetes Testing** (k8s deployment workflow)
   - Runs `make k8s-test` inside docker-shell
   - Uses user's kubeconfig to access cluster
   - Validates deployment, pod health, metrics endpoint
   - Reports deployment status and errors

## Workflow Selection

When you need to test or validate, this skill determines which workflow to invoke based on your request:

### Single Workflow Selection

- **"Run unit tests"** / **"Test the code"** → Invoke unit test workflow
- **"Run e2e tests"** / **"Run mock tests"** → Invoke E2E test workflow
- **"Push dev image"** / **"Load docker image"** → Invoke docker push workflow
- **"Test k8s deployment"** / **"Run k8s tests"** → Invoke k8s deployment workflow

### Full Test Sequence

- **"Run all tests"** / **"Validate everything"** / **"Test deployment"** → Execute all four workflows sequentially:
  1. Unit Tests (validate code)
  2. E2E Tests (validate integration with mock gpuagent)
  3. Docker Image Push (prepare deployment artifact)
  4. K8s Deployment Tests (verify cluster deployment)

## Prerequisites

Before starting any tests, ensure:

- **Docker daemon running**: All tests use Docker containers or images
- **Environment variables set**: `DOCKER_REGISTRY`, `USER` (from `dev.env`)
- **Build artifacts available**:
  - Unit tests: source code
  - E2E tests: exporter binary + mock gpuagent
  - Docker push: `docker/device-metrics-exporter-latest.tar.gz`
  - K8s tests: pushed dev image + valid kubeconfig

## Test Workflows Reference

### Unit Testing Workflow
Run Go unit test suite inside docker-shell environment.

**Container**: Ubuntu-based exporter build container  
**Command**: `make unit-test`  
**Output**: Test results, coverage, failure details  
**Success criteria**: All tests pass, exit code 0  

### E2E Testing Workflow
Run end-to-end tests with mock gpuagent inside docker-shell environment.

**Container**: Ubuntu-based exporter build container  
**Command**: `make e2e`  
**Requirements**: `bin/amd-metrics-exporter`, `assets/gpuagent_mock.bin.gz`  
**Output**: E2E scenario results, integration validation  
**Success criteria**: All scenarios pass, mock connection works, exit code 0  

### Docker Image Push Workflow
Load, retag, and push docker image to development registry.

**Location**: Host machine (not in container)  
**Input**: `docker/device-metrics-exporter-latest.tar.gz`  
**Output**: Image at `$DOCKER_REGISTRY/$USER/device-metrics-exporter:dev`  
**Success criteria**: Image pushed, digest confirmed  

### K8s Deployment Testing Workflow
Deploy and validate exporter on Kubernetes cluster.

**Container**: Docker-shell with kubeconfig mounted  
**Command**: `make k8s-test`  
**Input**: User's kubeconfig file  
**Output**: Deployment status, pod health, metrics endpoint  
**Success criteria**: Pods running, metrics accessible  

## Agent Delegation

For complex testing requiring orchestration, validation, and error handling, delegate to the **dev-test-agent**.

The dev-test-agent provides:
- Environment validation (docker, registry, kubeconfig)
- Sequential test execution
- Detailed result reporting
- Error diagnosis and remediation guidance
- Next steps suggestions

## Usage Examples

### Quick Code Validation
```bash
/dev-test unit
```
Runs unit tests to verify code changes before committing.

### End-to-End Integration Testing
```bash
/dev-test e2e
```
Runs E2E tests with mock gpuagent to verify integration.

### Prepare Development Deployment
```bash
/dev-test docker
```
Loads latest built image and pushes to dev registry for testing.

### Verify K8s Deployment
```bash
/dev-test k8s
```
Tests deployment on K8s cluster (will ask for kubeconfig path).

### Complete Validation Pipeline
```bash
/dev-test
```
Runs all four workflows: unit → e2e → docker → k8s (full CI/CD validation).

## Environment Setup

### Required Environment Variables

```bash
# Source environment file first
source dev.env

# Verify variables
echo $DOCKER_REGISTRY  # e.g., registry.example.com
echo $USER             # e.g., praveen
```

### Required Files

- **Source code**: `pkg/`, `cmd/` directories
- **Docker tarball**: `docker/device-metrics-exporter-latest.tar.gz` (for docker/k8s tests)
- **Environment file**: `dev.env` (contains DOCKER_REGISTRY)
- **Kubeconfig**: User-provided path (for k8s tests)

## Common Test Scenarios

### Scenario 1: Pre-commit validation
**Goal**: Verify code changes before committing  
**Command**: `/dev-test unit`  
**Workflow**: Runs unit tests only

### Scenario 2: Integration validation
**Goal**: Verify exporter works with gpuagent  
**Command**: `/dev-test e2e`  
**Workflow**: Runs E2E tests with mock gpuagent

### Scenario 3: Pre-deployment validation
**Goal**: Validate docker image before deploying  
**Command**: `/dev-test docker`  
**Workflow**: Loads tarball, pushes to registry

### Scenario 4: Deployment verification
**Goal**: Ensure deployment works on K8s  
**Command**: `/dev-test k8s`  
**Workflow**: Deploys to cluster, validates pods

### Scenario 5: Full CI/CD pipeline
**Goal**: Complete validation before production  
**Command**: `/dev-test`  
**Workflow**: Unit → E2E → Docker → K8s (all four)

## What to Expect

When you invoke this skill:

1. **Environment Check** ✓
   - Validates docker daemon running
   - Checks environment variables set
   - Verifies required files exist
   - Confirms prerequisites met

2. **Test Execution** ✓
   - Runs requested test workflows in sequence
   - Monitors progress and captures output
   - Detects failures immediately
   - Reports status for each stage

3. **Result Reporting** ✓
   - Clear pass/fail status for each test
   - Detailed failure messages with file:line references
   - Error diagnosis and root cause analysis
   - Suggested remediation steps

4. **Next Steps Guidance** ✓
   - What to do after successful tests
   - How to fix failures if any occurred
   - Commands to run for further validation
   - Deployment recommendations

## Troubleshooting Guide

### Unit Test Failures

**Symptom**: Tests fail with compilation errors  
**Cause**: Syntax errors or import issues in code  
**Fix**: Check file:line references, fix code errors  

**Symptom**: Tests fail with assertion errors  
**Cause**: Logic errors or incorrect test expectations  
**Fix**: Review failing test code, update logic or tests  

**Symptom**: Tests timeout  
**Cause**: Dependent services not available (mock gpuagent)  
**Fix**: Check test dependencies, verify mocks configured  

### Docker Push Failures

**Symptom**: Tarball not found  
**Cause**: Docker image not built yet  
**Fix**: Run `make docker` to build image first  

**Symptom**: Registry authentication failure  
**Cause**: Not logged into docker registry  
**Fix**: Run `docker login $DOCKER_REGISTRY`  

**Symptom**: Network timeout during push  
**Cause**: Network connectivity issues  
**Fix**: Check network, retry push  

### E2E Test Failures

**Symptom**: Mock gpuagent binary not found  
**Cause**: Mock assets not built  
**Fix**: Run `/builder gpuagent` to build mock binaries  

**Symptom**: Exporter binary not found  
**Cause**: Exporter not built  
**Fix**: Run `/builder exporter` to build exporter binary  

**Symptom**: Socket connection failure  
**Cause**: Mock gpuagent failed to start  
**Fix**: Check mock process, verify socket path, review logs  

**Symptom**: Mock data mismatch  
**Cause**: Test expectations don't match mock responses  
**Fix**: Review E2E test output for expected vs actual values  

### K8s Deployment Failures

**Symptom**: ImagePullBackOff  
**Cause**: Dev image not pushed to registry  
**Fix**: Run docker push workflow first (`/dev-test docker`)  

**Symptom**: CrashLoopBackOff  
**Cause**: Configuration errors or missing dependencies  
**Fix**: Check pod logs, verify config.json, check gpuagent socket  

**Symptom**: Kubeconfig not found  
**Cause**: Invalid kubeconfig path  
**Fix**: Provide correct path to valid kubeconfig  

## Test Results Interpretation

### Success Indicators

✓ **Unit Tests**:
- All test suites show `ok` status
- Exit code 0
- No panic/crash messages
- Coverage metrics reported

✓ **Docker Push**:
- Image loaded successfully
- Tag operation completes
- Push shows "digest: sha256:..." message
- Image visible in `docker images`

✓ **K8s Deployment**:
- Pods reach Running state (1/1)
- No restarts or crashes
- Metrics endpoint accessible
- Exit code 0 from k8s-test

### Failure Indicators

✗ **Unit Tests**:
- Any test shows `FAIL` status
- Non-zero exit code
- Panic messages or stack traces
- Compilation errors

✗ **Docker Push**:
- Load fails with tar errors
- Authentication denied
- Push timeout or failure
- Missing tarball

✗ **K8s Deployment**:
- Pods stuck in Pending/CrashLoopBackOff
- ImagePullBackOff errors
- Deployment timeout
- Non-zero exit code

## Advanced Usage

### Test Specific Components

```bash
# Test only GPU agent integration
go test -v ./pkg/amdgpu/gpuagent/...

# Test only NIC agent
go test -v ./pkg/amdnic/nicagent/...

# Test with coverage
make unit-test COVERAGE=1
```

### Docker Image Variants

The skill handles the latest built image. To test specific variants:

```bash
# Build specific variant first
make docker-sriov        # SR-IOV variant
make docker-ainic        # AINIC variant
make docker-mock         # Mock variant

# Then push to registry
/dev-test docker
```

### K8s Testing with Custom Namespace

```bash
# Set namespace before k8s-test
export K8S_NAMESPACE=my-test-namespace

# Then run k8s tests
/dev-test k8s
```

## Integration with Builder Skill

The dev-test skill works hand-in-hand with the builder skill:

**Typical workflow**:
1. `/ builder` - Build artifacts (gpuagent, exporter, docker)
2. `/dev-test` - Validate builds (unit, docker, k8s)
3. Deploy to production (if all tests pass)

**Quick iteration cycle**:
```bash
# Make code changes
vim pkg/exporter/exporter.go

# Rebuild only exporter
/builder exporter

# Test changes
/dev-test unit

# If tests pass, rebuild and push docker
/builder docker
/dev-test docker k8s
```

## Summary

The Dev Test skill provides a unified entry point for all testing and validation workflows. It intelligently orchestrates test execution and delegates complex validation to the dev-test-agent when needed.

**Key benefits**:
- ✓ Comprehensive test coverage (unit, docker, k8s)
- ✓ Clear result reporting with actionable feedback
- ✓ Environment validation before testing
- ✓ Integration with build workflows
- ✓ Next steps guidance for developers
