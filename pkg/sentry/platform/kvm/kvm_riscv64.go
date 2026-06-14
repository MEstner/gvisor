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

	"gvisor.dev/gvisor/pkg/abi/linux"
	"gvisor.dev/gvisor/pkg/ring0"
	"gvisor.dev/gvisor/pkg/sentry/arch"
)

type kvmOneReg struct {
	id   uint64
	addr uint64
}

// riscv64HypercallMMIOBase is MMIO base address used to dispatch hypercalls.
var riscv64HypercallMMIOBase uintptr

type userRegs struct {
	Regs    arch.Registers
	Sstatus uint64
}

type exception struct {
	sErrPending    uint8
	sErrHasEsr     uint8
	extDabtPending uint8
	pad            [5]uint8
	sErrEsr        uint64
}

type kvmVcpuEvents struct {
	exception
	rsvd [12]uint32
}

// updateGlobalOnce does global initialization. It has to be called only once.
func updateGlobalOnce(fd int) error {
	err := updateSystemValues(int(fd))
	ring0.Init()
	physicalInit()
	// The linux.Task represents the possible largest task size, which the UserspaceSize shouldn't be larger than.
	if linux.TaskSize < ring0.UserspaceSize {
		return fmt.Errorf("gVisor doesn't support 3-level page tables on KVM platform.")
	}
	return err
}

// ============================================
// SBI Handler Implementation for H-Extension
// ============================================

// handleSBIExtTimer handles SBI_EXT_TIMER extension calls.
// Timer extension allows VS-Mode guests to set timer interrupts.
func handleSBIExtTimer(funcID uint64, regs *arch.Registers) {
	// Function IDs within EXT_TIMER:
	// 0: sbi_set_timer - Set next timer event

	switch funcID {
	case _SBI_TIMER_SET_TIMER:
		// Timer value is in A0 (regs[10])
		// For now, we don't support actual timer emulation
		// Just return success (0)
		regs.Regs[10] = 0 // A0 = return value
		regs.Regs[11] = 0 // A1 = error code
	default:
		// Unknown timer function ID
		regs.Regs[10] = ^uint64(0) // Return -1 (error)
		regs.Regs[11] = 1          // Error code: not supported
	}
}

// handleSBIExtIPI handles SBI_EXT_IPI extension calls.
// IPI extension allows inter-processor interrupts between virtual cores.
func handleSBIExtIPI(funcID uint64, regs *arch.Registers) {
	// Function IDs within EXT_IPI:
	// 0: sbi_send_ipi - Send IPI to other harts

	switch funcID {
	case _SBI_IPI_SEND:
		// Hart mask is in A0, IPI type is in A1
		// For now, return success
		regs.Regs[10] = 0 // A0 = return value
		regs.Regs[11] = 0 // A1 = error code
	default:
		regs.Regs[10] = ^uint64(0) // Return -1 (error)
		regs.Regs[11] = 1          // Error code: not supported
	}
}

// handleSBIExtRFence handles SBI_EXT_RFENCE extension calls.
// Remote Fence extension allows remote TLB/SFENCE operations.
func handleSBIExtRFence(funcID uint64, regs *arch.Registers) {
	// Function IDs within EXT_RFENCE:
	// 0: sbi_remote_fence_i - Remote instruction fence
	// 1: sbi_remote_sfence_vma - Remote SFENCE.VMA
	// 2: sbi_remote_sfence_vma_asid - Remote SFENCE.VMA with ASID

	switch funcID {
	case _SBI_RFENCE_I, _SBI_RFENCE_VMA, _SBI_RFENCE_VMAID:
		// For now, return success (in nested virt, TLB ops are local)
		regs.Regs[10] = 0 // A0 = return value
		regs.Regs[11] = 0 // A1 = error code
	default:
		regs.Regs[10] = ^uint64(0) // Return -1 (error)
		regs.Regs[11] = 1          // Error code: not supported
	}
}

// handleSBIExtHSM handles SBI_EXT_HSM extension calls.
// HSM extension manages hart state (start, stop, resume).
func handleSBIExtHSM(funcID uint64, regs *arch.Registers) {
	// Function IDs within EXT_HSM:
	// 0: sbi_hart_start - Start a hart
	// 1: sbi_hart_stop - Stop a hart
	// 2: sbi_hart_get_status - Get hart status
	// 3: sbi_hart_suspend - Suspend a hart

	switch funcID {
	case _SBI_HSM_HART_START, _SBI_HSM_HART_STOP, _SBI_HSM_HART_GET_STATUS:
		// Hart management in nested virtualization is complex
		// For now, return SBI_ERR_INVALID_PARAM
		regs.Regs[10] = ^uint64(0) // Return error
		regs.Regs[11] = 2          // SBI_ERR_INVALID_PARAM
	default:
		regs.Regs[10] = ^uint64(0) // Return -1 (error)
		regs.Regs[11] = 1          // Error code: not supported
	}
}

// handleSBIExtSRST handles SBI_EXT_SRST extension calls.
// System Reset extension allows graceful shutdown/reboot.
func handleSBIExtSRST(funcID uint64, regs *arch.Registers) {
	// This extension is optional and used for reboot/shutdown
	// For now, return not implemented
	regs.Regs[10] = ^uint64(0) // Return -1 (error)
	regs.Regs[11] = 1          // Error code: not supported
}

// processSBICall handles incoming SBI calls from VS-Mode guests.
// It dispatches to the appropriate handler based on extension ID.
func processSBICall(extID uint64, funcID uint64, regs *arch.Registers) {
	switch extID {
	case _SBI_EXT_TIMER:
		handleSBIExtTimer(funcID, regs)
	case _SBI_EXT_IPI:
		handleSBIExtIPI(funcID, regs)
	case _SBI_EXT_RFENCE:
		handleSBIExtRFence(funcID, regs)
	case _SBI_EXT_HSM:
		handleSBIExtHSM(funcID, regs)
	case _SBI_EXT_SRST:
		handleSBIExtSRST(funcID, regs)
	default:
		// Unknown SBI extension - return not supported
		regs.Regs[10] = ^uint64(0) // Return -1 (error)
		regs.Regs[11] = 1          // Error code: not supported
	}
}
