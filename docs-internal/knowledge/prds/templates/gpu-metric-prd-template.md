# PRD: [Metric Name]

**Document ID**: PRD-GPU-[YYYYMMDD]-[NN]  
**Created**: [Date]  
**Author**: [User] with Claude PRD Agent  
**Status**: Draft | In Review | Approved | Implemented

---

## 1. Metric Overview

### 1.1 Metric Name
- **Prometheus Name**: `amd_gpu_[metric_name]`
- **Proto Field**: `[FieldName]`
- **Enum Index**: [Number 0-1200]
- **Metric Type**: Non-Profiler (0-800) | Profiler (801-1200)
- **Critical Metric**: Yes | No

### 1.2 Description
[Brief description of what this metric measures]

### 1.3 Purpose
[Why this metric is needed - use case, monitoring goals, etc.]

### 1.4 Critical Metric Classification
**Is this a critical metric?** [Yes/No]

If Yes:
- **Category**: [Temperature/Power/Activity/VRAM/Profiler/Other]
- **Why Critical**: [Explain why this metric is essential for workload evaluation]
- **Monitoring Use**: [How this metric is used to detect workload issues]

Critical metrics are essential for evaluating whether workloads run as expected on the GPU and must be documented in both `internal/metricsmap.md` (critical metrics list) and `docs/configuration/metricslist.md`.

---

## 2. Technical Specification

### 2.1 Proto Definition

**Location**: `pkg/amdgpu/proto/[proto_file].proto`

**Message**: [GPUStats/GPUStatus/GpuProfiler]

```protobuf
// Add/Modify in [Message]:
[data_type] [FieldName] = [field_number];  // [description]
```

**Exporter Config Enum**: `pkg/exporter/proto/exporterconfig.proto`

```protobuf
enum GPUMetricField {
    // ... existing fields
    GPU_[METRIC_NAME] = [index];  // [unit] - [description]
}
```

### 2.2 Data Specifications
- **Type**: [uint64/float64/uint32/repeated uint32/etc.]
- **Unit**: [Watts/MHz/percent/bytes/count/etc.]
- **Range**: [min-max values, e.g., 0-100 for percentage]
- **Collection Method**: [AMD-SMI API/sysfs/ROCProfiler/derived calculation]
- **Collection Frequency**: [Per scrape/cached/on-demand]

---

## 3. Driver and Platform Requirements

### 3.1 AMD-SMI Version
- **Minimum Version**: [e.g., 24.6.2]
- **API Function**: [e.g., rsmi_dev_memory_usage_get]
- **Availability**: Since AMD-SMI version [X.Y.Z]

### 3.2 Driver Requirements
- **Driver Type**: amdgpu driver | GIM driver
- **Minimum Version**: [e.g., amdgpu 6.8.0]
- **Kernel Module**: [e.g., amdgpu.ko, amdkcl.ko]

### 3.3 Platform Support

**Supported Platforms:**
- [ ] Common to all AMD GPUs
- [ ] MI2xx series (MI210, MI250, MI250X)
- [ ] MI3xx series (MI300A, MI300X)
- [ ] Other: [specify]

**Platform-Specific Notes:**
[Document any platform-specific behaviors or limitations]

### 3.4 Unsupported Platforms
[List platforms where this metric is not available and why]

---

## 4. Implementation Plan

### 4.1 Proto Changes

#### File: `pkg/amdgpu/proto/gpu.proto` (or rocprofiler.proto)

```diff
message GPUStats {
    // ... existing fields
+   // [Description of new field]
+   [type] [FieldName] = [number];
}
```

#### File: `pkg/exporter/proto/exporterconfig.proto`

```diff
enum GPUMetricField {
    // ... existing fields
+   GPU_[METRIC_NAME] = [index];  // [unit] - [description]
}
```

#### File: `pkg/amdgpu/proto/gpumetricssvc.proto` (if health service integration)

```diff
// Add field to health monitoring if applicable
```

### 4.2 Implementation Files

#### gpuagent_gpu_metrics.go

**Changes Required:**

1. Add prometheus.GaugeVec field to GpuMetrics struct (around line 65-256):
```go
type GpuMetrics struct {
    // ... existing fields
    gpu[MetricName] prometheus.GaugeVec
}
```

2. Register metric in `initPrometheusMetrics()` function (around line 721-1200):
```go
gpu[MetricName]: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
    Name: "amd_gpu_[metric_name]",
    Help: "[Description of metric]",
}, labels),
```

3. Add to field registration in `initFieldRegistration()`:
```go
// Map metric enum to GaugeVec for field logger
```

#### gpuagent_gpu.go

**Changes Required:**

1. Add value extraction in `updateGPUInfoToMetrics()` function (around line 2003):
```go
// Extract value from gpu.Stats or gpu.Status
ga.fl.logWithValidateAndExport(
    gpuid,
    ga.metrics.gpu[MetricName],
    exportermetrics.GPUMetricField_GPU_[METRIC_NAME].String(),
    labels,
    float64(stats.[FieldName]),  // or gpu.Status.[FieldName]
)
```

2. Add platform-specific handling if needed:
```go
if !utils.IsNonZeroValue(stats.[FieldName]) {
    ga.fl.markUnsupportedFields(gpuid, 
        exportermetrics.GPUMetricField_GPU_[METRIC_NAME].String())
}
```

#### rocprofiler/rocpclient.go (if profiler metric)

**Changes Required:**

1. Add counter name to profiler configuration in `getAllSupportedProfilerFields()`:
```go
profilerFields = append(profilerFields, "[rocpctl_counter_name]")
```

2. Handle 10s cache mechanism (already implemented)
3. Implement auto-disable on failure logic (already implemented)

### 4.3 File Checklist

**Files to Modify:**
- [ ] `pkg/amdgpu/proto/gpu.proto` - Add field to GPUStats/GPUStatus
- [ ] `pkg/exporter/proto/exporterconfig.proto` - Add enum entry
- [ ] `pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go` - Register Prometheus metric
- [ ] `pkg/amdgpu/gpuagent/gpuagent_gpu.go` - Implement collection logic
- [ ] `pkg/amdgpu/proto/gpumetricssvc.proto` - (Optional) If health service
- [ ] `pkg/amdgpu/rocprofiler/rocpclient.go` - (Optional) If profiler metric

**Proto Compilation:**
- [ ] Run `make gen` to regenerate protobuf code

---

## 5. Testing Requirements

### 5.1 Unit Tests

**File**: `pkg/amdgpu/gpuagent/gpuagent_test.go`

**Test Cases:**
- [ ] Add test case for metric collection
- [ ] Mock proto response with field populated
- [ ] Verify Prometheus metric registration
- [ ] Test field mapping and extraction
- [ ] Test zero/non-zero value handling

**Example Test Structure:**
```go
func TestGPU[MetricName](t *testing.T) {
    teardownSuite := setupTest(t)
    defer teardownSuite(t)
    
    ga := getNewAgent(t)
    assert.Assert(t, ga.InitConfigs() == nil)
    
    // Mock response with metric value
    mockGPUResponse.Response[0].Stats.[FieldName] = [test_value]
    
    assert.Assert(t, ga.UpdateMetricsStats(context.Background()) == nil)
    
    // Verify metric value
    // ... verification logic
}
```

### 5.2 Integration Tests

**Platform-Specific Tests:**
- [ ] Test on MI210 platform (if supported)
- [ ] Test on MI250X platform (if supported)
- [ ] Test on MI300A platform (if supported)
- [ ] Test on MI300X platform (if supported)

**Functional Tests:**
- [ ] Verify metric appears in Prometheus /metrics endpoint
- [ ] Verify correct labels attached to metric
- [ ] Verify metric value accuracy (compare with AMD-SMI/rocpctl output)
- [ ] Test metric with different GPU configurations
- [ ] Test metric persistence across exporter restarts

### 5.3 Platform-Specific Tests

**Unsupported Platform Handling:**
- [ ] Verify field logger marks unsupported fields correctly
- [ ] Verify metric is skipped on unsupported platforms
- [ ] Verify no errors logged repeatedly for unsupported fields

**Driver Version Compatibility:**
- [ ] Test with minimum required driver version
- [ ] Test with older driver versions (should gracefully handle missing field)
- [ ] Test with newer driver versions

### 5.4 Profiler-Specific Tests (if applicable)

**Cache Behavior:**
- [ ] Test 10-second cache mechanism
- [ ] Verify cache prevents excessive rocpctl calls

**Failure Handling:**
- [ ] Test auto-disable after 3 consecutive failures
- [ ] Verify profiler re-enable after success
- [ ] Test PTL state restoration (if PTL supported)

**SR-IOV:**
- [ ] Verify profiler disabled in SR-IOV mode

### 5.5 Test Team Additional Tests

**Performance Tests:**
- [ ] Measure metric collection overhead (< 1% CPU overhead target)
- [ ] Test with multiple GPUs (8+ GPUs)
- [ ] Test concurrent scrape requests

**Stress Tests:**
- [ ] Continuous collection for 24+ hours
- [ ] Rapid GPU state changes
- [ ] GPU hot-plug/unplug scenarios

**Deployment Tests:**
- [ ] Kubernetes deployment with DaemonSet
- [ ] Bare metal deployment
- [ ] SR-IOV mode deployment
- [ ] Container deployment

### 5.6 System Test Validation Criteria

**Purpose**: High-level functional validation for automation framework (blackbox testing).

#### 5.6.1 Functional Validation

**Metric Behavior:**
- [ ] Enable `GPU_[METRIC_NAME]` in config.json
- [ ] Verify metric appears in Prometheus `/metrics` endpoint as `amd_gpu_[metric_name]`
  - **Local deployment**: `curl http://localhost:5000/metrics | grep amd_gpu_[metric_name]`
  - **K8s deployment** (NodePort service): `curl http://<node_ip>:32500/metrics | grep amd_gpu_[metric_name]`
    - Use `kubectl get svc -n <namespace>` to find NodePort service details
- [ ] Verify metric has correct labels: `gpu_id`, `gpu_uuid`, `hostname`, etc.
- [ ] **Static vs Dynamic**: [Static - constant value / Dynamic - changes with workload]

**Workload Scenario** (for dynamic metrics):
- [ ] Workload to trigger metric change: [describe workload, e.g., "Run GEMM benchmark"]
- [ ] Expected behavior: [describe expected change, e.g., "Power usage increases from idle ~50W to load ~300W"]

#### 5.6.2 Metric Accuracy Validation

**Reference Comparison:**
- [ ] **Reference tool**: [amd-smi / rocpctl / rocm-smi / other]
- [ ] **Command**: [`amd-smi metric -P` / `rocpctl -m` / specific command]
- [ ] **Comparison method**: [Exact match / Within tolerance / Trend comparison]
- [ ] **Tolerance**: [±X% / ±XW / ±X units / Not applicable for counters]

**Example Validation:**
```
Reference: amd-smi metric -P (shows current_socket_power: 285W)
Exporter: amd_gpu_power_usage{gpu_id="0"} 287
Validation: Within ±5W tolerance ✓
```

**Accuracy Notes:**
- [ ] Metric is **exact match** with amd-smi (e.g., counters, IDs)
- [ ] Metric has **acceptable variance** due to sampling (specify tolerance)
- [ ] Metric is **derived/calculated** (describe calculation validation)
- [ ] Metric is **not directly comparable** (explain validation approach)

#### 5.6.3 Negative Test Cases

**Test Case 1: Metric Disabled**
- [ ] Remove `GPU_[METRIC_NAME]` from config.json fields list
- [ ] Expected: Metric should NOT appear in `/metrics` endpoint
- [ ] Restart exporter, verify metric absent

**Test Case 2: Unsupported Platform**
- [ ] Deploy on unsupported platform: [e.g., MI210 if MI300-only metric]
- [ ] Expected behavior: [Metric returns 0 / Metric skipped / Warning logged]
- [ ] Verify no errors/crashes

**Test Case 3: Driver Version Compatibility**
- [ ] Test with driver version: [older than minimum required]
- [ ] Expected: Graceful handling (metric skipped or 0, with log message)
- [ ] No crashes or repeated error logs

**Additional Negative Cases** (if applicable):
- [ ] [Describe specific negative case for this metric]
- [ ] [Another edge case to validate]

#### 5.6.4 Platform-Specific Validation

**MI2xx vs MI3xx Differences:**
- [ ] MI2xx behavior: [describe expected behavior or "not supported"]
- [ ] MI3xx behavior: [describe expected behavior or "not supported"]
- [ ] Field name differences: [if proto field names differ per platform]

**Deployment Mode Differences:**
- [ ] Hypervisor (VM/GIM driver): [supported/not supported, expected value]
- [ ] Baremetal (amdgpu driver): [supported/not supported, expected value]
- [ ] SR-IOV mode: [VF behavior vs PF behavior]

**Partitioned GPU Behavior** (if applicable):
- [ ] Unpartitioned (SPX): [behavior]
- [ ] CPX/DPX/QPX modes: [behavior - primary partition vs other partitions]
- [ ] Example: "Metric only available on partition_id=0, others return 0"

#### 5.6.5 Special Cases Validation

**For ECC Metrics:**
- [ ] **Error Injection Test**:
  - Tool: `metricsclient set-error --gpu-id 0 --error-type GPU_ECC_[TYPE] --count 5`
  - Verify counter increments by 5
  - Verify health status changes (if health-critical)
  - Clear errors: `metricsclient set-error --gpu-id 0 --error-type GPU_ECC_[TYPE] --count 0`

**For Profiler Metrics:**
- [ ] **Cache Behavior**: Verify 10-second cache (same value returned within 10s)
- [ ] **PTL Delay**: Test PTL state restoration with configured delay
- [ ] **Failure Handling**: Simulate rocpctl failure, verify auto-disable after 3 failures

**For Health Service Integration:**
- [ ] **Threshold Test**: Inject errors to exceed threshold
- [ ] **Health State**: Verify GPU health changes from HEALTHY to UNHEALTHY
- [ ] **Recovery**: Clear errors, verify health returns to HEALTHY

#### 5.6.6 Automation Test Summary

**High-Level Test Plan** (for automation framework):

1. **Setup**: Deploy exporter with metric enabled on [platform]
2. **Baseline**: Verify metric exists and has expected value range
3. **Workload**: [Run specific workload if dynamic metric]
4. **Accuracy**: Compare against [reference tool] within [tolerance]
5. **Negative**: Disable metric, verify removal from endpoint
6. **Teardown**: Cleanup and verify no residual state

**Expected Automation Test Duration**: [X minutes]

**Dependencies**:
- [ ] Hardware: [MI210 / MI250X / MI300A / MI300X]
- [ ] Software: [Driver version, AMD-SMI version, ROCm version]
- [ ] Tools: [amd-smi / rocpctl / metricsclient / workload binary]

---

## 6. Documentation Updates

### 6.1 Files to Update

**User-Facing Documentation:**
- [ ] **`docs/configuration/metricslist.md`** - Add metric to appropriate category section:
  - Temperature Metrics / Power Metrics / Activity Metrics / VRAM Metrics / etc.
  - Include: Metric name, Description, Platform support (`[MI2xx]`, `[MI3xx]`, `[MI2xx, MI3xx]`)
  - Mark with appropriate icons (✓ = supported, ✗ = not supported) for Hypervisor/Baremetal columns
  - Add deprecation note if replacing an existing metric

**Internal Documentation:**
- [ ] **`internal/metricsmap.md`** - Add two entries:
  1. **Mapping Table**: Add row with format: `Exporter Metric | GPU Agent Field | amd-smi Field | Platform Notes`
  2. **Critical Metrics List** (if critical): Add to appropriate category at top of file:
     - Temperature Metrics section (if temperature metric)
     - Power Metrics section (if power metric)
     - Activity Metrics section (if activity metric)
     - VRAM Metrics section (if VRAM metric)
     - Profiler Metrics section (if profiler metric)

**Release Notes:**
- [ ] Add to release notes: `docs/releasenotes.md`

**Configuration Examples:**
- [ ] Add example to `example/config.json`
- [ ] Update `CLAUDE.md` if new patterns introduced

### 6.2 Configuration Example

**Enabling the Metric:**

```json
{
  "gpu_config": {
    "selector": "all",
    "fields": [
      "GPU_[METRIC_NAME]"
    ],
    "labels": [
      "GPU_UUID",
      "GPU_ID",
      "HOSTNAME"
    ]
  }
}
```

**For Profiler Metrics:**

```json
{
  "gpu_config": {
    "profiler_metrics": {
      "all": true
    },
    "profiler_config": {
      "sampling_interval": 1000,
      "ptl_delay": 100
    }
  }
}
```

### 6.3 Release Notes Entry

```markdown
### New GPU Metric: [Metric Name]

Added support for `amd_gpu_[metric_name]` metric that monitors [description].

**Configuration:**
- Field: `GPU_[METRIC_NAME]`
- Type: [Profiler/Non-profiler]
- Platforms: [Supported platforms]
- Driver: [Minimum version]
- AMD-SMI: [Minimum version]

**Example:**
See `example/config.json` for configuration details.
```

---

## 7. Acceptance Criteria

**Proto and Code:**
- [ ] Proto field added to gpu.proto (or rocprofiler.proto)
- [ ] Enum added to exporterconfig.proto with correct index
- [ ] Prometheus metric registered in gpuagent_gpu_metrics.go
- [ ] Collection logic implemented in gpuagent_gpu.go
- [ ] Code compiles without errors
- [ ] `make gen` completes successfully

**Functionality:**
- [ ] Metric appears in Prometheus /metrics endpoint
- [ ] Correct value displayed for supported platforms
- [ ] Unsupported platforms handled gracefully (logged and skipped)
- [ ] Metric updates on each scrape
- [ ] Labels attached correctly

**Testing:**
- [ ] Unit tests added and passing
- [ ] Integration tests pass on target platforms
- [ ] Platform-specific tests pass
- [ ] No performance regression (< 1% overhead)

**Profiler-Specific (if applicable):**
- [ ] Profiler cache/failure logic working correctly
- [ ] Auto-disable after 3 failures working
- [ ] PTL delay working (if applicable)

**Documentation:**
- [ ] **`docs/configuration/metricslist.md`** updated with metric in appropriate category
- [ ] **`internal/metricsmap.md`** updated with mapping row
- [ ] **Critical metrics list** updated (if metric is critical)
- [ ] Configuration examples added to `example/config.json`
- [ ] Release notes updated in `docs/releasenotes.md`
- [ ] Developer guide updated (if new patterns introduced)

**Health Service (if applicable):**
- [ ] Health service integration working
- [ ] Health thresholds configurable
- [ ] Health status correctly reflects metric state

**System Test Automation (Blackbox Validation):**
- [ ] **Functional**: Metric appears in /metrics when enabled, absent when disabled
- [ ] **Accuracy**: Metric value matches reference tool within specified tolerance
  - Reference: [amd-smi command / rocpctl command / other]
  - Tolerance: [±X% / ±X units / exact match / not applicable]
  - Validation method: [direct comparison / trend analysis / calculated validation]
- [ ] **Platform**: Correct behavior on supported platforms (MI2xx/MI3xx/Hypervisor/Baremetal)
- [ ] **Negative**: Graceful handling on unsupported platforms and with older drivers
- [ ] **Workload** (if dynamic): Metric value changes appropriately with workload scenario
- [ ] **Special Cases**: ECC injection / Profiler cache / Health thresholds (if applicable)
- [ ] **Automation Framework**: All test cases executable in blackbox automation framework
- [ ] **No Manual Steps**: All validation steps are automated (no manual verification required)

---

## 8. Known Limitations

[Document any known limitations, edge cases, or platform-specific behaviors]

**Examples:**
- Metric may return zero on platforms without hardware support
- Collection requires elevated privileges (e.g., root access)
- Metric may have high overhead on certain platforms
- Cache delay may cause stale values in profiler metrics

---

## 9. Future Enhancements

[Optional: Document potential future improvements or related features]

**Examples:**
- Add derived metrics based on this field
- Add histogram metrics for distribution analysis
- Add alerting rules based on thresholds
- Extend to NIC/UAL metrics

---

## 10. References

**Documentation:**
- AMD-SMI Documentation: [link or path]
- ROCm Documentation: https://rocm.docs.amd.com/
- Driver Version Matrix: [link or path]
- Device Metrics Exporter Documentation: [docs/]

**Related PRDs:**
- [Link to related metric PRDs if any]

**Code References:**
- Proto Definitions: `pkg/amdgpu/proto/gpu.proto`
- Exporter Config: `pkg/exporter/proto/exporterconfig.proto`
- Implementation: `pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go`
- Tests: `pkg/amdgpu/gpuagent/gpuagent_test.go`

**Issue Tracking:**
- GitHub Issue: [link if applicable]
- Jira Ticket: [link if applicable]

---

## 11. Approval Sign-off

- [ ] Engineering Lead: _______________ Date: ___________
- [ ] Product Manager: _______________ Date: ___________
- [ ] Test Lead: _______________ Date: ___________
- [ ] Documentation: _______________ Date: ___________

---

**Implementation Start Date**: ___________  
**Target Completion Date**: ___________  
**Actual Completion Date**: ___________
