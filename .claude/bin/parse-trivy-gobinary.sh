#!/usr/bin/env bash
# parse-trivy-gobinary.sh — extract CVE findings from trivy gobinary sections in a CI scan log.
#
# Usage: parse-trivy-gobinary.sh <URL>
#
# Fetches the log at URL, parses gobinary sections for the four known binaries,
# and emits a JSON array of CVE findings to stdout.
# Exits 0 (with []) when no findings; exits 1 on usage error.

set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "Usage: $(basename "$0") <CI-scan-log-URL>" >&2
    exit 1
fi

URL="$1"

# Use -c so python reads the script from the argument string; stdin stays on the pipe.
curl -s --fail --compressed "$URL" | python3 -c "
import sys
import json
import re

BINARIES = {'amdgpuhealth', 'amd-metrics-exporter', 'metricsclient', 'amd-test-runner'}

def parse_trivy(text):
    findings = []
    current_binary = None
    in_table = False

    # State for blank-field inheritance within a section
    last_library = ''
    last_severity = ''
    last_installed = ''

    lines = text.splitlines()
    i = 0
    while i < len(lines):
        line = lines[i]

        # Detect any trivy section header: a line whose stripped form ends with (...type...)
        if re.search(r'\(\w[\w-]*\)\s*$', line.rstrip()):
            if line.rstrip().endswith('(gobinary)'):
                header = line.strip()
                # e.g. 'home/amd/bin/amdgpuhealth (gobinary)'
                path_part = header[:header.rfind('(gobinary)')].strip()
                basename = path_part.rstrip('/').split('/')[-1]

                if basename in BINARIES:
                    current_binary = basename
                else:
                    current_binary = None
            else:
                current_binary = None
            in_table = False
            last_library = ''
            last_severity = ''
            last_installed = ''
            i += 1
            continue

        # Only parse content under a matched binary
        if current_binary is None:
            i += 1
            continue

        # Detect table column-header row
        if '|' in line and 'Library' in line and 'Vulnerability' in line:
            in_table = True
            i += 1
            continue

        # Skip table separator lines (+---------+...)
        if re.match(r'^\s*\+[-+]+\+', line):
            i += 1
            continue

        # Parse table data rows
        if in_table and line.startswith('|'):
            parts = [p.strip() for p in line.split('|')]
            # parts[0]='' parts[1]=Library parts[2]=Vulnerability
            # parts[3]=Severity parts[4]=Status parts[5]=InstalledVersion parts[6]=FixedVersion
            if len(parts) < 7:
                i += 1
                continue

            library   = parts[1]
            cve       = parts[2]
            severity  = parts[3]
            # parts[4] = Status — not included in output
            installed = parts[5]
            fixed     = parts[6]

            # Skip column-header rows that slipped past the check above
            if library == 'Library' or cve == 'Vulnerability':
                i += 1
                continue

            # Blank-field inheritance: blank cell means same as last non-blank
            if library:
                last_library = library
            else:
                library = last_library

            if severity:
                last_severity = severity
            else:
                severity = last_severity

            if installed:
                last_installed = installed
            else:
                installed = last_installed

            # Emit only rows with a CVE or GHSA identifier
            if cve and (cve.startswith('CVE-') or cve.startswith('GHSA-')):
                findings.append({
                    'binary':    current_binary,
                    'library':   library,
                    'cve':       cve,
                    'severity':  severity,
                    'installed': installed,
                    'fixed':     fixed,
                })

            i += 1
            continue

        i += 1

    return findings

data = sys.stdin.read()
findings = parse_trivy(data)
print(json.dumps(findings, indent=2))
"
