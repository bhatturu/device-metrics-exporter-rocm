---
name: bkc-upgrade
description: Upgrade BKC firmware/software on dev servers and verify DME metrics. Use when the user says "upgrade BKC", "check firmware", "check BKC", or wants to run amd-smi/gpuctl checks on dev hardware.
---

# BKC Upgrade Skill

Upgrade and verify BKC firmware and software components on dev test servers.

## Connectivity Check

Always check reachability before attempting work.

## Deployment Workflow

### Step 1 — Identify target image

Ask the user which DME image tag to deploy if not specified.

Pull and transfer if missing:
```bash
docker pull <image>:<tag>
docker save <image>:<tag> | gzip > /tmp/dme-image.tar.gz
# Transfer to target and load
```

### Step 2 — Start container

```bash
docker rm -f dme-dev 2>/dev/null
docker run -d --name dme-dev \
  --device /dev/kfd \
  $(ls /dev/dri/renderD* | sed "s/^/--device /") \
  --group-add video \
  -p 5098:5000 \
  -v /etc/metrics/config.json:/etc/metrics/config.json \
  <IMAGE>:<TAG>
```

Wait 8 seconds for gpuagent to initialize.

### Step 3 — Verify firmware via amd-smi

Run inside the container:
```bash
docker exec dme-dev amd-smi version
docker exec dme-dev amd-smi firmware
```

Run IFWI check on the host: `amd-smi static -I` (flag `-I`, not `firmware` subcommand)

Key firmware versions to check:

| Firmware | gfx1201 (Navi48) | gfx950 (MI350P) |
|----------|------------------|-----------------|
| IFWI VERSION | `023.008.000.068.000001` | `00185129` (BRP0800) |
| IFWI BUILD_DATE | `2025/10/23` | `2026/04/27` |
| PSP_SOSDRV | `00.3A.0F.14` | `00.45.15.2B` |
| PM (SMU) | `00.104.74.00` | `00.86.61.00` |

### Step 4 — Verify gpuctl and metrics

```bash
# Check GPU status
docker exec dme-dev gpuctl show gpu -s 2>&1

# Check GPU statistics (energy, violation, process)
docker exec dme-dev gpuctl show gpu statistics 2>&1

# Check metrics endpoint
curl -s localhost:5098/metrics | grep "^[a-z]" | cut -d'{' -f1 | sort -u

# Check specific metrics
curl -s localhost:5098/metrics | grep -E "energy|violation|process|vram_max"
```

### Step 5 — BKC version comparison (MI350P / gfx950)

BKC 9.0 requirements (check the MI350P BKC PDF for the latest spec):

| Component | BKC 9.0 Requirement | Current | Status |
|-----------|--------------------|--------------:|--------|
| amdgpu-dkms build | `2328720` | `2328720` | ✓ |
| ROCm | `7.13.0a20260427` (therock) | `7.12.0` | ✗ needs upgrade |
| IFWI PART_NUMBER | `113-350P-01-1K1-0800` (BRP0800) | `113-350P-01-1K1-0800` | ✓ (flashed 2026-05-20) |
| IFWI VERSION | `00185129` | `00185129` | ✓ (flashed 2026-05-20) |
| IFWI BUILD_DATE | `2026/04/27` | `2026/04/27` | ✓ |
| Kernel | Ubuntu 24.04 LTS | `6.8.0-111-generic` | ✓ |
| BMC FW | `v2.16.0.95` | `v2.17` | ✓ (newer) |

**To upgrade MI350P ROCm to 7.13:**
```bash
# Use install-rocm-tarball.sh from the docker/testrunner directory
# Tarball: therock-dist-linux-gfx950-dcgpu-7.13.0a20260427.tar.gz
# From: https://therock-nightly-tarball.s3.amazonaws.com/
```
amdgpu-dkms is already at build `2328720` (BKC 9.0 driver) — no driver reinstall needed.

### Step 6 — Report results

For each server, report:
- Image deployed: `<image>:<tag>`
- GPU count detected
- amdsmi version
- Key firmware versions (PSP_SOSDRV, PM/SMU)
- Metrics endpoint status (total metric count)
- Any unexpected sentinel values (UINT64_MAX, 65535, 4294967295)

## IFWI Upgrade Procedure (MI350P)

**Tool:** `amdvbflash` (v5.0.916)
**IFWI file naming:** `MI350P_MI350P_Baseline_BRP<build>_<version>.sbin`

### Finding the IFWI file

The MI350P BKC PDF lists the IFWI index as a web link (not SMB):
- Index path: `dgpu-spiromfw/mi350P/pre_release/ifwi/BRP<build>/`
- 600W / CM heatsink variant: `MI350P_MI350P_Baseline_BRP<build>_<version>.sbin`
- 450W variant: `MI350P_450W_MI350P_Baseline_BRP<build>_<version>.sbin`
- SMB paths in PDF (`\\atlcorpnetfs\BiosOnly\`) are for host BIOS/BMC only, NOT IFWI

Copy the `.sbin` to the target server before flashing:
```bash
scp <local-sbin-file> <user>@<host>:/root/
```

### Flash procedure (requires 2 passes + 2 reboots)

Must be initiated after a fresh reboot. Each pass flashes one SPIROM partition.

```bash
# Pre-flash: verify tool detects GPU and file matches ASIC
amdvbflash -i                                           # shows detected GPUs and current part number
amdvbflash --show --ifwi-info --ifwi-file <file.sbin>  # verify DID matches (0x75A8 for MI350P)

# Pass 1 — flash first SPIROM partition
amdvbflash -p 0 <file.sbin> --show-progress
# Expected: "Flashed 0x6f0000 Bytes Successfully on GPU at [0]"
# Reboot

# Pass 2 — after reboot, flash second SPIROM partition (same command)
amdvbflash -p 0 <file.sbin> --show-progress
# Expected: "Flashed 0x6f0000 Bytes Successfully on GPU at [0]"
# Reboot

# Verify — after final reboot
amd-smi static -I
# PART_NUMBER and VERSION should match the sbin filename
```

**Version naming:** `PART_NUMBER` suffix = BRP build (e.g. `113-350P-01-1K1-0800` → BRP0800).
`VERSION` hex (e.g. `00185129`) matches the `_185129` suffix in the sbin filename.

**WARNING:** Never flash an MI350P `.sbin` onto a different ASIC. The tool detects DID mismatch — do not force-flash with `-f`.

## Known Platform Behaviors

### gfx1201 (Navi48)
- `gpu_energy_consumed` — absent (amdsmi returns 0, DME suppresses)
- `gpu_violation_*` — absent (not supported in hardware)
- `gpu_vram_max_bandwidth` — reports UINT64_MAX sentinel (18446744073709551615)
- `kfd_process_id` label — empty at idle, populated when workload runs
- `GFX/Memory activity accumulated` — returns UINT32_MAX (4294967295) sentinel
- `Power usage` in gpuctl stats — returns 65535 sentinel
- `driver_version`, `vbios_version`, `serial_number` labels — empty strings

### gfx950 (MI350P)
- `gpu_total_vram` works correctly (147440 MB) — was 0 in BRP0620, fixed in BRP0800
- `gpu_total_visible_vram` works correctly
- `gpu_average_package_power` (N/A) and `edge temperature` (N/A) — still firmware gaps in BRP0800/00185129
- `gpu_package_power` (current socket power) works at ~104W
- Run k8s-e2e tests on gfx950 (k3s kubeconfig at `/etc/rancher/k3s/k3s.yaml`)
- Partition modes: SPX, DPX, CPX supported; QPX not available on this hardware

## Cleanup

```bash
docker rm -f dme-dev
```
