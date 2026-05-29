# Partition and VM Environment Metrics - AMD Device Metrics Exporter

This document details metric availability in partitioned GPUs and VM/Hypervisor environments.

---

## Overview

**IMPORTANT:** Not all metrics are available in partition/VM environments.

**Metric availability varies based on:**
1. **Deployment Environment:** Hypervisor (VM) vs Baremetal
2. **Partition Mode:** SPX (single partition) vs CPX/DPX/QPX (multi-partition)
3. **Partition ID:** Primary partition (`partition_id=0`) vs Non-primary partitions

**Reference:** [docs/configuration/metricslist.md](../../docs/configuration/metricslist.md) - Complete metric availability matrix

---

## Deployment Environments

### Hypervisor (VM Environment)

**Characteristics:**
- GPU passed through to VM via SR-IOV or similar
- Limited access to physical GPU sensors
- Some power/thermal metrics unavailable

**Metrics Unavailable:**
- `GPU_AVERAGE_PACKAGE_POWER` - Requires physical access
- `GPU_GFX_BUSY_INSTANTANEOUS` - Physical sensor only
- `GPU_VC_BUSY_INSTANTANEOUS` - Physical sensor only
- `GPU_JPEG_BUSY_INSTANTANEOUS` - Physical sensor only

### Baremetal Environment

**Characteristics:**
- Direct hardware access
- Full access to all GPU sensors
- All metrics available (subject to partition mode)

---

## GPU Partition Modes

### SPX (Single Partition - Non-Partitioned)

**Full GPU access:**
- All metrics available
- No partition-specific restrictions
- Standard operating mode

### CPX (Compute Partition)

**Multi-partition mode:**
- GPU divided into multiple compute partitions
- Each partition has `partition_id` (0, 1, 2, ...)
- Primary partition: `partition_id=0`
- Non-primary partitions: `partition_id=1,2,...`

### DPX (Dual Partition)

**Two partitions:**
- Similar to CPX but limited to 2 partitions
- Primary: `partition_id=0`
- Non-primary: `partition_id=1`

### QPX (Quad Partition)

**Four partitions:**
- Similar to CPX but limited to 4 partitions
- Primary: `partition_id=0`
- Non-primary: `partition_id=1,2,3`

---

## Partition-Specific Metric Behavior

### Physical Sensor Metrics (Primary Partition Only)

These metrics access **physical GPU sensors** and are only available on the **primary partition** (`partition_id=0`).

**Temperature Metrics:**
- `GPU_JUNCTION_TEMPERATURE` - Physical GPU sensor
  - Primary partition: Reports actual temperature
  - Non-primary partitions: **Suppressed** (not reported)
- `GPU_MEMORY_TEMPERATURE` - Physical memory sensor
  - Primary partition: Reports actual temperature
  - Non-primary partitions: **Suppressed** (not reported)

**Power Metrics:**
- `GPU_PACKAGE_POWER` - Socket-level power sensor
  - Primary partition: Reports actual power
  - Non-primary partitions: **Reports 0** (no per-XCD power sensor)
- `GPU_POWER_USAGE` - GPU power usage
  - Primary partition: Reports actual power
  - Non-primary partitions: **Reports 0** (no per-XCD power sensor)
- `GPU_ENERGY_CONSUMED` - Accumulated energy
  - Primary partition: Reports actual energy
  - Non-primary partitions: **Suppressed** (physical GPU sensor)

**Activity Metrics:**
- `GPU_GFX_ACTIVITY` - Graphics engine usage
  - Primary partition: Reports activity
  - Non-primary partitions: **Not available** (amdsmi doesn't return per-partition data)
- `GPU_UMC_ACTIVITY` - Memory engine usage
  - Primary partition: Reports activity
  - Non-primary partitions: **Not available** (amdsmi doesn't return per-partition data)

**PCIe Metrics (Physical Link):**
- `PCIE_SPEED` - Current PCIe speed
  - Primary partition: Reports physical link speed
  - Non-primary partitions: **Suppressed** (physical PCIe link)
- `PCIE_MAX_SPEED` - Maximum PCIe speed
  - Primary partition: Reports max speed
  - Non-primary partitions: **Suppressed** (physical PCIe link)
- `PCIE_BANDWIDTH` - Instantaneous bandwidth
  - Primary partition: Reports bandwidth
  - Non-primary partitions: **Suppressed** (physical PCIe link)
- `PCIE_REPLAY_COUNT` - PCIe replays (NAKs)
  - Primary partition: Reports count
  - Non-primary partitions: **Suppressed**
- `PCIE_RECOVERY_COUNT` - PCIe recovery count
  - Primary partition: Reports count
  - Non-primary partitions: **Suppressed**
- `PCIE_REPLAY_ROLLOVER_COUNT` - PCIe replay rollover
  - Primary partition: Reports count
  - Non-primary partitions: **Suppressed**
- `PCIE_NACK_SENT_COUNT` - PCIe NACK sent count
  - Primary partition: Reports count
  - Non-primary partitions: **Suppressed**
- `PCIE_NACK_RECEIVED_COUNT` - PCIe NACK received count
  - Primary partition: Reports count
  - Non-primary partitions: **Suppressed**

### Per-Partition Metrics (All Partitions)

These metrics are available on **all partitions** (primary and non-primary).

**VRAM Metrics:**
- `GPU_TOTAL_VRAM` - Total VRAM per partition
- `GPU_USED_VRAM` - Used VRAM per partition
- `GPU_FREE_VRAM` - Free VRAM per partition

**Compute Metrics:**
- `GPU_COMPUTE_PARTITION_ID` - Current partition ID
- `GPU_MEMORY_PARTITION_ID` - Memory partition ID
- `GPU_PROCESS_CU_OCCUPANCY` - CU occupancy per partition

**Profiler Metrics (Per-Partition):**
- Profiler metrics (`GPU_PROF_*`) are per-partition when available
- May be restricted on non-primary partitions based on PTL

---

## Metric Availability Matrix Format

**Documentation Reference:** [docs/configuration/metricslist.md](../../docs/configuration/metricslist.md)

### Table Format:

| Hypervisor | Baremetal | Metric | Description |
|------------|-----------|--------|-------------|
| &check;    | &check;   | METRIC_NAME | Description with partition notes |

**Checkmarks:**
- &check; - Supported in this environment
- &cross; - Not supported in this environment

**Partition Annotations in Description:**
- "In partitioned mode (CPX/DPX/QPX) applicable for primary partition (`partition_id=0`)"
- "suppressed for all other partitions"
- "all other partitions report 0"

---

## Implementation Considerations

### Code Handling

**For metrics restricted to primary partition:**

```go
// Example: GPU_JUNCTION_TEMPERATURE (primary partition only)
if partitionID != 0 {
    // Non-primary partition: suppress metric or log warning
    logger.Debug("GPU_JUNCTION_TEMPERATURE not available on non-primary partition")
    return
}
// Primary partition: collect and report metric
temperature := gpu.Stats.JunctionTemperature
```

**For metrics that report 0 on non-primary:**

```go
// Example: GPU_PACKAGE_POWER (primary reports actual, others report 0)
if partitionID != 0 {
    // Non-primary partition: return 0 (no per-XCD power sensor)
    return 0
}
// Primary partition: return actual power
return gpu.Stats.PackagePower
```

### Configuration

**No special config required** - Partition detection is automatic via gpuagent.

**Partition Info Available:**
- `GPU_COMPUTE_PARTITION_ID` - Label with partition ID
- `GPU_MEMORY_PARTITION_ID` - Label with memory partition ID

---

## Testing in Partition Environments

### Setup Partitions

```bash
# Create CPX mode (2 compute partitions)
amd-smi partition --set-compute-partition CPX --gpu 0

# Create DPX mode (2 dual partitions)
amd-smi partition --set-compute-partition DPX --gpu 0

# Create QPX mode (4 quad partitions)
amd-smi partition --set-compute-partition QPX --gpu 0

# Reset to SPX (single partition)
amd-smi partition --set-compute-partition SPX --gpu 0
```

### Verify Partition Metrics

```bash
# Query all GPUs (shows all partitions)
curl http://localhost:5000/metrics | grep partition_id

# Expected output (CPX mode):
amd_gpu_total_vram{gpu_id="0",partition_id="0",...} 65536
amd_gpu_total_vram{gpu_id="1",partition_id="1",...} 32768
```

### Test Physical Sensor Metrics

```bash
# GPU_JUNCTION_TEMPERATURE should only appear for partition_id=0
curl http://localhost:5000/metrics | grep junction_temperature

# Expected (CPX mode):
amd_gpu_junction_temperature{gpu_id="0",partition_id="0",...} 45
# No entry for partition_id=1 (suppressed)
```

---

## Common Issues

### Issue: Metrics Missing on Non-Primary Partition

**Symptom:**
```
# Expected metric missing for partition_id=1
curl http://localhost:5000/metrics | grep -A1 partition_id=\"1\"
# GPU_JUNCTION_TEMPERATURE not present
```

**Solution:**
This is **expected behavior**. Physical sensor metrics (temperature, power, PCIe) are only available on primary partition (`partition_id=0`).

**Verify:**
- Check [docs/configuration/metricslist.md](../../docs/configuration/metricslist.md) for metric availability
- Look for "applicable for primary partition" in description
- Non-primary partitions should use per-partition metrics (VRAM, CU occupancy)

### Issue: Power Metrics Return 0 on Non-Primary

**Symptom:**
```
amd_gpu_package_power{gpu_id="1",partition_id="1",...} 0
```

**Solution:**
This is **expected behavior**. No per-XCD power sensors available. Only primary partition reports actual power.

---

## PRD Requirements

When adding new GPU metrics, **always specify partition behavior:**

### Questions to Answer:

1. **Is this metric available in all partition modes?**
   - Yes: Available in SPX, CPX, DPX, QPX
   - No: Specify which modes (e.g., "SPX only", "CPX/DPX/QPX only")

2. **Is this a physical sensor metric?**
   - Yes: Only available on primary partition (`partition_id=0`)
   - No: Available on all partitions

3. **Behavior on non-primary partitions:**
   - Suppressed (not reported)
   - Reports 0
   - Reports per-partition value
   - Not applicable (SPX only)

4. **Is this metric available in VM/Hypervisor environments?**
   - Yes: Available in both VM and baremetal
   - No: Baremetal only

### PRD Documentation Template:

```markdown
## Partition Support

**Partition Modes:** [SPX / CPX / DPX / QPX / All]

**Primary Partition (partition_id=0):**
- Behavior: [Reports actual value / Suppressed / N/A]

**Non-Primary Partitions (partition_id≠0):**
- Behavior: [Reports per-partition value / Reports 0 / Suppressed / N/A]

**Hypervisor/VM Support:** [Yes / No / Baremetal only]

**Justification:**
[Explain why metric is/isn't available in partitions]
[Note if physical sensor, per-XCD limitation, etc.]
```

---

## Related Documentation

- **User Documentation:** [docs/configuration/metricslist.md](../../docs/configuration/metricslist.md) - Complete availability matrix
- **GPU Metrics Details:** [gpu-metrics-details.md](gpu-metrics-details.md) - Static vs dynamic metrics
- **Configuration:** [configuration.md](configuration.md) - Config examples

---

**Last Updated:** 2026-04-06
