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

package kvm

// KVM ioctls for Riscv64.
const (
	_KVM_GET_ONE_REG = 0x4010aeab
	_KVM_SET_ONE_REG = 0x4010aeac

	_KVM_RISCV64_REG_TYPE_SHIFT = 24
	_KVM_RISCV64_REGS_ISA       = 0x8030000001000000
	_KVM_RISCV64_ISA_EXT_D      = 0x8030000007000002
	_KVM_RISCV64_ISA_EXT_F      = 0x8030000007000003
	_KVM_RISCV64_REGS           = 0x8030000002000000
	_KVM_RISCV64_FPREGS         = 0x8030000006000000
	_KVM_RISCV64_FPREGS_FCSR    = 0x8020000006000020
	_KVM_RISCV64_REGS_CORE      = 0x02 << _KVM_RISCV64_REG_TYPE_SHIFT
	_KVM_RISCV64_REG_SIZE       = 1 << 6
	_KVM_RISCV64_REGS_PC        = 0x8030000002000000
	_KVM_RISCV64_REGS_SP        = 0x8030000002000002
	_KVM_RISCV64_REGS_TP        = 0x8030000002000004
	_KVM_RISCV64_REGS_SSTATUS   = 0x8030000003000000
	_KVM_RISCV64_REGS_SIE       = 0x8030000003000001
	_KVM_RISCV64_REGS_STVEC     = 0x8030000003000002
	_KVM_RISCV64_REGS_SEPC      = 0x8030000003000004
	_KVM_RISCV64_REGS_SATP      = 0x8030000003000008
	_KVM_RISCV64_REGS_SCAUSE    = 0x8030000003000005
	_KVM_RISCV64_REGS_SSCRATCH  = 0x8030000003000003

	_KVM_RISCV64_REGS_TIMER_CNT = 0x8030000004000001

	_KVM_RISCV64_REGS_FP   = 0x8030000006000000
	_KVM_RISCV64_REGS_FCSR = 0x8030000006000020

	// ===== H-Mode Supervisor Registers (H-Extension) =====
	// Exception and interrupt handling in HS-Mode
	_KVM_RISCV64_REGS_HEPC    = 0x8030000003000004 // HS-Mode Exception PC
	_KVM_RISCV64_REGS_HSTATUS = 0x8030000003000000 // HS-Mode Status Register
	_KVM_RISCV64_REGS_HCAUSE  = 0x8030000003000006 // HS-Mode Exception Cause
	_KVM_RISCV64_REGS_HTVAL   = 0x8030000003000033 // HS-Mode Trap Value
	_KVM_RISCV64_REGS_HTVEC   = 0x8030000003000037 // HS-Mode Trap Vector

	// Exception and interrupt delegation
	_KVM_RISCV64_REGS_HEDELEG = 0x8030000003000038 // HS-Mode Exception Delegation
	_KVM_RISCV64_REGS_HIDELEG = 0x8030000003000039 // HS-Mode Interrupt Delegation

	// Nested virtualization
	_KVM_RISCV64_REGS_HGATP  = 0x8030000003000039 // HS-Mode Guest Address Translation and Protection
	_KVM_RISCV64_REGS_HVIP   = 0x8030000003000045 // HS-Mode Virtual Interrupt Pending
	_KVM_RISCV64_REGS_HVICTL = 0x8030000003000046 // HS-Mode Virtual Interrupt Control

	// Scratch and auxiliary
	_KVM_RISCV64_REGS_HSCRATCH = 0x8030000003000040 // HS-Mode Scratch Register
	_KVM_RISCV64_REGS_HGEIE    = 0x8030000003000047 // HS-Mode Guest External Interrupt Enable
	_KVM_RISCV64_REGS_HGEIP    = 0x8030000003000041 // HS-Mode Guest External Interrupt Pending

	// ===== Virtual Supervisor Registers (for VS-Mode guests) =====
	_KVM_RISCV64_REGS_VSEPC     = 0x8030000003000104 // VS-Mode Exception PC
	_KVM_RISCV64_REGS_VSSTATUS  = 0x8030000003000100 // VS-Mode Status Register
	_KVM_RISCV64_REGS_VSCAUSE   = 0x8030000003000106 // VS-Mode Exception Cause
	_KVM_RISCV64_REGS_VSTVAL    = 0x8030000003000133 // VS-Mode Trap Value
	_KVM_RISCV64_REGS_VSTVEC    = 0x8030000003000137 // VS-Mode Trap Vector
	_KVM_RISCV64_REGS_VSSCRATCH = 0x8030000003000140 // VS-Mode Scratch Register
	_KVM_RISCV64_REGS_VSATP     = 0x8030000003000180 // VS-Mode Address Translation and Protection
)

// Riscv64: Supervisor Interrupt Enable register
const (
	_SIE_SSIE = 1 << 1
	_SIE_UTIE = 1 << 4
	//_SIE_SEIE = 1 << 9
	_SIE_UEIE    = 1 << 8
	_SIE_VSIE    = 1 << 10
	_SIE_DEFAULT = _SIE_SSIE | _SIE_UTIE | _SIE_UEIE | _SIE_VSIE
)

const (
	_KVM_EXIT_RISCV_SBI = 35
)

const (
	// on Riscv64, the MMIO address must be 64-bit aligned.
	// Currently, we only need 1 hypercall: hypercall_vmexit.
	_RISCV64_HYPERCALL_MMIO_SIZE = 1 << 2
)

const (
	_RISCV64_ISA_MXL = 2 << 32
	_RISCV64_ISA_A   = 1 << 0
	_RISCV64_ISA_C   = 1 << 2
	_RISCV64_ISA_F   = 1 << 5
	_RISCV64_ISA_D   = 1 << 3
	_RISCV64_ISA_I   = 1 << 8
	_RISCV64_ISA_M   = 1 << 12
	_RISCV64_ISA_GC  = _RISCV64_ISA_A | _RISCV64_ISA_C | _RISCV64_ISA_D | _RISCV64_ISA_F | _RISCV64_ISA_I | _RISCV64_ISA_M
)

// ===== Guest Page Fault Exception Codes (H-Extension specific) =====
const (
	_GUEST_INST_PAGE_FAULT  = 0x14 // Guest Instruction Page Fault
	_GUEST_LOAD_PAGE_FAULT  = 0x15 // Guest Load Page Fault
	_GUEST_STORE_PAGE_FAULT = 0x16 // Guest Store/AMO Page Fault
)

// ===== H-Extension Status Register (HSTATUS) Bits =====
const (
	_HSTATUS_VSXLEN = uint64(1) << 17 // Virtual XLEN
	_HSTATUS_VTSR   = uint64(1) << 22 // Virtual Timer Interrupt
	_HSTATUS_VTW    = uint64(1) << 21 // Virtual Trap on WFI
	_HSTATUS_HUPMIE = uint64(1) << 4  // HUP Mode Interrupt Enable
	_HSTATUS_SPV    = uint64(1) << 7  // Supervisor Previous Virtualization Mode
	_HSTATUS_GVA    = uint64(1) << 6  // Guest Virtual Address
	_HSTATUS_SPVP   = uint64(1) << 5  // Supervisor Previous Virtualization Mode (previous)
)

// ===== HGATP Register (Guest Address Translation and Protection) Bits =====
const (
	_HGATP_MODE_SHIFT = 60
	_HGATP_MODE_MASK  = uint64(0xF) << _HGATP_MODE_SHIFT // Page Table Mode
	_HGATP_MODE_BARE  = uint64(0x0) << _HGATP_MODE_SHIFT // No translation
	_HGATP_MODE_SV39X = uint64(0x8) << _HGATP_MODE_SHIFT // 39-bit with NAPOT
	_HGATP_MODE_SV48X = uint64(0x9) << _HGATP_MODE_SHIFT // 48-bit with NAPOT

	_HGATP_VMID_SHIFT = 44
	_HGATP_VMID_MASK  = uint64(0x3FFF) << _HGATP_VMID_SHIFT // Virtual Machine ID (14 bits)

	_HGATP_PPN_SHIFT = 0
	_HGATP_PPN_MASK  = uint64(0xFFFFFFFFFFF) // Physical Page Number (44 bits)
)

// ===== SBI Extension IDs =====
const (
	_SBI_EXT_TIMER  = 0x54494D45 // "TIME"
	_SBI_EXT_IPI    = 0x735049   // "sPI"
	_SBI_EXT_RFENCE = 0x52464E43 // "RFNC"
	_SBI_EXT_HSM    = 0x48534D   // "HSM"
	_SBI_EXT_SRST   = 0x53525354 // "SRST"
	_SBI_EXT_PMU    = 0x504D55   // "PMU"
)

// ===== SBI Function IDs =====
const (
	// Timer extension functions
	_SBI_TIMER_SET_TIMER = 0

	// IPI extension functions
	_SBI_IPI_SEND = 0

	// RFENCE extension functions
	_SBI_RFENCE_I     = 0
	_SBI_RFENCE_VMA   = 1
	_SBI_RFENCE_VMAID = 2

	// HSM (Hart State Management) functions
	_SBI_HSM_HART_START      = 0
	_SBI_HSM_HART_STOP       = 1
	_SBI_HSM_HART_GET_STATUS = 2
	_SBI_HSM_HART_SUSPEND    = 3
)
