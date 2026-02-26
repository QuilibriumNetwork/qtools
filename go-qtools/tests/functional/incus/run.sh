#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_QTOOLS_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

SKIP_BUILD="${SKIP_BUILD:-0}"
KEEP_CONTAINERS="${KEEP_CONTAINERS:-0}"
INCLUDE_LIGHTWEIGHT="${INCLUDE_LIGHTWEIGHT:-1}"
SCENARIO="${SCENARIO:-full}"
BINARY_PATH="${BINARY_PATH:-${GO_QTOOLS_DIR}/dist/qtools-functional}"
declare -a OS_FILTERS=()

usage() {
  cat <<EOF
Usage: $(basename "$0") [--os <name>]... [--scenario <full|quick|all>]

Runs Incus functional tests across distro matrix.

Options:
  --os <name>   Filter matrix by OS label/family (repeatable).
                Examples: ubuntu, ubuntu-24.04, debian, alpine
  --scenario    Test scenario to run:
                  full  -> node update + CLI checks (default)
                  quick -> node update + CLI checks on one distro (faster)
                  all   -> both full and quick
  -h, --help    Show this help.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --os)
      if [[ $# -lt 2 || -z "${2}" ]]; then
        echo "ERROR: --os requires a value."
        usage
        exit 2
      fi
      OS_FILTERS+=("$(printf "%s" "$2" | tr '[:upper:]' '[:lower:]')")
      shift 2
      ;;
    --scenario)
      if [[ $# -lt 2 || -z "${2}" ]]; then
        echo "ERROR: --scenario requires a value."
        usage
        exit 2
      fi
      SCENARIO="$(printf "%s" "$2" | tr '[:upper:]' '[:lower:]')"
      if [[ "${SCENARIO}" != "full" && "${SCENARIO}" != "quick" && "${SCENARIO}" != "all" ]]; then
        echo "ERROR: invalid --scenario value: ${SCENARIO}"
        usage
        exit 2
      fi
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "ERROR: unknown argument: $1"
      usage
      exit 2
      ;;
  esac
done

if ! command -v incus >/dev/null 2>&1; then
  echo "ERROR: incus is required but not installed."
  exit 1
fi

if ! incus info >/dev/null 2>&1; then
  echo "ERROR: incus is not initialized or is not reachable."
  echo "Run 'incus admin init' first."
  exit 1
fi

if [[ "${SKIP_BUILD}" != "1" ]]; then
  echo "Building fresh qtools binary at ${BINARY_PATH}"
  mkdir -p "$(dirname "${BINARY_PATH}")"
  (
    cd "${GO_QTOOLS_DIR}"
    # Build a portable static Linux binary so Alpine (musl) can execute it.
    CGO_ENABLED=0 GOOS=linux GOARCH="$(go env GOARCH)" go build -o "${BINARY_PATH}" ./cmd/qtools
  )
else
  echo "Skipping build (SKIP_BUILD=1)."
fi

if [[ ! -f "${BINARY_PATH}" ]]; then
  echo "ERROR: qtools binary not found at ${BINARY_PATH}"
  exit 1
fi

TEST_ID="$(date +%Y%m%d%H%M%S)"
declare -a CREATED_CONTAINERS=()

cleanup() {
  if [[ "${KEEP_CONTAINERS}" == "1" ]]; then
    echo "KEEP_CONTAINERS=1 set; leaving test containers running."
    return
  fi

  for c in "${CREATED_CONTAINERS[@]:-}"; do
    incus delete --force "${c}" >/dev/null 2>&1 || true
  done
}
trap cleanup EXIT

# Strict matrix: expected to pass on supported distro families.
STRICT_MATRIX=(
  "ubuntu-22.04|images:ubuntu/22.04"
  "ubuntu-24.04|images:ubuntu/24.04"
  "debian-12|images:debian/12"
)

# Lightweight matrix: best-effort only (can be unsupported by install logic).
LIGHTWEIGHT_MATRIX=(
  "alpine-3.20|images:alpine/3.20"
)

# Quick scenario matrix: fast CLI-surface validation without install flow.
QUICK_MATRIX=(
  "ubuntu-24.04|images:ubuntu/24.04"
)

matches_os_filter() {
  local label="$1"
  local normalized_label
  normalized_label="$(printf "%s" "${label}" | tr '[:upper:]' '[:lower:]')"

  if [[ "${#OS_FILTERS[@]}" -eq 0 ]]; then
    return 0
  fi

  local filter
  for filter in "${OS_FILTERS[@]}"; do
    case "${filter}" in
      all)
        return 0
        ;;
      ubuntu|debian|alpine)
        if [[ "${normalized_label}" == "${filter}"-* ]]; then
          return 0
        fi
        ;;
      *)
        if [[ "${normalized_label}" == "${filter}" || "${normalized_label}" == "${filter}"-* ]]; then
          return 0
        fi
        ;;
    esac
  done

  return 1
}

run_case() {
  local label="$1"
  local image="$2"
  local strict="$3"
  local container="qtools-ft-${label//[^a-zA-Z0-9]/-}-${TEST_ID}"

  echo ""
  echo "=== [${label}] Launching ${image} ==="
  incus launch "${image}" "${container}" >/dev/null
  CREATED_CONTAINERS+=("${container}")

  # Wait until container is up.
  local tries=30
  until incus exec "${container}" -- sh -c "true" >/dev/null 2>&1; do
    tries=$((tries - 1))
    if [[ "${tries}" -le 0 ]]; then
      echo "FAIL [${label}] container did not become ready."
      return 1
    fi
    sleep 1
  done

  # Wait for network to be ready.
  local route_tries=30
  until incus exec "${container}" -- sh -ec 'ip route 2>/dev/null | grep -q "^default "' >/dev/null 2>&1; do
    route_tries=$((route_tries - 1))
    if [[ "${route_tries}" -le 0 ]]; then
      break
    fi
    sleep 1
  done
  if ! incus exec "${container}" -- sh -ec 'ip route 2>/dev/null | grep -q "^default "' >/dev/null 2>&1; then
    echo "FAIL [${label}] container network not ready (no default route)."
    incus exec "${container}" -- sh -ec 'ip -4 addr show eth0 || true; ip route || true' || true
    return 1
  fi

  # Ensure prerequisite tools used by update/config workflows.
  incus exec "${container}" -- sh -ec '
set -eu
if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update >/dev/null 2>&1
  apt-get install -y apt-utils ca-certificates curl sudo >/dev/null 2>&1
elif command -v apk >/dev/null 2>&1; then
  apk add --no-cache ca-certificates curl sudo shadow >/dev/null
fi

if ! command -v sudo >/dev/null 2>&1; then
  cat > /usr/local/bin/sudo <<'EOF'
#!/bin/sh
exec "$@"
EOF
  chmod +x /usr/local/bin/sudo
fi
'

  incus file push "${BINARY_PATH}" "${container}/tmp/qtools"
  incus exec "${container}" -- chmod +x /tmp/qtools

  set +e
  local output
  output="$(
    incus exec "${container}" -- sh -ec '
set -eu
export PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

/tmp/qtools node update --force

if [ ! -d /home/quilibrium/node ]; then
  echo "ASSERT FAIL: /home/quilibrium/node is missing"
  exit 1
fi

NODE_BIN="$(ls -1 /home/quilibrium/node/node-* 2>/dev/null | head -n1)"
if [ -z "${NODE_BIN}" ]; then
  echo "ASSERT FAIL: no node-* binary found in /home/quilibrium/node"
  ls -la /home/quilibrium/node || true
  exit 1
fi
if [ ! -x "${NODE_BIN}" ]; then
  echo "ASSERT FAIL: node binary is not executable: ${NODE_BIN}"
  ls -la "${NODE_BIN}" || true
  exit 1
fi

if [ ! -L /usr/local/bin/quilibrium-node ]; then
  echo "ASSERT FAIL: /usr/local/bin/quilibrium-node is not a symlink"
  ls -la /usr/local/bin/quilibrium-node || true
  exit 1
fi
if [ ! -x /usr/local/bin/quilibrium-node ]; then
  echo "ASSERT FAIL: /usr/local/bin/quilibrium-node is not executable"
  ls -la /usr/local/bin/quilibrium-node || true
  exit 1
fi
if ! command -v quilibrium-node >/dev/null 2>&1; then
  echo "ASSERT FAIL: quilibrium-node not discoverable in PATH"
  echo "PATH=${PATH}"
  exit 1
fi

# Ensure qtools config exists for config command testing.
mkdir -p /home/quilibrium/qtools
if [ ! -f /home/quilibrium/qtools/config.yml ]; then
  cat >/home/quilibrium/qtools/config.yml <<'EOF'
config_version: "1.4"
scheduled_tasks:
  updates:
    node:
      enabled: false
EOF
fi

# Validate additional CLI surfaces that should work after update.
/tmp/qtools --help >/dev/null
/tmp/qtools --version >/dev/null

# Completion generation should work without installation side effects.
/tmp/qtools completion bash --generate >/tmp/qtools-completion.bash
test -s /tmp/qtools-completion.bash

# Non-destructive placeholder commands should execute successfully.
SERVICE_STATUS_OUT="$(/tmp/qtools service status 2>&1)" || {
  echo "ASSERT FAIL: qtools service status failed"
  echo "${SERVICE_STATUS_OUT}"
  exit 1
}
printf "%s" "${SERVICE_STATUS_OUT}" | grep -q "Service status:" || {
  echo "ASSERT FAIL: unexpected service status output"
  echo "${SERVICE_STATUS_OUT}"
  exit 1
}

DIAGNOSTICS_RUN_OUT="$(/tmp/qtools diagnostics run 2>&1)" || {
  echo "ASSERT FAIL: qtools diagnostics run failed"
  echo "${DIAGNOSTICS_RUN_OUT}"
  exit 1
}
printf "%s" "${DIAGNOSTICS_RUN_OUT}" | grep -q "Running all diagnostics..." || {
  echo "ASSERT FAIL: unexpected diagnostics run output"
  echo "${DIAGNOSTICS_RUN_OUT}"
  exit 1
}

UPDATE_CHECK_OUT="$(/tmp/qtools update self --check 2>&1)" || {
  echo "ASSERT FAIL: qtools update self --check failed"
  echo "${UPDATE_CHECK_OUT}"
  exit 1
}
printf "%s" "${UPDATE_CHECK_OUT}" | grep -q "Checking for qtools updates..." || {
  echo "ASSERT FAIL: unexpected update self --check output"
  echo "${UPDATE_CHECK_OUT}"
  exit 1
}

# qtools config set/get should persist and return expected values.
if ! /tmp/qtools config set scheduled_tasks.updates.node.enabled true --quiet; then
  echo "ASSERT FAIL: qtools config set failed"
  exit 1
fi
CONFIG_GET_OUT="$(/tmp/qtools config get scheduled_tasks.updates.node.enabled)"
printf "%s" "${CONFIG_GET_OUT}" | grep -q "true" || {
  echo "ASSERT FAIL: qtools config get did not return true"
  echo "VALUE=${CONFIG_GET_OUT}"
  exit 1
}

# Node config create + set/get should work with default config.
/tmp/qtools node config create --force >/dev/null || {
  echo "ASSERT FAIL: node config create failed"
  exit 1
}
/tmp/qtools node config set --config quil --quiet p2p.listenMultiaddr /ip4/0.0.0.0/udp/9336/quic-v1 || {
  echo "ASSERT FAIL: node config set failed"
  exit 1
}
NODE_CONFIG_SET_GET_OUT="$(/tmp/qtools node config get --config quil p2p.listenMultiaddr)"
printf "%s" "${NODE_CONFIG_SET_GET_OUT}" | grep -q "/ip4/0.0.0.0/udp/9336/quic-v1" || {
  echo "ASSERT FAIL: node config get returned unexpected value"
  echo "VALUE=${NODE_CONFIG_SET_GET_OUT}"
  exit 1
}

# Node config get should return provided default when key is missing.
NODE_CONFIG_DEFAULT_OUT="$(/tmp/qtools node config get --config quil --default sentinel p2p.listenMultiaddr.child 2>&1)" || {
  echo "ASSERT FAIL: node config get with --default failed"
  echo "${NODE_CONFIG_DEFAULT_OUT}"
  exit 1
}

# Validation error path: conflicting mode flags must fail.
if /tmp/qtools node mode --manual --automatic >/tmp/qtools-node-mode.err 2>&1; then
  echo "expected node mode validation to fail" >&2
  exit 1
fi
if ! grep -q "cannot specify both --manual and --automatic" /tmp/qtools-node-mode.err; then
  echo "ASSERT FAIL: node mode error message mismatch"
  cat /tmp/qtools-node-mode.err || true
  exit 1
fi

# Validation error path: malformed worker list must fail before S3 operations.
if /tmp/qtools node backup --worker nope >/tmp/qtools-node-backup.err 2>&1; then
  echo "expected node backup validation to fail" >&2
  exit 1
fi
if ! grep -q "invalid --worker value" /tmp/qtools-node-backup.err; then
  echo "ASSERT FAIL: node backup error message mismatch"
  cat /tmp/qtools-node-backup.err || true
  exit 1
fi

# Ensure qclient symlink dispatch path works (binary invoked as qclient).
ln -sf /tmp/qtools /tmp/qclient
QCLIENT_HELP_OUT="$(/tmp/qclient --help 2>&1)" || {
  echo "ASSERT FAIL: qclient symlink help invocation failed"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}
printf "%s" "${QCLIENT_HELP_OUT}" | grep -q "^Usage:" || {
  echo "ASSERT FAIL: qclient help missing Usage header"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}
printf "%s" "${QCLIENT_HELP_OUT}" | grep -q "qtools qclient \\[command\\]" || {
  echo "ASSERT FAIL: qclient help usage command mismatch"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}
printf "%s" "${QCLIENT_HELP_OUT}" | grep -q "Available Commands:" || {
  echo "ASSERT FAIL: qclient help command list missing"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}

echo "ASSERTIONS_OK"
' 2>&1
  )"
  local rc=$?
  set -e

  if [[ "${rc}" -eq 0 ]]; then
    echo "PASS [${label}]"
    return 0
  fi

  if [[ "${strict}" == "0" ]]; then
    echo "WARN [${label}] lightweight/best-effort target failed:"
    echo "${output}"
    return 0
  fi

  echo "FAIL [${label}]"
  echo "${output}"
  return 1
}

run_quick_case() {
  local label="$1"
  local image="$2"
  local strict="$3"
  local container="qtools-ft-quick-${label//[^a-zA-Z0-9]/-}-${TEST_ID}"

  echo ""
  echo "=== [quick:${label}] Launching ${image} ==="
  incus launch "${image}" "${container}" >/dev/null
  CREATED_CONTAINERS+=("${container}")

  local tries=30
  until incus exec "${container}" -- sh -c "true" >/dev/null 2>&1; do
    tries=$((tries - 1))
    if [[ "${tries}" -le 0 ]]; then
      echo "FAIL [quick:${label}] container did not become ready."
      return 1
    fi
    sleep 1
  done

  local route_tries=30
  until incus exec "${container}" -- sh -ec 'ip route 2>/dev/null | grep -q "^default "' >/dev/null 2>&1; do
    route_tries=$((route_tries - 1))
    if [[ "${route_tries}" -le 0 ]]; then
      break
    fi
    sleep 1
  done
  if ! incus exec "${container}" -- sh -ec 'ip route 2>/dev/null | grep -q "^default "' >/dev/null 2>&1; then
    echo "FAIL [quick:${label}] container network not ready (no default route)."
    incus exec "${container}" -- sh -ec 'ip -4 addr show eth0 || true; ip route || true' || true
    return 1
  fi

  incus exec "${container}" -- sh -ec '
set -eu
if command -v apt-get >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  apt-get update >/dev/null 2>&1
  apt-get install -y apt-utils ca-certificates curl sudo >/dev/null 2>&1
elif command -v apk >/dev/null 2>&1; then
  apk add --no-cache ca-certificates curl sudo shadow >/dev/null
fi
'

  incus file push "${BINARY_PATH}" "${container}/tmp/qtools"
  incus exec "${container}" -- chmod +x /tmp/qtools

  set +e
  local output
  output="$(
    incus exec "${container}" -- sh -ec '
set -eu
export PATH="/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"

# Download/update node binary and ensure symlink for node config creation checks.
/tmp/qtools node update --force

# Create a deterministic default qtools config for config command testing.
mkdir -p /home/quilibrium/qtools
cat >/home/quilibrium/qtools/config.yml <<'"'"'EOF'"'"'
config_version: "1.4"
scheduled_tasks:
  updates:
    node:
      enabled: false
EOF

/tmp/qtools --help >/dev/null
/tmp/qtools --version >/dev/null

/tmp/qtools completion bash --generate >/tmp/qtools-completion.bash
test -s /tmp/qtools-completion.bash

SERVICE_STATUS_OUT="$(/tmp/qtools service status 2>&1)" || {
  echo "ASSERT FAIL: qtools service status failed"
  echo "${SERVICE_STATUS_OUT}"
  exit 1
}
printf "%s" "${SERVICE_STATUS_OUT}" | grep -q "Service status:" || {
  echo "ASSERT FAIL: unexpected service status output"
  echo "${SERVICE_STATUS_OUT}"
  exit 1
}

DIAGNOSTICS_RUN_OUT="$(/tmp/qtools diagnostics run 2>&1)" || {
  echo "ASSERT FAIL: qtools diagnostics run failed"
  echo "${DIAGNOSTICS_RUN_OUT}"
  exit 1
}
printf "%s" "${DIAGNOSTICS_RUN_OUT}" | grep -q "Running all diagnostics..." || {
  echo "ASSERT FAIL: unexpected diagnostics run output"
  echo "${DIAGNOSTICS_RUN_OUT}"
  exit 1
}

UPDATE_CHECK_OUT="$(/tmp/qtools update self --check 2>&1)" || {
  echo "ASSERT FAIL: qtools update self --check failed"
  echo "${UPDATE_CHECK_OUT}"
  exit 1
}
printf "%s" "${UPDATE_CHECK_OUT}" | grep -q "Checking for qtools updates..." || {
  echo "ASSERT FAIL: unexpected update self --check output"
  echo "${UPDATE_CHECK_OUT}"
  exit 1
}

if ! /tmp/qtools config set scheduled_tasks.updates.node.enabled true --quiet; then
  echo "ASSERT FAIL: qtools config set failed"
  exit 1
fi
CONFIG_GET_OUT="$(/tmp/qtools config get scheduled_tasks.updates.node.enabled)"
printf "%s" "${CONFIG_GET_OUT}" | grep -q "true" || {
  echo "ASSERT FAIL: qtools config get did not return true"
  echo "VALUE=${CONFIG_GET_OUT}"
  exit 1
}

/tmp/qtools node config create --force >/dev/null || {
  echo "ASSERT FAIL: node config create failed"
  exit 1
}
/tmp/qtools node config set --config quil --quiet p2p.listenMultiaddr /ip4/0.0.0.0/udp/9336/quic-v1 || {
  echo "ASSERT FAIL: node config set failed"
  exit 1
}
NODE_CONFIG_SET_GET_OUT="$(/tmp/qtools node config get --config quil p2p.listenMultiaddr)"
printf "%s" "${NODE_CONFIG_SET_GET_OUT}" | grep -q "/ip4/0.0.0.0/udp/9336/quic-v1" || {
  echo "ASSERT FAIL: node config get returned unexpected value"
  echo "VALUE=${NODE_CONFIG_SET_GET_OUT}"
  exit 1
}

NODE_CONFIG_DEFAULT_OUT="$(/tmp/qtools node config get --config quil --default sentinel p2p.listenMultiaddr.child 2>&1)" || {
  echo "ASSERT FAIL: node config get with --default failed"
  echo "${NODE_CONFIG_DEFAULT_OUT}"
  exit 1
}

if /tmp/qtools node mode --manual --automatic >/tmp/qtools-node-mode.err 2>&1; then
  echo "expected node mode validation to fail" >&2
  exit 1
fi
if ! grep -q "cannot specify both --manual and --automatic" /tmp/qtools-node-mode.err; then
  echo "ASSERT FAIL: node mode error message mismatch"
  cat /tmp/qtools-node-mode.err || true
  exit 1
fi

if /tmp/qtools node backup --worker nope >/tmp/qtools-node-backup.err 2>&1; then
  echo "expected node backup validation to fail" >&2
  exit 1
fi
if ! grep -q "invalid --worker value" /tmp/qtools-node-backup.err; then
  echo "ASSERT FAIL: node backup error message mismatch"
  cat /tmp/qtools-node-backup.err || true
  exit 1
fi

ln -sf /tmp/qtools /tmp/qclient
QCLIENT_HELP_OUT="$(/tmp/qclient --help 2>&1)" || {
  echo "ASSERT FAIL: qclient symlink help invocation failed"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}
printf "%s" "${QCLIENT_HELP_OUT}" | grep -q "^Usage:" || {
  echo "ASSERT FAIL: qclient help missing Usage header"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}
printf "%s" "${QCLIENT_HELP_OUT}" | grep -q "qtools qclient \\[command\\]" || {
  echo "ASSERT FAIL: qclient help usage command mismatch"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}
printf "%s" "${QCLIENT_HELP_OUT}" | grep -q "Available Commands:" || {
  echo "ASSERT FAIL: qclient help command list missing"
  echo "${QCLIENT_HELP_OUT}"
  exit 1
}

echo "ASSERTIONS_OK"
' 2>&1
  )"
  local rc=$?
  set -e

  if [[ "${rc}" -eq 0 ]]; then
    echo "PASS [quick:${label}]"
    return 0
  fi

  if [[ "${strict}" == "0" ]]; then
    echo "WARN [quick:${label}] lightweight/best-effort target failed:"
    echo "${output}"
    return 0
  fi

  echo "FAIL [quick:${label}]"
  echo "${output}"
  return 1
}

failures=0
passes=0
scheduled=0

if [[ "${SCENARIO}" == "full" || "${SCENARIO}" == "all" ]]; then
  for entry in "${STRICT_MATRIX[@]}"; do
    label="${entry%%|*}"
    image="${entry#*|}"
    if ! matches_os_filter "${label}"; then
      continue
    fi
    scheduled=$((scheduled + 1))
    if run_case "${label}" "${image}" "1"; then
      passes=$((passes + 1))
    else
      failures=$((failures + 1))
    fi
  done

  if [[ "${INCLUDE_LIGHTWEIGHT}" == "1" || "${#OS_FILTERS[@]}" -gt 0 ]]; then
    for entry in "${LIGHTWEIGHT_MATRIX[@]}"; do
      label="${entry%%|*}"
      image="${entry#*|}"
      if ! matches_os_filter "${label}"; then
        continue
      fi
      scheduled=$((scheduled + 1))
      if run_case "${label}" "${image}" "0"; then
        passes=$((passes + 1))
      else
        failures=$((failures + 1))
      fi
    done
  fi
fi

if [[ "${SCENARIO}" == "quick" || "${SCENARIO}" == "all" ]]; then
  for entry in "${QUICK_MATRIX[@]}"; do
    label="${entry%%|*}"
    image="${entry#*|}"
    if ! matches_os_filter "${label}"; then
      continue
    fi
    scheduled=$((scheduled + 1))
    if run_quick_case "${label}" "${image}" "1"; then
      passes=$((passes + 1))
    else
      failures=$((failures + 1))
    fi
  done
fi

if [[ "${scheduled}" -eq 0 ]]; then
  echo "ERROR: no matrix entries matched requested --os filters."
  echo "Available targets by scenario:"
  echo "  full : ubuntu-22.04, ubuntu-24.04, debian-12, alpine-3.20"
  echo "  quick: ubuntu-24.04"
  exit 2
fi

echo ""
echo "Functional matrix complete: ${passes}/${scheduled} passed, ${failures} failed."

if [[ "${failures}" -gt 0 ]]; then
  exit 1
fi
