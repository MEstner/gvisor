#!/usr/bin/env bash
# riscv64_qemu_test.sh — Boot a Debian riscv64 VM and test runsc inside it.
#
# Usage:
#   ./tools/riscv64_qemu_test.sh [--build] [--keep] [--shell] [--platform PLATFORM]
#
# Options:
#   --build      Build runsc for riscv64 before testing (default: use existing binary)
#   --keep       Keep the VM running after tests (connect via: ssh -p 2222 root@127.0.0.1)
#   --shell      Drop into an SSH shell inside the VM after tests
#   --platform   gVisor platform to use: systrap (default) or kvm
#   --image-dir  Directory for QEMU image files (default: /tmp/gvisor-riscv64-qemu)
#   --ssh-port   Local port to forward for SSH (default: 2222)
#   --timeout    Timeout in seconds waiting for VM boot (default: 300)
#
# Requirements:
#   qemu-system-riscv64, opensbi, u-boot-qemu, curl, unzip, ssh, scp
#
set -euo pipefail

# ---------------------------------------------------------------------------
# Defaults
# ---------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
RUNSC_BIN="${REPO_ROOT}/bazel-bin/runsc/runsc_/runsc"
IMAGE_DIR="/tmp/gvisor-riscv64-qemu"
IMAGE_URL="https://gitlab.com/api/v4/projects/giomasce%2Fdqib/jobs/artifacts/master/download?job=convert_riscv64-virt"
SSH_PORT=2222
BOOT_TIMEOUT=300
PLATFORM="systrap"
DO_BUILD=false
KEEP_VM=false
DROP_SHELL=false
QEMU_PID=""

# ---------------------------------------------------------------------------
# Argument parsing
# ---------------------------------------------------------------------------
while [[ $# -gt 0 ]]; do
    case "$1" in
        --build)      DO_BUILD=true; shift ;;
        --keep)       KEEP_VM=true; shift ;;
        --shell)      DROP_SHELL=true; KEEP_VM=true; shift ;;
        --platform)   PLATFORM="$2"; shift 2 ;;
        --image-dir)  IMAGE_DIR="$2"; shift 2 ;;
        --ssh-port)   SSH_PORT="$2"; shift 2 ;;
        --timeout)    BOOT_TIMEOUT="$2"; shift 2 ;;
        -h|--help)
            sed -n '2,/^$/s/^# \?//p' "$0"
            exit 0
            ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
info()  { echo -e "\033[1;34m==>\033[0m $*"; }
ok()    { echo -e "\033[1;32m OK\033[0m $*"; }
fail()  { echo -e "\033[1;31mFAIL\033[0m $*"; }
die()   { fail "$@"; cleanup; exit 1; }

SSH_KEY="${IMAGE_DIR}/dqib_riscv64-virt/ssh_user_rsa_key"
SSH_OPTS=(-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
          -o LogLevel=ERROR -o ConnectTimeout=10 \
          -i "${SSH_KEY}" -p "${SSH_PORT}")

remote() { ssh "${SSH_OPTS[@]}" root@127.0.0.1 "$@"; }
remote_copy() { scp "${SSH_OPTS[@]}" "$1" "root@127.0.0.1:$2"; }

cleanup() {
    if [[ -n "${QEMU_PID}" ]] && kill -0 "${QEMU_PID}" 2>/dev/null; then
        if [[ "${KEEP_VM}" == true ]]; then
            info "VM kept running (PID ${QEMU_PID}). Connect with:"
            echo "  ssh ${SSH_OPTS[*]} root@127.0.0.1"
            echo "  Kill with: kill ${QEMU_PID}"
        else
            info "Shutting down VM (PID ${QEMU_PID})..."
            kill "${QEMU_PID}" 2>/dev/null || true
            wait "${QEMU_PID}" 2>/dev/null || true
        fi
    fi
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Step 1: Optionally build runsc
# ---------------------------------------------------------------------------
if [[ "${DO_BUILD}" == true ]]; then
    info "Building runsc for riscv64..."
    cd "${REPO_ROOT}"
    bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_riscv64 //runsc:runsc
    ok "Build complete."
fi

if [[ ! -f "${RUNSC_BIN}" ]]; then
    die "runsc binary not found at ${RUNSC_BIN}. Run with --build or build manually first."
fi

RUNSC_ARCH=$(file "${RUNSC_BIN}")
if ! echo "${RUNSC_ARCH}" | grep -q "RISC-V"; then
    die "Binary is not RISC-V: ${RUNSC_ARCH}"
fi
ok "runsc binary: ${RUNSC_ARCH}"

# ---------------------------------------------------------------------------
# Step 2: Check host dependencies
# ---------------------------------------------------------------------------
for cmd in qemu-system-riscv64 curl unzip ssh scp; do
    command -v "${cmd}" >/dev/null 2>&1 || die "Required command not found: ${cmd}"
done

OPENSBI="/usr/lib/riscv64-linux-gnu/opensbi/generic/fw_jump.elf"
UBOOT="/usr/lib/u-boot/qemu-riscv64_smode/uboot.elf"
[[ -f "${OPENSBI}" ]] || die "OpenSBI firmware not found at ${OPENSBI}. Install: apt install opensbi"
[[ -f "${UBOOT}" ]]   || die "U-Boot not found at ${UBOOT}. Install: apt install u-boot-qemu"

# ---------------------------------------------------------------------------
# Step 3: Download Debian riscv64 image if needed
# ---------------------------------------------------------------------------
QCOW2="${IMAGE_DIR}/dqib_riscv64-virt/image.qcow2"
if [[ ! -f "${QCOW2}" ]]; then
    info "Downloading Debian riscv64 QEMU image..."
    mkdir -p "${IMAGE_DIR}"
    curl -L -o "${IMAGE_DIR}/dqib_riscv64-virt.zip" "${IMAGE_URL}"
    unzip -o "${IMAGE_DIR}/dqib_riscv64-virt.zip" -d "${IMAGE_DIR}"
    rm -f "${IMAGE_DIR}/dqib_riscv64-virt.zip"
    ok "Image downloaded to ${IMAGE_DIR}/dqib_riscv64-virt/"
else
    ok "Using cached image at ${QCOW2}"
fi
chmod 600 "${SSH_KEY}"

# ---------------------------------------------------------------------------
# Step 4: Boot QEMU VM
# ---------------------------------------------------------------------------
if ss -tlnp 2>/dev/null | grep -q ":${SSH_PORT} "; then
    die "Port ${SSH_PORT} is already in use. Is another VM running?"
fi

info "Booting riscv64 VM (this takes 1-3 minutes under emulation)..."
qemu-system-riscv64 \
    -machine virt \
    -cpu rv64 \
    -m 1G \
    -device virtio-blk-device,drive=hd \
    -drive "file=${QCOW2},if=none,id=hd" \
    -device virtio-net-device,netdev=net \
    -netdev "user,id=net,hostfwd=tcp:127.0.0.1:${SSH_PORT}-:22" \
    -bios "${OPENSBI}" \
    -kernel "${UBOOT}" \
    -object rng-random,filename=/dev/urandom,id=rng \
    -device virtio-rng-device,rng=rng \
    -nographic \
    -append "root=LABEL=rootfs console=ttyS0" \
    </dev/null >/dev/null 2>&1 &
QEMU_PID=$!

# Wait for SSH to become available
info "Waiting for SSH (timeout: ${BOOT_TIMEOUT}s)..."
SECONDS=0
while (( SECONDS < BOOT_TIMEOUT )); do
    if remote 'true' 2>/dev/null; then
        break
    fi
    # Check QEMU is still alive
    if ! kill -0 "${QEMU_PID}" 2>/dev/null; then
        die "QEMU process died unexpectedly."
    fi
    sleep 5
done

if (( SECONDS >= BOOT_TIMEOUT )); then
    die "Timed out waiting for VM to boot after ${BOOT_TIMEOUT}s."
fi
ok "VM booted in ~${SECONDS}s."
remote 'uname -a'

# ---------------------------------------------------------------------------
# Step 5: Copy runsc into the VM
# ---------------------------------------------------------------------------
info "Copying runsc to VM..."
remote_copy "${RUNSC_BIN}" /usr/local/bin/runsc
remote 'chmod +x /usr/local/bin/runsc'
ok "runsc installed. Version: $(remote '/usr/local/bin/runsc --version' 2>/dev/null)"

# ---------------------------------------------------------------------------
# Step 6: Create a minimal OCI container bundle
# ---------------------------------------------------------------------------
info "Creating OCI test container..."
remote 'bash -s' << 'BUNDLE_EOF'
set -e
rm -rf /tmp/test-container
mkdir -p /tmp/test-container/rootfs/{lib,proc,sys,dev,tmp,etc}

# Copy a static-ish echo or fall back to busybox
if command -v busybox >/dev/null 2>&1; then
    cp "$(command -v busybox)" /tmp/test-container/rootfs/busybox
    ENTRYPOINT='["/busybox", "echo", "Hello from gVisor on RISC-V!"]'
else
    cp /bin/echo /tmp/test-container/rootfs/echo
    # Copy dynamic libraries
    for lib in $(ldd /bin/echo 2>/dev/null | grep -oP '/\S+'); do
        cp -n "$lib" /tmp/test-container/rootfs/lib/ 2>/dev/null || true
    done
    cp /lib/ld-linux-riscv64-lp64d.so.1 /tmp/test-container/rootfs/lib/ 2>/dev/null || true
    ENTRYPOINT='["/echo", "Hello from gVisor on RISC-V!"]'
fi

cat > /tmp/test-container/config.json << CFGEOF
{
  "ociVersion": "1.0.0",
  "process": {
    "terminal": false,
    "user": { "uid": 0, "gid": 0 },
    "args": ${ENTRYPOINT},
    "env": ["PATH=/"],
    "cwd": "/"
  },
  "root": { "path": "rootfs", "readonly": true },
  "linux": {
    "namespaces": [
      {"type": "pid"},
      {"type": "mount"},
      {"type": "ipc"},
      {"type": "uts"}
    ]
  },
  "mounts": [
    {"destination": "/proc", "type": "proc", "source": "proc"},
    {"destination": "/sys", "type": "sysfs", "source": "sysfs",
     "options": ["nosuid","noexec","nodev","ro"]},
    {"destination": "/dev", "type": "tmpfs", "source": "tmpfs"}
  ]
}
CFGEOF
echo "OCI bundle ready."
BUNDLE_EOF

# ---------------------------------------------------------------------------
# Step 7: Run gVisor
# ---------------------------------------------------------------------------
info "Running gVisor with --platform=${PLATFORM}..."
CONTAINER_ID="riscv64-test-$(date +%s)"
RUN_OUTPUT=$(remote "cd /tmp/test-container && timeout 120 \
    /usr/local/bin/runsc \
        --platform=${PLATFORM} \
        --network=none \
        --debug --debug-log=/tmp/runsc-debug.log \
        run ${CONTAINER_ID} 2>&1; echo EXIT_CODE=\$?" 2>&1) || true

EXIT_CODE=$(echo "${RUN_OUTPUT}" | grep -oP 'EXIT_CODE=\K\d+' || echo "unknown")

echo "--- runsc output ---"
echo "${RUN_OUTPUT}" | grep -v "^EXIT_CODE="
echo "--- end output ---"
echo ""

if echo "${RUN_OUTPUT}" | grep -q "Hello from gVisor on RISC-V!"; then
    ok "gVisor ran successfully on riscv64!"
elif [[ "${EXIT_CODE}" == "124" ]]; then
    fail "gVisor timed out (120s). Likely stuck in SwitchToUser/systrap."
    info "Fetching debug log tail..."
    remote 'tail -20 /tmp/runsc-debug.log' 2>/dev/null || true
    echo ""
    info "Fetching goroutine dump..."
    remote "kill -SIGABRT \$(pgrep -f 'runsc-sandbox.*${CONTAINER_ID}' | head -1) 2>/dev/null; \
            sleep 2; \
            grep -A5 'goroutine 1\b' /tmp/runsc-debug.log" 2>/dev/null || true
else
    fail "gVisor exited with code ${EXIT_CODE}."
    info "Fetching debug log tail..."
    remote 'tail -30 /tmp/runsc-debug.log' 2>/dev/null || true
fi

# ---------------------------------------------------------------------------
# Step 8: Optional interactive shell
# ---------------------------------------------------------------------------
if [[ "${DROP_SHELL}" == true ]]; then
    info "Dropping into VM shell (exit to quit)..."
    ssh "${SSH_OPTS[@]}" root@127.0.0.1
fi

exit "${EXIT_CODE:-1}"
