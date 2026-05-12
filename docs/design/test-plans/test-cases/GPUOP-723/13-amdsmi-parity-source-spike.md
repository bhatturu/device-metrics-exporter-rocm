---
status: dropped
reason: "gpuagent does not expose its built-against amd-smi version anywhere consultable (no manifest, no --version, no SONAME mapping). The only viable parity check is runtime LD_PRELOAD symbol resolution, which is already covered by TC09 (libamd-smi-26.3.0-linkage). No spike needed."
---

# (Dropped) Spike: amd-smi parity source identification

This investigation was dropped after confirming with the gpuagent team
that there is no built-in amd-smi version exposure: no manifest in the
tarball, no `--version` output, no useful ELF SONAME mapping. The only
practical compatibility check available today is runtime symbol
resolution via LD_PRELOAD — which is exactly what
[TC09](09-libamd-smi-26.3.0-linkage.md) validates.

Sub-task S7 in the Story design doc is updated accordingly: parity
verification = "TC09 passes" (runtime LD_PRELOAD validates symbol
compatibility). No standalone parity script is needed in
`update_ual_assets.sh`.

If a future gpuagent build exposes its amd-smi build version through
a manifest or `--version` flag, this spike can be reconsidered.
