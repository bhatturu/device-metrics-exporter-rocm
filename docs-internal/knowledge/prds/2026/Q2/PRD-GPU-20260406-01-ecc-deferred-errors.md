# PRD: ECC Deferred Error Count Metrics

**Document ID**: PRD-GPU-20260406-01  
**Created**: 2026-04-06  
**Author**: praveen with Claude PRD Agent  
**Status**: Draft

---

## 1. Metric Overview

### 1.1 Metric Names

This PRD covers **19 new GPU metrics** for deferred ECC errors:

**Total Deferred Errors:**
- **Prometheus Name**: `amd_gpu_ecc_deferred_total`
- **Proto Field**: `TotalDeferredErrors`
- **Enum Index**: 122
- **Metric Type**: Non-Profiler (0-800)

**Per-Block Deferred Errors (18 blocks):**

| Block | Prometheus Name | Proto Field | Enum Index |
|-------|----------------|-------------|------------|
| SDMA | `amd_gpu_ecc_deferred_sdma` | `SDMADeferredErrors` | 123 |
| GFX | `amd_gpu_ecc_deferred_gfx` | `GFXDeferredErrors` | 124 |
| MMHUB | `amd_gpu_ecc_deferred_mmhub` | `MMHUBDeferredErrors` | 125 |
| ATHUB | `amd_gpu_ecc_deferred_athub` | `ATHUBDeferredErrors` | 126 |
| BIF | `amd_gpu_ecc_deferred_bif` | `BIFDeferredErrors` | 127 |
| HDP | `amd_gpu_ecc_deferred_hdp` | `HDPDeferredErrors` | 128 |
| XGMI_WAFL | `amd_gpu_ecc_deferred_xgmi_wafl` | `XGMIWAFLDeferredErrors` | 129 |
| DF | `amd_gpu_ecc_deferred_df` | `DFDeferredErrors` | 130 |
| SMN | `amd_gpu_ecc_deferred_smn` | `SMNDeferredErrors` | 131 |
| SEM | `amd_gpu_ecc_deferred_sem` | `SEMDeferredErrors` | 132 |
| MP0 | `amd_gpu_ecc_deferred_mp0` | `MP0DeferredErrors` | 133 |
| MP1 | `amd_gpu_ecc_deferred_mp1` | `MP1DeferredErrors` | 134 |
| FUSE | `amd_gpu_ecc_deferred_fuse` | `FUSEDeferredErrors` | 135 |
| UMC | `amd_gpu_ecc_deferred_umc` | `UMCDeferredErrors` | 136 |
| MCA | `amd_gpu_ecc_deferred_mca` | `MCADeferredErrors` | 137 |
| VCN | `amd_gpu_ecc_deferred_vcn` | `VCNDeferredErrors` | 138 |
| JPEG | `amd_gpu_ecc_deferred_jpeg` | `JPEGDeferredErrors` | 139 |
| IH | `amd_gpu_ecc_deferred_ih` | `IHDeferredErrors` | 140 |
| MPIO | `amd_gpu_ecc_deferred_mpio` | `MPIODeferredErrors` | 141 |

### 1.2 Description

Deferred errors are a type of ECC (Error Correcting Code) error that represents uncorrectable errors that have been deferred for handling. These errors are accumulated counts per ECC block (GPU hardware component). The AMD-SMI library exposes deferred error counts via the `amdsmi_error_count_t` structure which contains:
- `correctable_count` - Already exported (GPU_ECC_CORRECT_*)
- `uncorrectable_count` - Already exported (GPU_ECC_UNCORRECT_*)
- `deferred_count` - **New field to be exported**

### 1.3 Purpose

**Primary Use Case**: Memory reliability monitoring and predictive failure analysis.

**Goals**:
1. **Completeness**: Export all ECC error types that AMD-SMI provides for comprehensive observability
2. **Proactive Monitoring**: Track deferred errors to predict potential memory failures before they escalate to uncorrectable errors
3. **Granular Diagnostics**: Per-block breakdown allows pinpointing which GPU hardware component is experiencing memory errors
4. **Alerting**: Enable customers to set up alerts based on deferred error thresholds to take preventive action

**Note**: These metrics are for **monitoring only** and will **NOT** be used in the GPU health service determination. Health service continues to use only correctable/uncorrectable errors.

### 1.4 Critical Metric Classification

**Classification**: **No - Standard Metric (Not Critical)**

**Rationale**: 
- Deferred error metrics are NOT classified as critical metrics
- Critical metrics are essential for evaluating whether workloads run as expected (Temperature, Power, Activity, VRAM, Profiler metrics)
- Deferred errors are for **predictive monitoring and alerting**, not workload performance evaluation
- They indicate potential hardware issues but do not directly measure workload execution

**Documentation Impact**:
- Will be added to `docs/configuration/metricslist.md` in the ECC Error Metrics section
- Will be added to `internal/metricsmap.md` metric mapping table
- Will **NOT** be added to the Critical Metrics list in `internal/metricsmap.md`

---

## 2. Technical Specification

### 2.1 Proto Definition

#### 2.1.1 GPUAgent Submodule Proto Changes

**Location**: `gpuagent/sw/nic/gpuagent/protos/gpu.proto`

**Message**: GPUStats (line ~630+)

```protobuf
message GPUStats {
  // ... existing fields
  
  // Total deferred errors across all blocks
  uint64 TotalDeferredErrors          = 92;  // Accumulated deferred errors (all blocks)
  
  // Per-block deferred errors (following existing correctable/uncorrectable pattern)
  uint64 SDMADeferredErrors           = 93;  // SDMA block deferred errors
  uint64 GFXDeferredErrors            = 94;  // GFX block deferred errors
  uint64 MMHUBDeferredErrors          = 95;  // MMHUB block deferred errors
  uint64 ATHUBDeferredErrors          = 96;  // ATHUB block deferred errors
  uint64 BIFDeferredErrors            = 97;  // BIF block deferred errors
  uint64 HDPDeferredErrors            = 98;  // HDP block deferred errors
  uint64 XGMIWAFLDeferredErrors       = 99;  // XGMI WAFL block deferred errors
  uint64 DFDeferredErrors             = 100; // DF block deferred errors
  uint64 SMNDeferredErrors            = 101; // SMN block deferred errors
  uint64 SEMDeferredErrors            = 102; // SEM block deferred errors
  uint64 MP0DeferredErrors            = 103; // MP0 block deferred errors
  uint64 MP1DeferredErrors            = 104; // MP1 block deferred errors
  uint64 FUSEDeferredErrors           = 105; // FUSE block deferred errors
  uint64 UMCDeferredErrors            = 106; // UMC block deferred errors
  uint64 MCADeferredErrors            = 107; // MCA block deferred errors
  uint64 VCNDeferredErrors            = 108; // VCN block deferred errors
  uint64 JPEGDeferredErrors           = 109; // JPEG block deferred errors
  uint64 IHDeferredErrors             = 110; // IH block deferred errors
  uint64 MPIODeferredErrors           = 111; // MPIO block deferred errors
  
  // Continue with existing fields
}
```

**Action Required**: Proto changes need to be made in the gpuagent submodule repository (git@github.com:ROCm/gpu-agent.git).

#### 2.1.2 GPUAgent AMD-SMI Implementation Changes (Baremetal)

**Location**: `gpuagent/sw/nic/gpuagent/api/smi/amdsmi/smi_api.cc`

**Function**: `smi_fill_ecc_stats_` (line ~991)

The function currently populates correctable and uncorrectable errors from `amdsmi_error_count_t.correctable_count` and `amdsmi_error_count_t.uncorrectable_count`. Need to add similar logic for `deferred_count`:

```cpp
static void
smi_fill_ecc_stats_ (aga_gpu_handle_t gpu_handle,
                     aga_gpu_stats_t *stats)
{
    amdsmi_error_count_t ec;
    amdsmi_status_t amdsmi_ret;
    uint64_t total_correctable_count = 0;
    uint64_t total_uncorrectable_count = 0;
    uint64_t total_deferred_count = 0;  // NEW: Add total deferred counter
    
    for (uint32_t b = AMDSMI_GPU_BLOCK_FIRST; b <= AMDSMI_GPU_BLOCK_LAST;
         b = b * 2) {
        ec = { 0 };
        amdsmi_ret = amdsmi_get_gpu_ecc_count(gpu_handle,
                                              (amdsmi_gpu_block_t)(b), &ec);
        if (amdsmi_ret == AMDSMI_STATUS_SUCCESS) {
            total_correctable_count += ec.correctable_count;
            total_uncorrectable_count += ec.uncorrectable_count;
            total_deferred_count += ec.deferred_count;  // NEW: Accumulate deferred
            
            switch (b) {
            case AMDSMI_GPU_BLOCK_UMC:
                stats->umc_correctable_errors = ec.correctable_count;
                stats->umc_uncorrectable_errors = ec.uncorrectable_count;
                stats->umc_deferred_errors = ec.deferred_count;  // NEW
                break;
            case AMDSMI_GPU_BLOCK_SDMA:
                stats->sdma_correctable_errors = ec.correctable_count;
                stats->sdma_uncorrectable_errors = ec.uncorrectable_count;
                stats->sdma_deferred_errors = ec.deferred_count;  // NEW
                break;
            // ... similar for all other 16 blocks ...
            }
        }
    }
    
    stats->total_correctable_errors = total_correctable_count;
    stats->total_uncorrectable_errors = total_uncorrectable_count;
    stats->total_deferred_errors = total_deferred_count;  // NEW: Set total
}
```

**Change Summary**:
1. Add `total_deferred_count` accumulator variable
2. Accumulate `ec.deferred_count` in the loop
3. Add `stats->*_deferred_errors = ec.deferred_count;` for each of the 18 blocks
4. Set `stats->total_deferred_errors = total_deferred_count;` at the end

#### 2.1.3 GPUAgent GIM AMD-SMI Implementation Changes (SR-IOV/Hypervisor)

**Location**: `gpuagent/sw/nic/gpuagent/api/smi/gimamdsmi/smi_api.cc`

**Function**: `smi_fill_ecc_stats_` (line ~603)

**IMPORTANT**: The GIM AMD-SMI version has the exact same function structure as the regular AMD-SMI version. Apply the **same changes** as section 2.1.2:

```cpp
static sdk_ret_t
smi_fill_ecc_stats_ (aga_gpu_handle_t gpu_handle,
                     aga_gpu_stats_t *stats)
{
    amdsmi_error_count_t ec;
    amdsmi_status_t amdsmi_ret;
    uint64_t total_correctable_count = 0;
    uint64_t total_uncorrectable_count = 0;
    uint64_t total_deferred_count = 0;  // NEW: Add total deferred counter
    
    for (uint32_t b = AMDSMI_GPU_BLOCK_FIRST; b <= AMDSMI_GPU_BLOCK_LAST;
         b = b * 2) {
        ec = { 0 };
        amdsmi_ret = amdsmi_get_gpu_ecc_count(gpu_handle,
                                              (amdsmi_gpu_block_t)(b), &ec);
        if (amdsmi_ret == AMDSMI_STATUS_SUCCESS) {
            total_correctable_count += ec.correctable_count;
            total_uncorrectable_count += ec.uncorrectable_count;
            total_deferred_count += ec.deferred_count;  // NEW: Accumulate deferred
            
            switch (b) {
            case AMDSMI_GPU_BLOCK_UMC:
                stats->umc_correctable_errors = ec.correctable_count;
                stats->umc_uncorrectable_errors = ec.uncorrectable_count;
                stats->umc_deferred_errors = ec.deferred_count;  // NEW
                break;
            case AMDSMI_GPU_BLOCK_SDMA:
                stats->sdma_correctable_errors = ec.correctable_count;
                stats->sdma_uncorrectable_errors = ec.uncorrectable_count;
                stats->sdma_deferred_errors = ec.deferred_count;  // NEW
                break;
            // ... similar for all other 16 blocks ...
            }
        }
    }
    
    stats->total_correctable_errors = total_correctable_count;
    stats->total_uncorrectable_errors = total_uncorrectable_count;
    stats->total_deferred_errors = total_deferred_count;  // NEW: Set total
    
    return SDK_RET_OK;
}
```

**Note**: Both baremetal (amdsmi) and hypervisor (gimamdsmi) implementations require identical changes to support deferred errors in both deployment modes.

#### 2.1.4 Device Metrics Exporter Proto Changes

**Location**: `pkg/amdgpu/proto/gpu.proto`

**Message**: GPUStats (line 597+)

*This proto file uses the gpuagent proto as the source. The changes from section 2.1.1 will be reflected here after proto regeneration (`make gen`).*

**Location**: `pkg/exporter/proto/exporterconfig.proto`

**Enum**: GPUMetricField

```protobuf
enum GPUMetricField {
    // ... existing fields (lines 64-121)
    GPU_VRAM_MAX_BANDWIDTH       = 121;
    
    // Deferred ECC error metrics (indices 122-141)
    GPU_ECC_DEFERRED_TOTAL       = 122;  // count - Total deferred errors across all blocks
    GPU_ECC_DEFERRED_SDMA        = 123;  // count - SDMA block deferred errors
    GPU_ECC_DEFERRED_GFX         = 124;  // count - GFX block deferred errors
    GPU_ECC_DEFERRED_MMHUB       = 125;  // count - MMHUB block deferred errors
    GPU_ECC_DEFERRED_ATHUB       = 126;  // count - ATHUB block deferred errors
    GPU_ECC_DEFERRED_BIF         = 127;  // count - BIF block deferred errors
    GPU_ECC_DEFERRED_HDP         = 128;  // count - HDP block deferred errors
    GPU_ECC_DEFERRED_XGMI_WAFL   = 129;  // count - XGMI WAFL block deferred errors
    GPU_ECC_DEFERRED_DF          = 130;  // count - DF block deferred errors
    GPU_ECC_DEFERRED_SMN         = 131;  // count - SMN block deferred errors
    GPU_ECC_DEFERRED_SEM         = 132;  // count - SEM block deferred errors
    GPU_ECC_DEFERRED_MP0         = 133;  // count - MP0 block deferred errors
    GPU_ECC_DEFERRED_MP1         = 134;  // count - MP1 block deferred errors
    GPU_ECC_DEFERRED_FUSE        = 135;  // count - FUSE block deferred errors
    GPU_ECC_DEFERRED_UMC         = 136;  // count - UMC block deferred errors
    GPU_ECC_DEFERRED_MCA         = 137;  // count - MCA block deferred errors
    GPU_ECC_DEFERRED_VCN         = 138;  // count - VCN block deferred errors
    GPU_ECC_DEFERRED_JPEG        = 139;  // count - JPEG block deferred errors
    GPU_ECC_DEFERRED_IH          = 140;  // count - IH block deferred errors
    GPU_ECC_DEFERRED_MPIO        = 141;  // count - MPIO block deferred errors
}
```

### 2.2 Data Specifications

- **Type**: uint64
- **Unit**: count (accumulated deferred errors)
- **Range**: 0 to 2^64-1 (typically stays in lower range unless severe memory issues)
- **Collection Method**: AMD-SMI API via gpuagent gRPC → smi_fill_ecc_stats_() → amdsmi_get_gpu_ecc_count()
- **Collection Frequency**: Per scrape (no caching)
- **AMD-SMI Structure**: `amdsmi_error_count_t.deferred_count` field

### 2.3 Example Prometheus Output

**Curl command to fetch metrics:**
```bash
curl -s http://localhost:2112/metrics | grep amd_gpu_ecc_deferred
```

**Expected output:**
```
# HELP amd_gpu_ecc_deferred_total Total accumulated deferred ECC errors across all GPU blocks
# TYPE amd_gpu_ecc_deferred_total gauge
amd_gpu_ecc_deferred_total{gpu_id="0",gpu_uuid="GPU-12345678-1234-1234-1234-123456789abc",hostname="gpu-node-01"} 42

# HELP amd_gpu_ecc_deferred_sdma Accumulated deferred ECC errors in SDMA block
# TYPE amd_gpu_ecc_deferred_sdma gauge
amd_gpu_ecc_deferred_sdma{gpu_id="0",gpu_uuid="GPU-12345678-1234-1234-1234-123456789abc",hostname="gpu-node-01"} 0

# HELP amd_gpu_ecc_deferred_gfx Accumulated deferred ECC errors in GFX block
# TYPE amd_gpu_ecc_deferred_gfx gauge
amd_gpu_ecc_deferred_gfx{gpu_id="0",gpu_uuid="GPU-12345678-1234-1234-1234-123456789abc",hostname="gpu-node-01"} 5

# HELP amd_gpu_ecc_deferred_umc Accumulated deferred ECC errors in UMC block
# TYPE amd_gpu_ecc_deferred_umc gauge
amd_gpu_ecc_deferred_umc{gpu_id="0",gpu_uuid="GPU-12345678-1234-1234-1234-123456789abc",hostname="gpu-node-01"} 37

# HELP amd_gpu_ecc_deferred_df Accumulated deferred ECC errors in DF block
# TYPE amd_gpu_ecc_deferred_df gauge
amd_gpu_ecc_deferred_df{gpu_id="0",gpu_uuid="GPU-12345678-1234-1234-1234-123456789abc",hostname="gpu-node-01"} 0

# ... (additional per-block metrics)

# Multi-GPU example (GPU 1)
amd_gpu_ecc_deferred_total{gpu_id="1",gpu_uuid="GPU-87654321-4321-4321-4321-cba987654321",hostname="gpu-node-01"} 15
amd_gpu_ecc_deferred_umc{gpu_id="1",gpu_uuid="GPU-87654321-4321-4321-4321-cba987654321",hostname="gpu-node-01"} 10
amd_gpu_ecc_deferred_gfx{gpu_id="1",gpu_uuid="GPU-87654321-4321-4321-4321-cba987654321",hostname="gpu-node-01"} 5
```

**Example Prometheus queries:**

```promql
# Total deferred errors across all GPUs
sum(amd_gpu_ecc_deferred_total)

# Rate of deferred errors (errors per minute)
rate(amd_gpu_ecc_deferred_total[5m]) * 60

# GPUs with UMC deferred errors > 100
amd_gpu_ecc_deferred_umc > 100

# Top 5 GPUs by total deferred errors
topk(5, amd_gpu_ecc_deferred_total)

# Deferred errors by block type across all GPUs
sum(amd_gpu_ecc_deferred_umc) by (hostname)
```

---

## 3. Driver and Platform Requirements

### 3.1 AMD-SMI Version

- **Minimum Version**: To be determined (requires investigation)
- **API Function**: `amdsmi_get_gpu_ecc_count()` (same as existing ECC metrics)
- **Availability**: The `deferred_count` field exists in `amdsmi_error_count_t` structure (see `libamdsmi/include/amd_smi/amdsmi.h:2096`)

**Action Required**: Verify minimum AMD-SMI version that populates `deferred_count` field. Check AMD-SMI release notes or test on available systems.

### 3.2 Driver Requirements

**Baremetal:**
- **Driver Type**: amdgpu driver
- **Minimum Version**: amdgpu 6.4.x and up
- **Kernel Module**: amdgpu.ko

**SR-IOV/Hypervisor:**
- **Driver Type**: GIM driver
- **Minimum Version**: 8.3.0.K and up
- **Implementation**: gimamdsmi (GPU agent SR-IOV variant)

**Note**: Deferred error support is determined by driver and hardware capabilities. Both baremetal and SR-IOV implementations read the `deferred_count` field from AMD-SMI API.

### 3.3 Platform Support

**Supported Platforms:**
- [x] MI2xx series
- [x] MI3xx series

**GPU Partition Support:**
- **Primary Partition (Partition 0)**: ✓ Supported
- **Secondary Partitions (Partition 1-7)**: ✗ Not Supported

**IMPORTANT**: ECC deferred error metrics are **only available on primary partition 0** (CPX/DPX/QPX mode). Secondary partitions (1-7) do not expose ECC error counts via AMD-SMI.

**Behavior by Partition Mode:**
- **Non-partitioned (SPX mode)**: Single partition 0, full ECC metrics available
- **CPX mode (8 partitions)**: Only partition 0 exports ECC metrics, partitions 1-7 skip
- **DPX mode (2 partitions)**: Only partition 0 exports ECC metrics, partition 1 skips
- **QPX mode (4 partitions)**: Only partition 0 exports ECC metrics, partitions 1-3 skip

**Note**: All platforms that support ECC should support deferred error counting. If a platform does not populate the `deferred_count` field, the exporter will handle via field logger (log once as unsupported, then skip).

### 3.4 Hypervisor/SR-IOV Support

**SR-IOV Implementation**: Changes included in GIM AMD-SMI implementation (`gimamdsmi/smi_api.cc`).

**Hypervisor Support Status**: TBD based on testing

**Implementation Complete**:
- Baremetal support: `amdsmi/smi_api.cc` (section 2.1.2)
- SR-IOV support: `gimamdsmi/smi_api.cc` (section 2.1.3)

**Action Required**: Test on SR-IOV system with GIM driver to verify AMD-SMI support for deferred_count field. Update metricslist.md Hypervisor column accordingly (&check; or &cross;).

### 3.5 Unsupported Platforms

**Handling Strategy**: Log once as unsupported (field logger pattern)

For platforms where `deferred_count` returns zero/unsupported:
1. Field logger marks the metric as unsupported (logs once)
2. Metric is skipped from export on subsequent scrapes
3. No repeated error logging

**Expected Behavior**:
- If AMD-SMI returns non-zero `deferred_count`: Export metric normally
- If AMD-SMI returns zero consistently: Mark as unsupported via field logger, skip export

---

## 4. Implementation Plan

### 4.1 GPUAgent Submodule Changes

#### File: `gpuagent/sw/nic/gpuagent/protos/gpu.proto`

```diff
message GPUStats {
    // ... existing fields (lines ~628-713)
    
+   // Total deferred errors
+   uint64 TotalDeferredErrors          = 92;
+   
+   // SDMA deferred errors
+   uint64 SDMADeferredErrors           = 93;
+   // GFX deferred errors
+   uint64 GFXDeferredErrors            = 94;
+   // MMHUB deferred errors
+   uint64 MMHUBDeferredErrors          = 95;
+   // ATHUB deferred errors
+   uint64 ATHUBDeferredErrors          = 96;
+   // BIF deferred errors
+   uint64 BIFDeferredErrors            = 97;
+   // HDP deferred errors
+   uint64 HDPDeferredErrors            = 98;
+   // XGMI WAFL deferred errors
+   uint64 XGMIWAFLDeferredErrors       = 99;
+   // DF deferred errors
+   uint64 DFDeferredErrors             = 100;
+   // SMN deferred errors
+   uint64 SMNDeferredErrors            = 101;
+   // SEM deferred errors
+   uint64 SEMDeferredErrors            = 102;
+   // MP0 deferred errors
+   uint64 MP0DeferredErrors            = 103;
+   // MP1 deferred errors
+   uint64 MP1DeferredErrors            = 104;
+   // FUSE deferred errors
+   uint64 FUSEDeferredErrors           = 105;
+   // UMC deferred errors
+   uint64 UMCDeferredErrors            = 106;
+   // MCA deferred errors
+   uint64 MCADeferredErrors            = 107;
+   // VCN deferred errors
+   uint64 VCNDeferredErrors            = 108;
+   // JPEG deferred errors
+   uint64 JPEGDeferredErrors           = 109;
+   // IH deferred errors
+   uint64 IHDeferredErrors             = 110;
+   // MPIO deferred errors
+   uint64 MPIODeferredErrors           = 111;
    
    // Continue with existing fields
}
```

#### File: `gpuagent/sw/nic/gpuagent/api/smi/amdsmi/smi_api.cc`

**Function**: `smi_fill_ecc_stats_` (around line 991)

```diff
 static void
 smi_fill_ecc_stats_ (aga_gpu_handle_t gpu_handle,
                      aga_gpu_stats_t *stats)
 {
     amdsmi_error_count_t ec;
     amdsmi_status_t amdsmi_ret;
     uint64_t total_correctable_count = 0;
     uint64_t total_uncorrectable_count = 0;
+    uint64_t total_deferred_count = 0;
     
     for (uint32_t b = AMDSMI_GPU_BLOCK_FIRST; b <= AMDSMI_GPU_BLOCK_LAST;
          b = b * 2) {
         ec = { 0 };
         amdsmi_ret = amdsmi_get_gpu_ecc_count(gpu_handle,
                                               (amdsmi_gpu_block_t)(b), &ec);
         if (amdsmi_ret == AMDSMI_STATUS_SUCCESS) {
             total_correctable_count += ec.correctable_count;
             total_uncorrectable_count += ec.uncorrectable_count;
+            total_deferred_count += ec.deferred_count;
             
             switch (b) {
             case AMDSMI_GPU_BLOCK_UMC:
                 stats->umc_correctable_errors = ec.correctable_count;
                 stats->umc_uncorrectable_errors = ec.uncorrectable_count;
+                stats->umc_deferred_errors = ec.deferred_count;
                 break;
             case AMDSMI_GPU_BLOCK_SDMA:
                 stats->sdma_correctable_errors = ec.correctable_count;
                 stats->sdma_uncorrectable_errors = ec.uncorrectable_count;
+                stats->sdma_deferred_errors = ec.deferred_count;
                 break;
             case AMDSMI_GPU_BLOCK_GFX:
                 stats->gfx_correctable_errors = ec.correctable_count;
                 stats->gfx_uncorrectable_errors = ec.uncorrectable_count;
+                stats->gfx_deferred_errors = ec.deferred_count;
                 break;
             // ... similar for all other 15 blocks (MMHUB, ATHUB, BIF, HDP, XGMI_WAFL, 
             //     DF, SMN, SEM, MP0, MP1, FUSE, MCA, VCN, JPEG, IH, MPIO)
             }
         }
     }
     
     stats->total_correctable_errors = total_correctable_count;
     stats->total_uncorrectable_errors = total_uncorrectable_count;
+    stats->total_deferred_errors = total_deferred_count;
 }
```

#### File: `gpuagent/sw/nic/gpuagent/api/smi/gimamdsmi/smi_api.cc`

**Function**: `smi_fill_ecc_stats_` (around line 603)

**Apply the exact same changes as amdsmi/smi_api.cc** (shown above). The GIM AMD-SMI implementation has identical structure.

**Build/Test**:
- [ ] Changes made in gpuagent submodule repository
- [ ] Run `make` in gpuagent directory
- [ ] Test with gpuagent CLI tool to verify deferred errors are populated (both baremetal and SR-IOV)
- [ ] Create PR/commit in gpuagent repository

### 4.2 Device Metrics Exporter Proto Changes

#### File: `pkg/exporter/proto/exporterconfig.proto`

```diff
enum GPUMetricField {
    // ... existing fields
    GPU_VRAM_MAX_BANDWIDTH       = 121;
    
+   // Deferred ECC error metrics
+   GPU_ECC_DEFERRED_TOTAL       = 122;
+   GPU_ECC_DEFERRED_SDMA        = 123;
+   GPU_ECC_DEFERRED_GFX         = 124;
+   GPU_ECC_DEFERRED_MMHUB       = 125;
+   GPU_ECC_DEFERRED_ATHUB       = 126;
+   GPU_ECC_DEFERRED_BIF         = 127;
+   GPU_ECC_DEFERRED_HDP         = 128;
+   GPU_ECC_DEFERRED_XGMI_WAFL   = 129;
+   GPU_ECC_DEFERRED_DF          = 130;
+   GPU_ECC_DEFERRED_SMN         = 131;
+   GPU_ECC_DEFERRED_SEM         = 132;
+   GPU_ECC_DEFERRED_MP0         = 133;
+   GPU_ECC_DEFERRED_MP1         = 134;
+   GPU_ECC_DEFERRED_FUSE        = 135;
+   GPU_ECC_DEFERRED_UMC         = 136;
+   GPU_ECC_DEFERRED_MCA         = 137;
+   GPU_ECC_DEFERRED_VCN         = 138;
+   GPU_ECC_DEFERRED_JPEG        = 139;
+   GPU_ECC_DEFERRED_IH          = 140;
+   GPU_ECC_DEFERRED_MPIO        = 141;
}
```

### 4.3 Device Metrics Exporter Implementation Files

#### gpuagent_gpu_metrics.go

**Location**: `pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go`

**Changes Required:**

1. **Add GaugeVec fields to GpuMetrics struct** (around lines 65-256):

```go
type GpuMetrics struct {
    // ... existing ECC metrics
    gpuEccCorrectTotal       prometheus.GaugeVec
    gpuEccUncorrectTotal     prometheus.GaugeVec
    
    // Add deferred error metrics
    gpuEccDeferredTotal      prometheus.GaugeVec
    gpuEccDeferredSdma       prometheus.GaugeVec
    gpuEccDeferredGfx        prometheus.GaugeVec
    gpuEccDeferredMmhub      prometheus.GaugeVec
    gpuEccDeferredAthub      prometheus.GaugeVec
    gpuEccDeferredBif        prometheus.GaugeVec
    gpuEccDeferredHdp        prometheus.GaugeVec
    gpuEccDeferredXgmiWafl   prometheus.GaugeVec
    gpuEccDeferredDf         prometheus.GaugeVec
    gpuEccDeferredSmn        prometheus.GaugeVec
    gpuEccDeferredSem        prometheus.GaugeVec
    gpuEccDeferredMp0        prometheus.GaugeVec
    gpuEccDeferredMp1        prometheus.GaugeVec
    gpuEccDeferredFuse       prometheus.GaugeVec
    gpuEccDeferredUmc        prometheus.GaugeVec
    gpuEccDeferredMca        prometheus.GaugeVec
    gpuEccDeferredVcn        prometheus.GaugeVec
    gpuEccDeferredJpeg       prometheus.GaugeVec
    gpuEccDeferredIh         prometheus.GaugeVec
    gpuEccDeferredMpio       prometheus.GaugeVec
}
```

2. **Register metrics in initPrometheusMetrics()** (around lines 721-1200):

```go
func (ga *GpuAgent) initPrometheusMetrics() error {
    // ... existing metric registrations
    
    // Deferred ECC error metrics
    gpuEccDeferredTotal: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "amd_gpu_ecc_deferred_total",
        Help: "Total accumulated deferred ECC errors across all GPU blocks",
    }, labels),
    
    gpuEccDeferredSdma: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
        Name: "amd_gpu_ecc_deferred_sdma",
        Help: "Accumulated deferred ECC errors in SDMA block",
    }, labels),
    
    // ... (similar for all 18 per-block deferred errors)
}
```

3. **Add to field registration in initFieldRegistration()**

#### gpuagent_gpu.go

**Location**: `pkg/amdgpu/gpuagent/gpuagent_gpu.go`

**Changes Required:**

Add value extraction in `updateGPUInfoToMetrics()` function (around line 2003):

```go
func (ga *GpuAgent) updateGPUInfoToMetrics(ctx context.Context, gpu *gpuproto.GPU) error {
    // ... existing ECC error collection
    
    // Total deferred errors
    ga.fl.logWithValidateAndExport(
        gpuid,
        ga.metrics.gpuEccDeferredTotal,
        exportermetrics.GPUMetricField_GPU_ECC_DEFERRED_TOTAL.String(),
        labels,
        float64(stats.TotalDeferredErrors),
    )
    
    // SDMA deferred errors
    ga.fl.logWithValidateAndExport(
        gpuid,
        ga.metrics.gpuEccDeferredSdma,
        exportermetrics.GPUMetricField_GPU_ECC_DEFERRED_SDMA.String(),
        labels,
        float64(stats.SDMADeferredErrors),
    )
    
    // ... (similar for all 18 per-block deferred errors)
}
```

**Note**: The `logWithValidateAndExport()` function automatically handles:
- Zero value detection (via `utils.IsNonZeroValue`)
- Unsupported field logging (via field logger)
- Metric export (sets Prometheus gauge value)

### 4.4 File Checklist

**GPUAgent Submodule (git@github.com:ROCm/gpu-agent.git):**
- [ ] `gpuagent/sw/nic/gpuagent/protos/gpu.proto` - Add 19 deferred error fields to GPUStats
- [ ] `gpuagent/sw/nic/gpuagent/api/smi/amdsmi/smi_api.cc` - Populate deferred_count from AMD-SMI in smi_fill_ecc_stats_ (baremetal)
- [ ] `gpuagent/sw/nic/gpuagent/api/smi/gimamdsmi/smi_api.cc` - Populate deferred_count from AMD-SMI in smi_fill_ecc_stats_ (SR-IOV/GIM)
- [ ] Build and test gpuagent submodule (both baremetal and SR-IOV modes)
- [ ] Create PR/commit in gpuagent repository

**Device Metrics Exporter:**
- [ ] Update gpuagent submodule pointer to include deferred error changes
- [ ] `pkg/exporter/proto/exporterconfig.proto` - Add 19 enum entries (GPU_ECC_DEFERRED_*)
- [ ] `pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go` - Add 19 GaugeVec fields and register them
- [ ] `pkg/amdgpu/gpuagent/gpuagent_gpu.go` - Add 19 logWithValidateAndExport calls

**Proto Compilation:**
- [ ] Run `make gen` in gpuagent submodule to regenerate protobuf code
- [ ] Run `make gen` in device-metrics-exporter to regenerate protobuf code
- [ ] Verify generated Go code compiles without errors

**Note**: No changes needed to `gpumetricssvc.proto` (health service) as deferred errors will NOT be used for health determination.

---

---

## 5. Testing Requirements

### 5.1 Unit Tests

**File**: `pkg/amdgpu/gpuagent/gpuagent_test.go`

**Test Cases:**
- [ ] Test metric collection with mock deferred error values
- [ ] Test zero value handling (field logger marks as unsupported)
- [ ] Test field mapping and extraction
- [ ] Test Prometheus metric registration
- [ ] Verify all 19 metrics are registered correctly
- [ ] Test that deferred errors do NOT affect GPU health status

### 5.2 Integration Tests

**Platform-Specific Tests:**
- [ ] Test on MI2xx platform (non-partitioned)
- [ ] Test on MI3xx platform (non-partitioned)

**Partitioned GPU Tests:**
- [ ] Test on MI3xx in SPX mode (non-partitioned) → partition 0 exports metrics
- [ ] Test on MI3xx in CPX mode (8 partitions) → only partition 0 exports metrics, partitions 1-7 skip
- [ ] Test on MI3xx in DPX mode (2 partitions) → only partition 0 exports metrics, partition 1 skips
- [ ] Test on MI3xx in QPX mode (4 partitions) → only partition 0 exports metrics, partitions 1-3 skip
- [ ] Verify field logger marks secondary partitions as unsupported (logs once, then skips)

**Functional Tests:**
- [ ] Verify metrics appear in Prometheus /metrics endpoint
- [ ] Verify correct labels attached to metrics (gpu_id, gpu_uuid, hostname)
- [ ] Verify metric values match AMD-SMI output (partition 0 only)
- [ ] Test metric with different GPU configurations (single GPU, multi-GPU)
- [ ] Test metric persistence across exporter restarts
- [ ] Verify configuration-based metric filtering works

### 5.3 SR-IOV/Hypervisor Tests

- [ ] Test on hypervisor with GIM driver
- [ ] Verify AMD-SMI exposes deferred_count field in SR-IOV mode
- [ ] Verify metrics are exported correctly in SR-IOV environment

### 5.4 Performance and Stress Tests

- [ ] Measure metric collection overhead (target: < 1% CPU overhead)
- [ ] Test with 8+ GPUs (multi-GPU performance)
- [ ] 24-hour stress test (no memory leaks)
- [ ] Kubernetes DaemonSet deployment
- [ ] Bare metal deployment
- [ ] SR-IOV mode deployment

### 5.5 System Test Validation Criteria

This section defines high-level blackbox automation test requirements for system/acceptance testing.

#### 5.5.1 Functional Validation

**Config Behavior:**
- Enable `GPU_ECC_DEFERRED_*` fields in `/etc/metrics/config.json`
- Restart exporter (or wait for 3s auto-reload)
- Verify all 19 deferred error metrics appear in `/metrics` endpoint

**Expected Prometheus Output:**
```bash
curl -s http://localhost:2112/metrics | grep amd_gpu_ecc_deferred
```

Should return all 19 metrics with labels: `gpu_id`, `gpu_uuid`, `hostname`

**Workload Scenario:**
- Deferred errors are **static metrics** (accumulated counters, not workload-dependent)
- Values change only when ECC errors occur in hardware (rare event)
- For testing: Use `metricsclient` for mock injection or `AMDGPURAS` for real HW injection (see Special Cases)

**Static vs Dynamic:**
- **Static**: Deferred error counts do not change with normal workload activity
- Values increment only when uncorrectable errors are deferred by hardware
- For functional validation: Baseline read → inject error → verify counter increment
- **Reference**: See [docs-internal/knowledge/exporter/gpu-metrics-details.md](../../exporter/gpu-metrics-details.md) for detailed explanation of static vs dynamic metrics

#### 5.5.2 Metric Accuracy Validation

**Reference Tool:**
```bash
amd-smi metric -e
```

**Validation Method:**
1. Query `amd-smi metric -e` for deferred error counts per block
2. Compare with Prometheus metrics from `/metrics` endpoint
3. Expected match: **Exact match** (deferred_count should match exactly)

**Tolerance:**
- **Exact match** required (deferred errors are accumulated counters, no sampling variance)
- If AMD-SMI shows `UMC deferred_count: 42`, Prometheus `amd_gpu_ecc_deferred_umc` must be `42`

**Example Validation:**
```bash
# AMD-SMI
amd-smi metric -e | grep -A 5 "GPU 0"
# UMC: correctable=0 uncorrectable=0 deferred=42

# Prometheus
curl -s localhost:2112/metrics | grep 'amd_gpu_ecc_deferred_umc{gpu_id="0"}'
# amd_gpu_ecc_deferred_umc{gpu_id="0",...} 42
```

#### 5.5.3 Negative Test Cases

**Test 1: Disabled Metric in Config**
- Remove `GPU_ECC_DEFERRED_*` from config.json fields array
- Reload config (3s auto-reload or restart)
- Verify: Metrics should NOT appear in `/metrics` endpoint

**Test 2: Unsupported Platform**
- Run on platform where `deferred_count` field returns 0 (if such platform exists)
- Expected: Field logger marks as unsupported (logs once), metric skipped from export
- Check logs for: `"field <metric> not supported on this platform"`

**Test 3: Invalid/Out-of-Range Values**
- AMD-SMI should never return negative deferred_count (uint64)
- If AMD-SMI fails, gpuagent returns 0 for all ECC fields
- Verify: Exporter handles gracefully (no crashes), exports 0 or skips metric

**Test 4: Driver Compatibility**
- **Baremetal**: Test with amdgpu driver < 6.4.x (if available)
- **SR-IOV**: Test with GIM driver < 8.3.0.K (if available)
- Expected behavior: Field logger marks as unsupported, or AMD-SMI returns 0
- Exporter should not crash, gracefully handle missing field

#### 5.5.4 Platform-Specific Validation

**MI2xx vs MI3xx:**
- Both platforms should support deferred errors (amdgpu 6.4+)
- No behavioral difference expected between MI210, MI250, MI250X, MI300A, MI300X
- Verify: All 19 metrics exported on both platform families

**Hypervisor vs Baremetal:**
- **Baremetal**: AMD-SMI via `amdsmi` driver (amdgpu 6.4.x+)
- **Hypervisor/SR-IOV**: AMD-SMI via `gimamdsmi` driver (GIM 8.3.0.K+)
- **Test Required**: Verify deferred_count field is populated in SR-IOV mode with GIM 8.3.0.K+
- Mark metricslist.md Hypervisor column as &check; or &cross; based on test result
- Both implementations use identical `smi_fill_ecc_stats_()` logic

**Partitioned GPU (CPX/DPX/QPX):**
- **PRIMARY PARTITION ONLY**: ECC deferred errors are ONLY available on partition 0
- Secondary partitions (1-7) do not expose ECC error counts via AMD-SMI
- Verify: Only partition 0 exports deferred error metrics
- Verify: Partitions 1-7 skip deferred error metrics (field logger marks as unsupported)

**Test Cases:**
- **SPX mode (non-partitioned)**: Single GPU (partition 0) → exports all 19 metrics
- **CPX mode (8 partitions)**: Partition 0 → exports all 19 metrics, Partitions 1-7 → skip metrics
- **DPX mode (2 partitions)**: Partition 0 → exports all 19 metrics, Partition 1 → skips metrics
- **QPX mode (4 partitions)**: Partition 0 → exports all 19 metrics, Partitions 1-3 → skip metrics

**Validation:**
```bash
# On partitioned GPU (e.g., CPX mode with 8 partitions)
curl -s localhost:2112/metrics | grep amd_gpu_ecc_deferred_total

# Expected: Only partition 0 shows metrics
amd_gpu_ecc_deferred_total{gpu_id="0",...} 42
# Partitions 1-7 should NOT appear in output
```

#### 5.5.5 Special Cases

**ECC Error Injection Testing:**

**Option 1: metricsclient (Mock Injection - Safe, Recommended for Testing)**
- Tool: `metricsclient` (part of device-metrics-exporter testing tools)
- Usage:
  ```bash
  metricsclient set-error --gpu-id 0 --error-type GPU_ECC_DEFERRED_UMC --count 42
  ```
- Injects mock deferred error value without touching real hardware
- Safe, reversible, designed for automated testing
- See: [docs-internal/knowledge/exporter/gpu-metrics-details.md](../../exporter/gpu-metrics-details.md) - ECC Error Injection section

**Option 2: AMDGPURAS (Real HW Injection - Risky, Platform-Specific)**
- Tool: AMD GPU RAS (Reliability, Availability, Serviceability) utility
- Injects real ECC errors into GPU hardware blocks
- **WARNING**: Uncorrectable errors may crash node, use with caution
- Check platform support: `amd-smi -ecc`
- Only use if metricsclient is insufficient for validation

**Validation Steps:**
1. Baseline: Read current deferred error counts from `/metrics`
2. Inject: Use metricsclient to inject deferred errors for specific block (e.g., UMC)
3. Verify: Curl `/metrics` and confirm deferred error count incremented
4. Reset: Use metricsclient to reset error counts to baseline

**Example Test Flow:**
```bash
# Baseline
curl -s localhost:2112/metrics | grep amd_gpu_ecc_deferred_umc
# amd_gpu_ecc_deferred_umc{gpu_id="0"} 0

# Inject
metricsclient set-error --gpu-id 0 --error-type GPU_ECC_DEFERRED_UMC --count 5

# Verify
curl -s localhost:2112/metrics | grep amd_gpu_ecc_deferred_umc
# amd_gpu_ecc_deferred_umc{gpu_id="0"} 5

# Reset
metricsclient reset-error --gpu-id 0
```

**Reference Documentation:**
- [docs-internal/knowledge/exporter/gpu-metrics-details.md](../../exporter/gpu-metrics-details.md) - Static vs dynamic metrics, ECC error injection procedures
- [internal/metricsmap.md](../../internal/metricsmap.md) - Critical metrics classification (deferred errors are NOT critical)

#### 5.5.6 Test Automation Summary

All tests in this section should be automated for CI/CD integration:

- **Functional tests**: Config enable/disable, /metrics endpoint verification
- **Accuracy tests**: Automated comparison with amd-smi output
- **Negative tests**: Config validation, unsupported platform handling
- **Platform tests**: MI2xx vs MI3xx, Baremetal vs Hypervisor
- **Injection tests**: metricsclient-based error injection (scripted)

**No manual testing required** - all validation can be automated.

---

## 6. Documentation Updates

### 6.1 Files to Update

**User Documentation:**
- [ ] `docs/configuration/metricslist.md` - Add deferred error metrics to ECC section
- [ ] `docs/index.md` - Update compatibility matrix if driver/platform requirements changed
- [ ] `docs/releasenotes.md` - Add release notes entry
- [ ] `example/config.json` - Add deferred error configuration examples

**Developer Documentation:**
- [ ] `internal/metricsmap.md` - Add metric mapping rows (Exporter Metric | GPU Agent Field | amd-smi Field | Platform)
  - **Note**: Do NOT add to Critical Metrics list (these are standard metrics for monitoring, not critical for workload evaluation)
- [ ] `docs-internal/knowledge/exporter/gpu-metrics-details.md` - Reference for ECC error injection procedures and static/dynamic metric classification

### 6.2 Metrics List Documentation

**File**: `docs/configuration/metricslist.md`

Add to the "### ECC Error Metrics" section (after line 195):

```markdown
| Hypervisor | Baremetal | Metric                                       | Description                          |
|------------|-----------|----------------------------------------------|--------------------------------------|
| TBD        | &check;   | GPU_ECC_DEFERRED_TOTAL `[MI2xx, MI3xx]`      | Total Deferred ECC error count       |
| TBD        | &check;   | GPU_ECC_DEFERRED_SDMA `[MI2xx, MI3xx]`       | Deferred ECC error in SDMA           |
| TBD        | &check;   | GPU_ECC_DEFERRED_GFX `[MI2xx, MI3xx]`        | Deferred ECC error in GFX            |
| TBD        | &check;   | GPU_ECC_DEFERRED_MMHUB `[MI2xx, MI3xx]`      | Deferred ECC error in MMHUB          |
| TBD        | &check;   | GPU_ECC_DEFERRED_ATHUB `[MI2xx, MI3xx]`      | Deferred ECC error in ATHUB          |
| TBD        | &check;   | GPU_ECC_DEFERRED_BIF `[MI2xx, MI3xx]`        | Deferred ECC error in BIF            |
| TBD        | &check;   | GPU_ECC_DEFERRED_HDP `[MI2xx, MI3xx]`        | Deferred ECC error in HDP            |
| TBD        | &check;   | GPU_ECC_DEFERRED_XGMI_WAFL `[MI2xx, MI3xx]`  | Deferred ECC error in XGMI WAFL      |
| TBD        | &check;   | GPU_ECC_DEFERRED_DF `[MI2xx, MI3xx]`         | Deferred ECC error in DF             |
| TBD        | &check;   | GPU_ECC_DEFERRED_SMN `[MI2xx, MI3xx]`        | Deferred ECC error in SMN            |
| TBD        | &check;   | GPU_ECC_DEFERRED_SEM `[MI2xx, MI3xx]`        | Deferred ECC error in SEM            |
| TBD        | &check;   | GPU_ECC_DEFERRED_MP0 `[MI2xx, MI3xx]`        | Deferred ECC error in MP0            |
| TBD        | &check;   | GPU_ECC_DEFERRED_MP1 `[MI2xx, MI3xx]`        | Deferred ECC error in MP1            |
| TBD        | &check;   | GPU_ECC_DEFERRED_FUSE `[MI2xx, MI3xx]`       | Deferred ECC error in FUSE           |
| TBD        | &check;   | GPU_ECC_DEFERRED_UMC `[MI2xx, MI3xx]`        | Deferred ECC error in UMC            |
| TBD        | &check;   | GPU_ECC_DEFERRED_MCA `[MI2xx, MI3xx]`        | Deferred ECC error in MCA            |
| TBD        | &check;   | GPU_ECC_DEFERRED_VCN `[MI2xx, MI3xx]`        | Deferred ECC error in VCN            |
| TBD        | &check;   | GPU_ECC_DEFERRED_JPEG `[MI2xx, MI3xx]`       | Deferred ECC error in JPEG           |
| TBD        | &check;   | GPU_ECC_DEFERRED_IH `[MI2xx, MI3xx]`         | Deferred ECC error in IH             |
| TBD        | &check;   | GPU_ECC_DEFERRED_MPIO `[MI2xx, MI3xx]`       | Deferred ECC error in MPIO           |
```

**Note**: "TBD" in Hypervisor column will be replaced with &check; or &cross; after SR-IOV testing confirms AMD-SMI field availability.

### 6.3 Internal Metrics Map Documentation

**File**: `internal/metricsmap.md`

Add mapping rows (do NOT add to Critical Metrics section):

```markdown
| amd_gpu_ecc_deferred_total      | TotalDeferredErrors      | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_sdma       | SDMADeferredErrors       | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_gfx        | GFXDeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_mmhub      | MMHUBDeferredErrors      | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_athub      | ATHUBDeferredErrors      | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_bif        | BIFDeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_hdp        | HDPDeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_xgmi_wafl  | XGMIWAFLDeferredErrors   | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_df         | DFDeferredErrors         | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_smn        | SMNDeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_sem        | SEMDeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_mp0        | MP0DeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_mp1        | MP1DeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_fuse       | FUSEDeferredErrors       | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_umc        | UMCDeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_mca        | MCADeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_vcn        | VCNDeferredErrors        | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_jpeg       | JPEGDeferredErrors       | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_ih         | IHDeferredErrors         | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
| amd_gpu_ecc_deferred_mpio       | MPIODeferredErrors       | amdsmi_error_count_t.deferred_count | MI2xx, MI3xx |
```

### 6.4 Configuration Example

**Enabling All Deferred Error Metrics:**

```json
{
  "gpu_config": {
    "selector": "all",
    "fields": [
      "GPU_ECC_DEFERRED_TOTAL",
      "GPU_ECC_DEFERRED_SDMA",
      "GPU_ECC_DEFERRED_GFX",
      "GPU_ECC_DEFERRED_MMHUB",
      "GPU_ECC_DEFERRED_ATHUB",
      "GPU_ECC_DEFERRED_BIF",
      "GPU_ECC_DEFERRED_HDP",
      "GPU_ECC_DEFERRED_XGMI_WAFL",
      "GPU_ECC_DEFERRED_DF",
      "GPU_ECC_DEFERRED_SMN",
      "GPU_ECC_DEFERRED_SEM",
      "GPU_ECC_DEFERRED_MP0",
      "GPU_ECC_DEFERRED_MP1",
      "GPU_ECC_DEFERRED_FUSE",
      "GPU_ECC_DEFERRED_UMC",
      "GPU_ECC_DEFERRED_MCA",
      "GPU_ECC_DEFERRED_VCN",
      "GPU_ECC_DEFERRED_JPEG",
      "GPU_ECC_DEFERRED_IH",
      "GPU_ECC_DEFERRED_MPIO"
    ],
    "labels": [
      "GPU_UUID",
      "GPU_ID",
      "HOSTNAME"
    ]
  }
}
```

### 6.5 Release Notes Entry

```markdown
## Release X.Y.Z - [Date]

### New Features

#### ECC Deferred Error Metrics

Added support for 19 new ECC deferred error metrics for memory reliability monitoring.

**Metrics Added:**
- `amd_gpu_ecc_deferred_total` - Total deferred errors across all blocks
- `amd_gpu_ecc_deferred_<block>` - Per-block deferred errors (18 blocks)

**Supported Blocks:**
SDMA, GFX, MMHUB, ATHUB, BIF, HDP, XGMI_WAFL, DF, SMN, SEM, MP0, MP1, FUSE, UMC, MCA, VCN, JPEG, IH, MPIO

**Configuration:**
- Field prefix: `GPU_ECC_DEFERRED_*`
- Type: Non-profiler (collected per scrape)
- Platforms: MI2xx, MI3xx series
- Driver: amdgpu 6.4.x+ (baremetal), GIM 8.3.0.K+ (SR-IOV)
- AMD-SMI: Uses `amdsmi_error_count_t.deferred_count` field

**Use Case:**
Monitor deferred errors for memory reliability tracking and predictive failure analysis.

**Note:** Deferred error metrics are for monitoring only and do NOT affect GPU health service determination.

**Important Limitation:** ECC deferred error metrics are only available on primary partition 0. In partitioned GPU configurations (CPX/DPX/QPX), secondary partitions (1-7) do not expose ECC metrics.

**Dependencies:**
Requires gpuagent submodule update (git@github.com:ROCm/gpu-agent.git).
```

---

## 7. Acceptance Criteria

**GPUAgent Submodule:**
- [ ] 19 proto fields added to gpuagent/protos/gpu.proto
- [ ] smi_fill_ecc_stats_() updated in amdsmi/smi_api.cc (baremetal)
- [ ] smi_fill_ecc_stats_() updated in gimamdsmi/smi_api.cc (SR-IOV/GIM)
- [ ] gpuagent builds without errors
- [ ] gpuagent PR created in ROCm/gpu-agent repository

**Device Metrics Exporter:**
- [ ] gpuagent submodule pointer updated
- [ ] 19 enums added to exporterconfig.proto (indices 122-141)
- [ ] 19 GaugeVecs registered in gpuagent_gpu_metrics.go
- [ ] 19 collection calls in gpuagent_gpu.go
- [ ] Code compiles without errors
- [ ] `make gen` completes successfully

**Functionality:**
- [ ] All 19 metrics appear in /metrics endpoint (partition 0 only)
- [ ] Correct values for supported platforms
- [ ] Unsupported platforms handled gracefully
- [ ] Partitioned GPUs: Only partition 0 exports metrics, partitions 1-7 skip
- [ ] Field logger marks partitions 1-7 as unsupported for ECC metrics
- [ ] Configuration-based filtering works

**Testing:**
- [ ] Unit tests pass
- [ ] Integration tests pass on MI2xx and MI3xx (non-partitioned)
- [ ] Partitioned GPU tests pass (SPX/CPX/DPX/QPX modes)
- [ ] Partition 0 exports metrics correctly
- [ ] Partitions 1-7 skip metrics gracefully
- [ ] SR-IOV testing completed
- [ ] No performance regression
- [ ] 24-hour stress test passes

**Documentation:**
- [ ] Metrics added to docs/configuration/metricslist.md (TBD for Hypervisor column)
- [ ] Compatibility matrix updated in docs/index.md (if driver/platform requirements changed)
- [ ] Mapping added to internal/metricsmap.md (NOT in Critical Metrics section)
- [ ] Reference to docs-internal/knowledge/exporter/gpu-metrics-details.md for ECC special cases
- [ ] Configuration examples added to example/config.json
- [ ] Release notes updated

---

## 8. Known Limitations

### 8.1 GPUAgent Submodule Dependency
- Changes require updates to external gpuagent submodule (git@github.com:ROCm/gpu-agent.git)
- Create PR in gpuagent repository early, coordinate with ROCm team

### 8.2 Platform Support Uncertainty
- Exact platform support for `deferred_count` field not yet verified
- Field logger pattern handles gracefully if unsupported

### 8.3 AMD-SMI Version Unknown
- Minimum AMD-SMI version that populates `deferred_count` not yet determined
- Test with multiple AMD-SMI versions to determine minimum version

### 8.4 SR-IOV/Hypervisor Support Unknown
- SR-IOV support for deferred_count field not yet verified
- Marked as "TBD" in metricslist.md until testing confirms
- Test on SR-IOV system with GIM driver

### 8.5 No Health Service Integration
- Deferred errors will NOT be used in GPU health monitoring
- Users must set up own Prometheus alerting rules

### 8.6 Primary Partition Only Limitation
- **ECC deferred error metrics are ONLY available on primary partition 0**
- Secondary partitions do not expose ECC error counts (hardware/AMD-SMI limitation)
- Partitioned GPU configurations:
  - **CPX mode (8 partitions)**: Partition 0 ✓, Partitions 1-7 ✗
  - **DPX mode (2 partitions)**: Partition 0 ✓, Partition 1 ✗
  - **QPX mode (4 partitions)**: Partition 0 ✓, Partitions 1-3 ✗
  - **SPX mode (non-partitioned)**: Partition 0 ✓ (no limitation)
- Secondary partitions: Field logger marks ECC metrics as unsupported

**Impact**: In multi-partition configurations, only partition 0 provides ECC monitoring visibility. Users monitoring partitioned GPUs must rely on partition 0 ECC metrics for the entire physical GPU.

---

## 9. References

**AMD-SMI Documentation:**
- `libamdsmi/include/amd_smi/amdsmi.h` (line 2096: `amdsmi_error_count_t`)

**GPUAgent Repository:**
- git@github.com:ROCm/gpu-agent.git
- `gpuagent/sw/nic/gpuagent/protos/gpu.proto`
- `gpuagent/sw/nic/gpuagent/api/smi/amdsmi/smi_api.cc`
- `gpuagent/sw/nic/gpuagent/api/smi/gimamdsmi/smi_api.cc`

**Device Metrics Exporter Documentation:**
- [CLAUDE.md](../../CLAUDE.md) - Project quick reference
- [Developer Guide](../../docs/developerguide.md) - Build and development instructions
- [Metrics List](../../docs/configuration/metricslist.md) - User-facing metrics catalog
- [Metrics Map](../../internal/metricsmap.md) - Internal metric mappings and critical metrics list
- [GPU Metrics Details](../../docs-internal/knowledge/exporter/gpu-metrics-details.md) - Static/dynamic metrics, ECC error injection, special cases

**Code References:**
- [pkg/amdgpu/proto/gpu.proto](../../pkg/amdgpu/proto/gpu.proto)
- [pkg/exporter/proto/exporterconfig.proto](../../pkg/exporter/proto/exporterconfig.proto)
- [pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go](../../pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go)
- [pkg/amdgpu/gpuagent/gpuagent_gpu.go](../../pkg/amdgpu/gpuagent/gpuagent_gpu.go)

---

## 10. Approval Sign-off

- [ ] Engineering Lead: _______________ Date: ___________
- [ ] Product Manager: _______________ Date: ___________
- [ ] Test Lead: _______________ Date: ___________
- [ ] Documentation: _______________ Date: ___________

---

**Implementation Start Date**: ___________  
**Target Completion Date**: ___________  
**Actual Completion Date**: ___________
