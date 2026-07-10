# AGENTS.md - Context for AI Coding Assistants

## Persona & Expertise

You are an expert Systems Engineer specializing in Linux Kernel internals, the
Linux ABI, and systems programming in Go. You understand how system calls work,
the nuances of memory management, and the security implications of sandbox
escape vulnerabilities.

## Project Overview

gVisor is a user-space kernel, written in Go, that implements a substantial
portion of the Linux system surface. It provides an isolation boundary between
applications and the host kernel.

-   **Sentry:** The heart of gVisor; it acts as the "kernel" running the
    application.
-   **Gofer:** Handles file system operations to provide further isolation.
-   **runsc:** The OCI-compatible runtime executable.

## Tech Stack & Tooling

-   **Language:** Go (Golang).
-   **Build System:** Bazel (primary). Use `make` as a wrapper for common tasks.
-   **Platform:** Linux (x86_64, ARM64).

## Critical Development Commands

AI agents should use these commands to build, test, and verify:

-   **Build all targets:** `make build`
-   **Run unit tests:** `make tests`
-   **Run a specific test:** `make test TARGETS="//runsc:version_test"`

## Repository Structure

-   `/pkg/sentry`: The core "kernel" logic (process management, memory,
    syscalls).
-   `/pkg/abi`: Definitions of Linux constants and structures.
-   `/pkg/sentry/syscalls`: Implementation of individual Linux syscall handlers.
-   `/runsc`: Entry point for the OCI runtime.
-   `/tools`: Development and build utilities.

## Git & PR Guidelines

-   **Breaking Changes:** Any change to the ABI implementation must be verified
    against the equivalent Linux kernel behavior.

## Testing gVisor on QEMU RISC-V 64-bit

A helper script at `tools/riscv64_qemu_test.sh` automates the full test cycle:
boot a Debian riscv64 VM under QEMU, copy `runsc` into it, and execute a
minimal OCI container to verify that gVisor works end-to-end on RISC-V.

### Host prerequisites

```bash
apt install qemu-system-misc opensbi u-boot-qemu  # QEMU + firmware
# also needed on PATH: curl, unzip, ssh, scp
```

### Cross-compile `runsc` for riscv64

```bash
bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_riscv64 //runsc:runsc
# output: bazel-bin/runsc/runsc_/runsc  (ELF 64-bit, RISC-V)
```

### Run the full test

```bash
# Build + boot VM + run container + shut down:
./tools/riscv64_qemu_test.sh --build

# Skip rebuild if runsc is already built:
./tools/riscv64_qemu_test.sh

# Use the kvm platform instead of the default systrap:
./tools/riscv64_qemu_test.sh --platform kvm

# Keep VM alive after tests (useful for manual inspection):
./tools/riscv64_qemu_test.sh --keep

# Drop into an interactive SSH shell inside the VM after tests:
./tools/riscv64_qemu_test.sh --shell
```

### What the script does (step by step)

| Step | Action |
|------|--------|
| 1 | (Optional) Builds `runsc` for riscv64 via Bazel |
| 2 | Verifies host tools (`qemu-system-riscv64`, OpenSBI, U-Boot) are present |
| 3 | Downloads the Debian riscv64 QCOW2 image from the **dqib** GitLab project (cached in `/tmp/gvisor-riscv64-qemu` on subsequent runs) |
| 4 | Boots the VM: `qemu-system-riscv64 -machine virt -cpu rv64 -m 1G` with virtio-blk, virtio-net (SSH forwarded to `localhost:2222`), OpenSBI + U-Boot |
| 5 | Waits up to 300 s for SSH to become reachable, then copies `runsc` to `/usr/local/bin/runsc` inside the VM |
| 6 | Creates a minimal OCI bundle (`/tmp/test-container`) using busybox or `/bin/echo` |
| 7 | Runs `runsc --platform=<systrap\|kvm> run <id>` and checks for `Hello from gVisor on RISC-V!` in stdout |
| 8 | (Optional) Opens an interactive SSH shell |

### Useful options

| Flag | Default | Description |
|------|---------|-------------|
| `--build` | off | Cross-compile before testing |
| `--platform` | `systrap` | gVisor platform (`systrap` or `kvm`) |
| `--image-dir` | `/tmp/gvisor-riscv64-qemu` | Where QEMU images are cached |
| `--ssh-port` | `2222` | Host port forwarded to VM SSH |
| `--timeout` | `300` | Seconds to wait for VM boot |
| `--keep` | off | Leave VM running after tests |
| `--shell` | off | Open interactive SSH shell (implies `--keep`) |

### Connecting to a kept-alive VM

```bash
ssh -i /tmp/gvisor-riscv64-qemu/dqib_riscv64-virt/ssh_user_rsa_key \
    -o StrictHostKeyChecking=no -p 2222 root@127.0.0.1
```

### Debug logs

`runsc` writes its debug output to `/tmp/runsc-debug.log` inside the VM. If the
container hangs or fails, retrieve it with:

```bash
ssh ... 'tail -50 /tmp/runsc-debug.log'
```
