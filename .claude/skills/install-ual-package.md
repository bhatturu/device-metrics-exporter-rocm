---
name: install-ual-package
description: Install or upgrade the UAL/IFOE exporter package (`amdgpuifoe-exporter*.deb` on Ubuntu, `amdgpuifoe-exporter*.x86_64.rpm` on RHEL9) on a remote host via scp + apt/dnf. Auto-detects target OS family. Enables and starts the systemd service, and verifies `/metrics` responds. Trigger when an agent needs the bare-metal install path (not container) before validate-ifoe-exporter, or when the user asks "install ual deb/rpm on <host>", "upgrade ifoe exporter on <host>".
---

# Install UAL/IFOE exporter package on remote host

Pre-built `.deb` (Ubuntu 22.04/24.04) or `.rpm` (RHEL9) are produced by
`build-ual-deb` and `build-ual-rpm` skills. This skill copies the
package to the target, installs it, enables the systemd unit, and
verifies the metrics endpoint responds.

**Inputs:**
- `TARGET` — `<user>@<host>` (must be sudo-capable).
- `PKG_PATH` — local path to the `.deb` or `.rpm` (extension drives
  the install path).
- `RESTART_GPUAGENT` — `true`/`false` (default `true`). Restart
  `gpuagent.service` after install — IFOE field exporter depends on
  the gRPC unix socket the agent owns.

---

## Step 0 — Detect OS family

```bash
OS_FAMILY=$(ssh ${TARGET} "
  if   [ -f /etc/os-release ]; then . /etc/os-release; echo \$ID_LIKE \$ID
  fi" | tr ' ' '\n' | grep -E '^(debian|ubuntu|rhel|fedora|centos)$' | head -1)
```

Map: `debian`/`ubuntu` → `apt`; `rhel`/`fedora`/`centos` → `dnf`.

Validate the `PKG_PATH` extension agrees with the detected family
(`.deb` ↔ apt, `.rpm` ↔ dnf). Mismatch is a FAIL — stop and report.

## Step 1 — scp the package

```bash
PKG_NAME=$(basename ${PKG_PATH})
scp ${PKG_PATH} ${TARGET}:/tmp/${PKG_NAME}
```

## Step 2 — Install

### Ubuntu (apt)

```bash
ssh ${TARGET} "sudo DEBIAN_FRONTEND=noninteractive apt install -y /tmp/${PKG_NAME}"
```

`apt install ./pkg.deb` resolves dependencies (`ethtool`, `iproute2`)
automatically. If `apt` is unavailable, fall back to
`sudo dpkg -i /tmp/${PKG_NAME} && sudo apt-get install -f -y`.

### RHEL9 (dnf)

```bash
ssh ${TARGET} "sudo dnf install -y /tmp/${PKG_NAME}"
```

## Step 3 — Service enable & start

```bash
ssh ${TARGET} "sudo systemctl daemon-reload && \
  sudo systemctl enable --now amd-metrics-exporter.service"
```

(`amd-metrics-exporter` is the unit shipped by both the Ubuntu and
RHEL UAL packages. The NIC-only Debian package uses
`amd-nic-metrics-exporter.service` — that's a different package and not
in scope here.)

## Step 4 — Restart gpuagent (if `RESTART_GPUAGENT=true`)

The IFOE field exporter consumes UAL state from `gpuagent` over a
Unix socket. After a fresh install the agent may have stale state; bounce it.

```bash
ssh ${TARGET} "sudo systemctl restart gpuagent.service && sleep 5"
```

Skip this if the agent is not present (host has the exporter package
installed without the agent — uncommon, but possible on cross-arch
build hosts).

## Step 5 — Verify

```bash
ssh ${TARGET} "
  systemctl is-active amd-metrics-exporter.service && \
  PORT=\$(jq -r .ServerPort /etc/metrics/config.json) && \
  curl -fsS http://localhost:\$PORT/metrics | head -5"
```

PASS if systemd reports `active`, `ServerPort` parses, and `/metrics`
returns ≥1 line. Fail with which step broke and the journal tail:

```bash
ssh ${TARGET} "sudo journalctl -u amd-metrics-exporter.service --since '2 min ago' | tail -50"
```

## Step 6 — Hand-off

```
INSTALL_OK host=<host> pkg=${PKG_NAME} version=$(rpm/dpkg -q result) port=<ServerPort>
```

The next step is typically `validate-ifoe-exporter` against the same `${TARGET}`.

## Pitfalls

- **OS detection on minimal images.** `/etc/os-release` is reliable;
  don't fall back to `lsb_release` (often missing).
- **Stale gpuagent state.** Step 4 is the difference between IFOE
  metrics being present-but-zero vs. present-and-correct.
- **`amd-metrics-exporter` vs `amd-nic-metrics-exporter`.** UAL package
  uses the former (port 5000); NIC-only uses the latter (port 5001 per
  `docs/installation/nic-debian-package.md`).
- **Don't `apt remove --purge`** between upgrade test runs — it deletes
  `/etc/metrics/config.json`, breaking validation.

## Cross-references

- Build the deb: `build-ual-deb` skill
- Build the rpm: `build-ual-rpm` skill
- Validate after install: `validate-ifoe-exporter` skill
- Container alternative: `deploy-exporter-container` skill
- NIC-only install reference (different package): `docs/installation/nic-debian-package.md`
