// Copyright 2019 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build riscv64
// +build riscv64

package kvm

import (
	"fmt"

	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/ring0"
	"gvisor.dev/gvisor/pkg/sentry/arch"
	"gvisor.dev/gvisor/pkg/sighandling"
)

var (
	// The action for bluepillSignal is changed by sigaction().
	bluepillSignal = unix.SIGILL
	debugEnterPC   uint64
	debugEnterA0   uint64
	debugEnterSP   uint64
	debugEnterTP   uint64

	debugExitPC uint64
	debugExitA0 uint64
	debugExitSP uint64
	debugExitTP uint64
)

// bluepill enters guest mode.
//
//go:nosplit
func bluepill(c *vCPU)

// bluepillArchEnter is called during bluepillEnter.
//
//go:nosplit
func bluepillArchEnter(context *arch.SignalContext64) (c *vCPU) {
	debugEnterPC = context.Regs[0]
	debugEnterA0 = context.Regs[10]
	debugEnterSP = context.Regs[2]
	debugEnterTP = context.Regs[4]
	c = vCPUPtr(uintptr(context.Regs[11]))
	regs := c.CPU.Registers()
	regs.Regs = context.Regs

	// TODO(riscv64): sstatus CSR manipulation needs a separate field
	// since PtraceRegs.Regs only has 32 entries (matching user_regs_struct).
	// KVM on riscv64 is not yet functional.

	return
}

// bluepillArchExit is called during bluepillEnter.
//
//go:nosplit
func bluepillArchExit(c *vCPU, context *arch.SignalContext64) {
	regs := c.CPU.Registers()

	debugExitPC = regs.Regs[0]
	debugExitA0 = regs.Regs[10]
	debugExitSP = regs.Regs[2]
	debugExitTP = regs.Regs[4]
	context.Regs = regs.Regs

	// Nested KVM compatibility mode does not maintain c's FP buffer. Preserve
	// the host signal frame's existing FP state instead of restoring stale data.
	printHex([]byte("RISC-V bluepill return PC:"), context.Regs[0])
	printHex([]byte("RISC-V bluepill return SP:"), context.Regs[2])
}

// KernelSyscall handles kernel syscalls.
//
// +checkescape:all
//
//go:nosplit
func (c *vCPU) KernelSyscall() {
	regs := c.Registers()
	printHex([]byte("RISC-V kernel syscall PC:"), regs.Regs[0])
	printHex([]byte("RISC-V kernel syscall a7:"), regs.Regs[17])
	if regs.Regs[17] == ^uint64(0) {
		regs.Regs[0] += 4 // Forward.
	}

	ring0.Halt()
}

// KernelException handles kernel exceptions.
//
// +checkescape:all
//
//go:nosplit
func (c *vCPU) KernelException(vector ring0.Vector) {
	regs := c.Registers()
	printHex([]byte("RISC-V kernel exception vector:"), uint64(vector))
	printHex([]byte("RISC-V kernel exception PC:"), regs.Regs[0])
	printHex([]byte("RISC-V kernel exception a7:"), regs.Regs[17])
	if vector == ring0.Vector(bounce) {
		regs.Regs[0] = 0
	}

	ring0.Halt()
}

// hltSanityCheck verifies the current state to detect obvious corruption.
//
//go:nosplit
func (c *vCPU) hltSanityCheck() {
}

// inKernelMode returns true if we are in S-mode (supervisor).
func inKernelMode() bool {
	return false
}

func init() {
	// Install the SIGILL handler used to enter KVM guest mode.
	if err := sighandling.ReplaceSignalHandler(
		bluepillSignal,
		addrOfSighandler(),
		&savedHandler,
	); err != nil {
		panic(fmt.Sprintf(
			"unable to set handler for signal %d: %v",
			bluepillSignal,
			err,
		))
	}
}
