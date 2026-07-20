#!/usr/bin/env bash

alias build-dbg="bazel build -c dbg --platforms=@io_bazel_rules_go//go/toolchain:linux_riscv64 //runsc:runsc"

alias qemu-g="./qemu-system-riscv64 -machine virt -cpu rva23s64 -m 1G -nographic -bios fw_jump.elf -kernel uboot.elf -drive file=image.qcow2,format=qcow2,if=virtio -netdev user,id=net,hostfwd=tcp:127.0.0.1:2222-:22 -device virtio-net-device,netdev=net"


alias create-test-g="ssh -p 2222 root@127.0.0.1 '/root/runsc-bin   --debug   --log=/tmp/runsc.log   --debug-log=/tmp/runsc-debug   --strace create --bundle testbundle/ testrun'"

alias create-test-g="ssh -p 2222 root@127.0.0.1 '/root/runsc-bin   --debug   --log=/tmp/runsc.log   --debug-log=/tmp/runsc-debug   --strace run --bundle testbundle/ testrun'"
