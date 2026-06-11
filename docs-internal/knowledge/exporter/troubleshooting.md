# Troubleshooting Guide - AMD Device Metrics Exporter

This guide covers common failure scenarios, diagnostic steps, and solutions for debugging DME issues.

---

## Primary Failure Mode: Driver Load Timing Issues

### Scenario

Boot race conditions between amdgpu driver and gpuagent startup.

### Symptoms

- DME reports 0 GPUs on startup
- `ErrZeroGPUs` error in logs
- All GPUs marked unhealthy
- Health service returns unhealthy for all devices

### Root Causes

1. **amdgpu driver loads late** after DME/gpuagent starts
2. **Driver initialization slow** - takes longer than expected
3. **KFD device not ready** - `/dev/kfd` doesn't exist yet
4. **Boot race** - gpuagent starts before kernel driver fully loaded

### Solutions

**With `-exit-on-agent-down` flag:**
- DME exits after 3 consecutive failures
- K8s restarts pod automatically
- DME retries once driver ready
- Eventually succeeds when driver loaded

**Without flag:**
- DME stays running
- Marks all GPUs as unhealthy
- Reconnects on next health poll (30s default)
- May recover without restart

**Systemd Service:**
- Add `After=amdgpu.target` to service file
- Ensures driver loaded before DME starts

### Prevention

**Container Entrypoint:**
```bash
#!/bin/bash
# Wait for KFD device
while [ ! -e /dev/kfd ]; do
    echo "Waiting for /dev/kfd..."
    sleep 1
done

# Start DME
exec /usr/bin/amd-metrics-exporter "$@"
```

**K8s Liveness/Readiness Probes:**
```yaml
livenessProbe:
  httpGet:
    path: /metrics
    port: 5000
  initialDelaySeconds: 30
  periodSeconds: 10
  
readinessProbe:
  httpGet:
    path: /metrics
    port: 5000
  initialDelaySeconds: 15
  periodSeconds: 5
```

### Diagnostic Commands

```bash
# Check DME logs
journalctl -u amd-metrics-exporter -f

# Check kernel logs for amdgpu driver
dmesg | grep amdgpu

# Check KFD device
ls -la /dev/kfd

# Check driver loaded
lsmod | grep amdgpu

# Check GPU devices
ls -la /dev/dri/

# Test gpuagent connection
gpuctl -s /var/run/gpuagent.sock get gpus
```

---

## GPU Agent Communication Failures

### Scenario

DME cannot connect to gpuagent socket.

### Symptoms

- `ErrAgentUnreachable` error in logs
- All GPU metrics return 0 or missing
- Health service returns unhealthy for all GPUs
- No response from health queries

### Root Causes

1. **gpuagent process crashed** or not started
2. **Socket path mismatch** - DME and gpuagent using different paths
3. **Permission issues** on socket file
4. **Mount namespace mismatch** - gpuagent in different container namespace

### Solutions

**Check gpuagent process:**
```bash
ps aux | grep gpuagent
# Should see: /usr/bin/gpuagent -s /var/run/gpuagent.sock
```

**Verify socket path:**
```bash
ls -la /var/run/gpuagent.sock
# Should exist with correct permissions

# Check DME socket path
ps aux | grep amd-metrics-exporter | grep -- '-s'
```

**Check socket permissions:**
```bash
# Socket should be accessible to DME user
stat /var/run/gpuagent.sock

# Fix permissions if needed
chmod 666 /var/run/gpuagent.sock
```

**Restart gpuagent:**
```bash
# Systemd
systemctl restart gpuagent

# K8s (restart pod)
kubectl delete pod <gpuagent-pod>
```

**Mount namespace (containers):**
```yaml
# Ensure socket in shared mount
volumes:
  - name: gpuagent-socket
    hostPath:
      path: /var/run
      type: Directory
volumeMounts:
  - name: gpuagent-socket
    mountPath: /var/run
```

### Diagnostic Commands

```bash
# Test gpuagent connection with gpuctl
gpuctl -s /var/run/gpuagent.sock get gpus

# Check DME socket configuration
cat /etc/metrics/config.json | jq .

# Test gRPC connection
grpcurl -plaintext -unix /var/run/gpuagent.sock list

# Check socket in use
lsof /var/run/gpuagent.sock
```

---

## ROCProfiler Failures

### Scenario

rocpctl crashes or times out repeatedly.

### Symptoms

- Profiler metrics (GPU_PROF_*) missing or 0
- Logs show "rocpctl timeout" or "profiler disabled after 3 failures"
- GPU workloads may hang or crash
- Core dumps from rocpctl process

### Root Causes

1. **ROCprofiler-SDK incompatible** with GPU model
2. **PTL delay too short** for MI300 series
3. **GPU driver unstable** or outdated
4. **Concurrent workload** interfering with profiler lock
5. **Platform not supported** - profiler only works with amdgpu driver

### Solutions

**Disable profiler in config:**
```json
{
  "GPUConfig": {
    "ProfilerMetrics": {
      "all": false
    }
  }
}
```

**Increase PTL delay for MI300:**
```json
{
  "GPUConfig": {
    "ProfilerConfig": {
      "SamplingInterval": 1000,
      "PtlDelay": 100
    }
  }
}
```

**Update driver:**
```bash
# Check driver version
modinfo amdgpu | grep version

# Update to latest ROCm
# Follow ROCm installation guide for your distro
```

**Check for core dumps:**
```bash
dmesg | grep rocpctl
coredumpctl list | grep rocpctl

# Analyze core dump
coredumpctl debug <PID>
```

**Verify GPU model support:**
```bash
# Profiler only works with amdgpu driver platforms
rocminfo | grep "Name:" | head -1

# Check if platform is supported
rocpctl --help
```

### Prevention

- **Start with profiler disabled**, enable per-node after testing
- **Monitor logs** for 3 consecutive failures before auto-disable
- **Test on single node** before rolling out
- **Check GPU model compatibility** before enabling

### Diagnostic Commands

```bash
# Test rocpctl manually
rocpctl -d 1000 -p 0 GRBM_GUI_ACTIVE SQ_WAVES

# Check profiler cache
ls -la /tmp/rocprofiler-*

# Check for lock files
lsof | grep rocprofiler

# Monitor profiler execution
strace -f -e trace=openat,read,write rocpctl -d 1000 GRBM_GUI_ACTIVE
```

---

## Configuration Errors

### Scenario

Invalid config.json causes DME to fail startup or ignore metrics.

### Symptoms

- DME fails to start with validation error
- Expected metrics not exported
- Labels missing from output
- "Unknown field" errors in logs

### Root Causes

1. **Invalid JSON syntax** - missing commas, brackets
2. **Unknown field names** - typos in metric/label names
3. **Invalid enum values** - incorrect field references
4. **Missing required fields** - incomplete configuration

### Solutions

**Validate JSON syntax:**
```bash
jq . /etc/metrics/config.json
# Should output formatted JSON without errors
```

**Check field names:**
```bash
# Compare against example config
diff /etc/metrics/config.json /path/to/example/config.json

# Verify field names in proto
grep "GPU_" pkg/exporter/proto/exporterconfig.proto
```

**Check enum values:**
```bash
# View all available fields
cat pkg/exporter/proto/exporterconfig.proto | grep "GPU_"

# View all available labels
cat pkg/exporter/proto/exporterconfig.proto | grep "enum.*Label"
```

**Enable debug logging:**
```json
{
  "CommonConfig": {
    "Logging": {
      "Level": "debug"
    }
  }
}
```

### Common Mistakes

```json
// ❌ Wrong: Typo in field name
"Fields": ["GPU_PAKAGE_POWER"]
// ✅ Correct
"Fields": ["GPU_PACKAGE_POWER"]

// ❌ Wrong: Invalid label
"Labels": ["GPU_SERIAL_NUMBER"]
// ✅ Correct
"Labels": ["SERIAL_NUMBER"]

// ❌ Wrong: Invalid profiler config
"ProfilerMetrics": "false"
// ✅ Correct
"ProfilerMetrics": {"all": false}

// ❌ Wrong: String instead of object
"HealthThresholds": "default"
// ✅ Correct
"HealthThresholds": {
  "GPU_ECC_UNCORRECT_SDMA": 0,
  "GPU_ECC_UNCORRECT_GFX": 0
}
```

### Diagnostic Commands

```bash
# Validate JSON
jq . /etc/metrics/config.json

# Check for specific field
jq '.GPUConfig.Fields[]' /etc/metrics/config.json

# View full config
cat /etc/metrics/config.json

# Check DME startup logs
journalctl -u amd-metrics-exporter -n 100

# Test config reload
# Edit config, wait 3 seconds, check logs
tail -f /var/log/amd-metrics-exporter.log
```

---

## Kubernetes Scheduler Client Failures

### Scenario

Pod labels not appearing in metrics, workload correlation missing.

### Symptoms

- POD, NAMESPACE, CONTAINER labels missing or empty
- ExtraPodLabels not populated
- "Failed to connect to kubelet" errors in logs
- Workload information not correlated with devices

### Root Causes

1. **Kubelet PodResources socket not accessible**
2. **K8s API client not initialized** - missing RBAC permissions
3. **Device plugin resource name mismatch**
4. **Pod not scheduled yet** or already terminated

### Solutions

**Check kubelet socket:**
```bash
ls -la /var/lib/kubelet/pod-resources/kubelet.sock
# Should exist and be accessible
```

**Mount socket in container (DaemonSet):**
```yaml
volumes:
  - name: kubelet-pod-resources
    hostPath:
      path: /var/lib/kubelet/pod-resources
      type: Directory
volumeMounts:
  - name: kubelet-pod-resources
    mountPath: /var/lib/kubelet/pod-resources
    readOnly: true
```

**Verify RBAC permissions:**
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: amd-metrics-exporter
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
```

**Check permissions:**
```bash
kubectl auth can-i get pods --as=system:serviceaccount:kube-system:amd-metrics-exporter
# Should return "yes"
```

**Check device plugin prefix:**
```bash
# Should be "amd.com/*" or DRA driver "gpu.amd.com"
kubectl get pod <gpu-pod> -o json | jq '.spec.containers[].resources'
```

### Diagnostic Commands

```bash
# Check if socket mounted in container
kubectl exec -it <dme-pod> -- ls -la /var/lib/kubelet/pod-resources/

# Test kubelet connection
kubectl exec -it <dme-pod> -- grpcurl -plaintext -unix /var/lib/kubelet/pod-resources/kubelet.sock list

# Check RBAC
kubectl auth can-i get pods --as=system:serviceaccount:<namespace>:<serviceaccount>

# Check device allocations
kubectl get pod <gpu-pod> -o json | jq '.spec.containers[].resources.limits'

# View DME logs
kubectl logs <dme-pod> | grep -i scheduler
```

---

## SLURM Scheduler Client Failures

### Scenario

SLURM job labels not appearing in metrics.

### Symptoms

- JOB_ID, JOB_USER, JOB_PARTITION labels missing
- `/var/run/exporter/` directory empty or missing
- No job file correlation

### Root Causes

1. **SLURM prolog script not configured**
2. **Job environment files not created**
3. **Directory permissions incorrect**
4. **File watcher not detecting changes**

### Solutions

**Configure SLURM prolog:**
```bash
# /etc/slurm/prolog.sh
#!/bin/bash
mkdir -p /var/run/exporter
cat > /var/run/exporter/${SLURM_JOB_ID}.json <<EOF
{
  "SLURM_JOB_ID": "${SLURM_JOB_ID}",
  "SLURM_JOB_USER": "${SLURM_JOB_USER}",
  "SLURM_JOB_PARTITION": "${SLURM_JOB_PARTITION}",
  "CUDA_VISIBLE_DEVICES": "${CUDA_VISIBLE_DEVICES}",
  "SLURM_CLUSTER_NAME": "${SLURM_CLUSTER_NAME}"
}
EOF
chmod 644 /var/run/exporter/${SLURM_JOB_ID}.json
```

**Configure SLURM epilog:**
```bash
# /etc/slurm/epilog.sh
#!/bin/bash
rm -f /var/run/exporter/${SLURM_JOB_ID}.json
```

**Check directory:**
```bash
ls -la /var/run/exporter/
# Should show job JSON files

# Check permissions
stat /var/run/exporter/
# Should be writable by SLURM
```

**Test file watcher:**
```bash
# Create test file
echo '{"SLURM_JOB_ID":"12345"}' > /var/run/exporter/test.json

# Check DME logs
journalctl -u amd-metrics-exporter -f | grep -i slurm
```

### Diagnostic Commands

```bash
# Check SLURM prolog/epilog config
grep -i prolog /etc/slurm/slurm.conf

# List job files
ls -la /var/run/exporter/

# View job file content
cat /var/run/exporter/*.json

# Test manual job file creation
echo '{"SLURM_JOB_ID":"99999","CUDA_VISIBLE_DEVICES":"0"}' > /var/run/exporter/99999.json

# Monitor file watcher
inotifywait -m /var/run/exporter/
```

---

## NIC Exporter Failures

### Scenario

NIC metrics missing or incorrect.

### Symptoms

- NIC_PORT_STATS_* metrics missing
- RDMA stats timeout errors
- Ethtool stats empty
- "Command not found" errors

### Root Causes

1. **nicctl/ethtool/rdma binaries not installed**
2. **Insufficient permissions** to run CLI tools
3. **NIC devices not initialized**
4. **Pod network namespace issues**

### Solutions

**Check binaries:**
```bash
which nicctl
which ethtool
which rdma

# Install if missing
apt-get install ethtool iproute2
# nicctl is AMD-specific, install from AMD package
```

**Test CLI tools:**
```bash
# Test nicctl
nicctl show nic

# Test ethtool
ethtool -S eth0

# Test rdma
rdma statistic -j
```

**Check permissions:**
```bash
# DME needs NET_ADMIN capability
getcap /usr/bin/ethtool

# K8s: Add capability to container
securityContext:
  capabilities:
    add:
      - NET_ADMIN
```

**Increase RDMA timeout:**
```bash
# RDMA stats can take >5s on large systems
# Timeout is hardcoded to 20s in DME
# Check logs for timeout warnings
```

**Verify nsenter for pod metrics:**
```bash
# For pod-level metrics, ensure nsenter available
which nsenter

# Test namespace entry
nsenter --net=/proc/<pid>/ns/net ethtool -S <interface>
```

### Diagnostic Commands

```bash
# Test nicctl
nicctl show nic -j | jq .

# Test ethtool
ethtool -S eth0

# Test rdma with timeout
timeout 20s rdma statistic -j

# Check DME capabilities (K8s)
kubectl get pod <dme-pod> -o json | jq '.spec.containers[].securityContext.capabilities'

# Check NIC devices
ip link show

# Test pod network access
nsenter --net=/proc/<pid>/ns/net ip link show
```

---

## General Diagnostic Workflow

### 1. Check Service Status

```bash
# Systemd
systemctl status amd-metrics-exporter
systemctl status gpuagent

# K8s
kubectl get pods -n kube-system | grep amd-metrics
kubectl describe pod <dme-pod>
```

### 2. View Logs

```bash
# Systemd
journalctl -u amd-metrics-exporter -f
journalctl -u gpuagent -f

# K8s
kubectl logs <dme-pod> -f
kubectl logs <dme-pod> --previous  # Previous container
```

### 3. Test Metrics Endpoint

```bash
# Prometheus metrics
curl http://localhost:5000/metrics

# Full GPU metrics (JSON)
curl http://localhost:5000/gpumetrics | jq .

# In-band RAS errors (JSON)
curl http://localhost:5000/inbandraserrors | jq .

# Health service (gRPC)
grpcurl -plaintext -unix /var/lib/amd-metrics-exporter/amdgpu_device_metrics_exporter_grpc.socket \
  amdgpu.MetricsService/List
```

### 4. Check Configuration

```bash
# Validate config
jq . /etc/metrics/config.json

# Check config reload
# Edit config, wait 3 seconds, check logs
tail -f /var/log/amd-metrics-exporter.log
```

### 5. Verify Hardware

```bash
# Check GPU devices
ls -la /dev/dri/
ls -la /dev/kfd

# Check driver
lsmod | grep amdgpu
dmesg | grep amdgpu

# Check NIC devices
ip link show
lspci | grep -i network
```

### 6. Test Components

```bash
# Test gpuagent
gpuctl -s /var/run/gpuagent.sock get gpus

# Test profiler
rocpctl -d 1000 GRBM_GUI_ACTIVE

# Test NIC tools
nicctl show nic
ethtool -S eth0
rdma statistic -j
```

---

## Log Analysis

### Key Log Patterns

**Successful startup:**
```
INFO Starting AMD Metrics Exporter
INFO Deployment mode: kubernetes
INFO Initializing GPU monitoring
INFO Connected to gpuagent at /var/run/gpuagent.sock
INFO Health service started
INFO HTTP server listening on :5000
```

**Driver race condition:**
```
WARN Failed to connect to gpuagent: connection refused
WARN Zero GPUs reported, marking all as unhealthy
INFO Retry 1/3 for gpuagent connection
```

**Profiler failure:**
```
ERROR rocpctl execution failed: signal: aborted (core dumped)
WARN Profiler consecutive failures: 1/3
ERROR Profiler disabled after 3 consecutive failures
```

**Config reload:**
```
INFO Config file modified, reloading
INFO Stopping metrics server
INFO Reloading configuration
INFO Reinitializing metrics registry
INFO Starting metrics server on :5000
```

### Log Locations

**Systemd:**
- DME: `journalctl -u amd-metrics-exporter`
- gpuagent: `journalctl -u gpuagent`

**K8s:**
- DME: `kubectl logs <pod>`
- Previous: `kubectl logs <pod> --previous`

**File-based (Debian/RPM):**
- DME: `/var/log/amd-metrics-exporter.log`
- Rotation: Configured in LoggingConfig

---

## Related Documentation

- **Architecture:** [architecture.md](architecture.md)
- **GPU Exporter:** [gpu-exporter.md](gpu-exporter.md)
- **NIC Exporter:** [nic-exporter.md](nic-exporter.md)
- **Health Monitoring:** [health-monitoring.md](health-monitoring.md)
- **Configuration:** [configuration.md](configuration.md)

---

**Last Updated:** 2026-04-05
