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
	"reflect"
	"unsafe"

	"golang.org/x/sys/unix"
	"gvisor.dev/gvisor/pkg/abi/linux"
	"gvisor.dev/gvisor/pkg/hostarch"
	"gvisor.dev/gvisor/pkg/log"
	"gvisor.dev/gvisor/pkg/ring0"
	"gvisor.dev/gvisor/pkg/ring0/pagetables"
	"gvisor.dev/gvisor/pkg/sentry/platform"
	ktime "gvisor.dev/gvisor/pkg/sentry/time"
)

// initArchState initializes architecture-specific state.
func (m *machine) initArchState() error {
	m.mu.Lock()
	for i := 0; i < m.maxVCPUs; i++ {
		m.createVCPU(i)
	}
	m.mu.Unlock()
	return nil
}

// initArchState initializes architecture-specific state.
func (c *vCPU) initArchState() error {
	cpuVirtual := uintptr(unsafe.Pointer(&c.CPU))
	cpuPhysical, _, cpuOK := translateToPhysical(cpuVirtual)
	// Install a guest-writable virtual page backed by an otherwise unused GPA.
	// Do not use GPA 0: nested RISC-V KVM under QEMU intermittently reports the
	// resulting guest store-page fault as EIO instead of KVM_EXIT_MMIO.
	const (
		hypercallVirtual  = hostarch.PageSize
		hypercallPhysical = uintptr(0x1_0000_0000)
	)

	c.machine.kernel.PageTables.Map(
		hostarch.Addr(hypercallVirtual),
		hostarch.PageSize,
		pagetables.MapOpts{AccessType: hostarch.ReadWrite},
		hypercallPhysical, // Deliberately absent from KVM memory slots.
	)

	riscv64HypercallMMIOBase = hypercallPhysical
	stackKernel := uintptr(c.CPU.StackTop())
	stackVirtual := (stackKernel & ring0.MaximumUserAddress) - 1
	stackPhysical, _, stackOK := translateToPhysical(stackVirtual)

	_, cpuBlockPhysical, _, cpuRegion :=
		calculateBluepillFault(cpuPhysical)
	_, stackBlockPhysical, _, stackRegion :=
		calculateBluepillFault(stackPhysical)

	printHex([]byte("RISC-V CPU virtual:"), uint64(cpuVirtual))
	printHex([]byte("RISC-V CPU physical:"), uint64(cpuPhysical))
	printHex(
		[]byte("RISC-V CPU physical end:"),
		uint64(cpuPhysical+unsafe.Sizeof(c.CPU)),
	)
	printHex([]byte("RISC-V CPU block:"), uint64(cpuBlockPhysical))

	printHex([]byte("RISC-V stack kernel:"), uint64(stackKernel))
	printHex([]byte("RISC-V stack virtual:"), uint64(stackVirtual))
	printHex([]byte("RISC-V stack physical:"), uint64(stackPhysical))
	printHex([]byte("RISC-V stack block:"), uint64(stackBlockPhysical))

	log.Infof(
		"RISC-V CPU mapping flags: cpuOK=%t cpuMMIO=%t cpuRO=%t stackOK=%t stackMMIO=%t stackRO=%t",
		cpuOK,
		cpuRegion != nil && cpuRegion.mmio,
		cpuRegion != nil && cpuRegion.readOnly,
		stackOK,
		stackRegion != nil && stackRegion.mmio,
		stackRegion != nil && stackRegion.readOnly,
	)

	fpState := c.CPU.FloatingPointState()
	virtual := uintptr(unsafe.Pointer(fpState.BytePointer()))
	remaining := uintptr(len(*fpState))

	log.Infof(
		"RISC-V kernel FP state: virtual=%#x length=%#x",
		virtual,
		remaining,
	)

	cpuAddress := uintptr(unsafe.Pointer(&c.CPU))
	fpHeaderAddress := uintptr(unsafe.Pointer(fpState))

	log.Infof(
		"RISC-V CPU layout: cpu=%#x fpHeader=%#x fpOffset=%d",
		cpuAddress,
		fpHeaderAddress,
		fpHeaderAddress-cpuAddress,
	)
	for remaining != 0 {
		physical, length, ok := translateToPhysical(virtual)
		log.Infof(
			"RISC-V kernel FP mapping: virtual=%#x physical=%#x length=%#x ok=%t",
			virtual,
			physical,
			length,
			ok,
		)

		if !ok {
			return fmt.Errorf(
				"cannot translate floating-point state address %#x",
				virtual,
			)
		}
		if length > remaining {
			length = remaining
		}

		log.Infof(
			"RISC-V kernel FP slot: virtual=%#x physical=%#x length=%#x",
			virtual,
			physical,
			length,
		)

		c.machine.mapPhysical(physical, length)

		virtual += length
		remaining -= length
	}
	var (
		reg     kvmOneReg
		data    uint64
		regGet  kvmOneReg
		dataGet uint64
	)

	reg.addr = uint64(reflect.ValueOf(&data).Pointer())
	regGet.addr = uint64(reflect.ValueOf(&dataGet).Pointer())

	// isa
	reg.id = _KVM_RISCV64_REGS_ISA
	data = _RISCV64_ISA_GC
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// Enable floating-point extensions through the per-extension KVM API.
	// The legacy config ISA bitmap above is retained for older kernels.
	data = 1
	reg.id = _KVM_RISCV64_ISA_EXT_F
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}
	reg.id = _KVM_RISCV64_ISA_EXT_D
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// tp
	reg.id = _KVM_RISCV64_REGS_TP
	data = uint64(reflect.ValueOf(&c.CPU).Pointer() | ring0.KernelStartAddress)
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// sscratch
	reg.id = _KVM_RISCV64_REGS_SSCRATCH
	data = uint64(reflect.ValueOf(&c.CPU).Pointer() | ring0.KernelStartAddress)
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// sp
	reg.id = _KVM_RISCV64_REGS_SP
	data = c.CPU.StackTop()
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// satp
	reg.id = _KVM_RISCV64_REGS_SATP
	data = c.machine.kernel.PageTables.SATP(false, 0)
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}
	c.SetSatpKvm(uintptr(data))

	// pc
	reg.id = _KVM_RISCV64_REGS_PC
	data = uint64(ring0.AddrOfStart())
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// sie
	reg.id = _KVM_RISCV64_REGS_SIE
	data = _SIE_DEFAULT
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// stvec
	reg.id = _KVM_RISCV64_REGS_STVEC

	vectorLocation := ring0.AddrOfVectors()
	vectorKernelAddress :=
		ring0.KernelStartAddress | (vectorLocation &^ uintptr(0x3))

	data = uint64(vectorKernelAddress)
	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	// Initialize the PCID database.
	if hasGuestPCID {
		// Note that NewPCIDs may return a nil table here, in which
		// case we simply don't use PCID support (see below). In
		// practice, this should not happen, however.
		c.PCIDs = pagetables.NewPCIDs(fixedKernelPCID+1, poolPCIDs)
	}

	return c.setSystemTimeLegacy()
}

// setTSC sets the counter Virtual Offset.
func (c *vCPU) setTSC(value uint64) error {
	var (
		reg  kvmOneReg
		data uint64
	)

	reg.addr = uint64(reflect.ValueOf(&data).Pointer())
	reg.id = _KVM_RISCV64_REGS_TIMER_CNT
	data = uint64(value)

	if err := c.setOneRegister(&reg); err != nil {
		return err
	}

	return nil
}

// getTSC gets the counter Physical Counter minus Virtual Offset.
func (c *vCPU) getTSC() error {
	var (
		reg  kvmOneReg
		data uint64
	)

	reg.addr = uint64(reflect.ValueOf(&data).Pointer())
	reg.id = _KVM_RISCV64_REGS_TIMER_CNT

	if err := c.getOneRegister(&reg); err != nil {
		return err
	}

	return nil
}

// setSystemTime sets the vCPU to the system time.
func (c *vCPU) setSystemTime() error {
	const minIterations = 10
	minimum := uint64(0)
	for iter := 0; ; iter++ {
		// Use get the TSC to an estimate of where it will be
		// on the host during a "fast" system call iteration.
		// replace getTSC to another setOneRegister syscall can get more accurate value?
		start := uint64(ktime.Rdtsc())
		if err := c.getTSC(); err != nil {
			return err
		}
		// See if this is our new minimum call time. Note that this
		// serves two functions: one, we make sure that we are
		// accurately predicting the offset we need to set. Second, we
		// don't want to do the final set on a slow call, which could
		// produce a really bad result.
		end := uint64(ktime.Rdtsc())
		if end < start {
			continue // Totally bogus: unstable TSC?
		}
		current := end - start
		if current < minimum || iter == 0 {
			minimum = current // Set our new minimum.
		}
		// Is this past minIterations and within ~10% of minimum?
		upperThreshold := (((minimum << 3) + minimum) >> 3)
		if iter >= minIterations && (current <= upperThreshold || minimum < 50) {
			// Try to set the TSC
			if err := c.setTSC(end + (minimum / 2)); err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}

//go:nosplit
func (c *vCPU) loadSegments(tid uint64) {
	// TODO(gvisor.dev/issue/1238):  TLS is not supported.
	// Get TLS from tpidr_el0.
	c.tid.Store(tid)
}

//go:nosplit
func (c *vCPU) setOneRegister(reg *kvmOneReg) error {
	if _, _, errno := unix.RawSyscall(
		unix.SYS_IOCTL,
		uintptr(c.fd),
		_KVM_SET_ONE_REG,
		uintptr(unsafe.Pointer(reg))); errno != 0 {
		return fmt.Errorf("error setting one register: %v", errno)
	}
	return nil
}

//go:nosplit
func (c *vCPU) getOneRegister(reg *kvmOneReg) error {
	if _, _, errno := unix.RawSyscall(
		unix.SYS_IOCTL,
		uintptr(c.fd),
		_KVM_GET_ONE_REG,
		uintptr(unsafe.Pointer(reg))); errno != 0 {
		return fmt.Errorf("error getting one register: %v", errno)
	}
	return nil
}

// SwitchToUser unpacks architectural-details.
func (c *vCPU) SwitchToUser(switchOpts ring0.SwitchOpts, info *linux.SignalInfo) (hostarch.AccessType, error) {
	// Check for canonical addresses.
	if regs := switchOpts.Registers; !ring0.IsCanonical(regs.Regs[0]) {
		return nonCanonical(regs.Regs[0], int32(unix.SIGSEGV), info)
	} else if !ring0.IsCanonical(regs.Regs[2]) {
		return nonCanonical(regs.Regs[2], int32(unix.SIGSEGV), info)
	}

	// Assign PCIDs.
	if c.PCIDs != nil {
		var requireFlushPCID bool // Force a flush?
		switchOpts.UserASID, requireFlushPCID = c.PCIDs.Assign(switchOpts.PageTables)
		switchOpts.Flush = switchOpts.Flush || requireFlushPCID
	}

	var vector ring0.Vector
	satpVal := switchOpts.PageTables.SATP(false, 0)
	c.SetSatpApp(uintptr(satpVal))

	// Full context-switch supporting for RISCV64.
	// The RISCV64 user-mode execution state consists of:
	// X0-X31
	// F0-F31: 32 128-bit registers for floating point, and simd
	// FCSR
	appRegs := switchOpts.Registers
	c.SetAppAddr(ring0.KernelStartAddress | uintptr(unsafe.Pointer(appRegs)))

	// The active goroutine stack may have been allocated after the machine's
	// initial memory-slot scan. Ensure its complete physical fault block is
	// registered before returning from bluepill into Go code in guest mode.
	stackAnchorVirtual := uintptr(unsafe.Pointer(&vector))
	stackAnchorPhysical, _, stackAnchorOK :=
		translateToPhysical(stackAnchorVirtual)
	if !stackAnchorOK {
		return hostarch.NoAccess, fmt.Errorf(
			"cannot translate active Go stack address %#x",
			stackAnchorVirtual,
		)
	}

	_, stackBlockPhysical, stackBlockLength, stackRegion :=
		calculateBluepillFault(stackAnchorPhysical)
	if stackRegion == nil || stackBlockLength == 0 {
		return hostarch.NoAccess, fmt.Errorf(
			"cannot locate active Go stack physical address %#x",
			stackAnchorPhysical,
		)
	}
	if stackRegion.mmio {
		return hostarch.NoAccess, fmt.Errorf(
			"active Go stack address %#x is marked MMIO",
			stackAnchorVirtual,
		)
	}

	log.Infof(
		"RISC-V active Go stack: virtual=%#x physical=%#x block=[%#x,%#x)",
		stackAnchorVirtual,
		stackAnchorPhysical,
		stackBlockPhysical,
		stackBlockPhysical+stackBlockLength,
	)

	c.machine.mapPhysical(stackBlockPhysical, stackBlockLength)

	entersyscall()
	bluepill(c)
	vector = c.CPU.SwitchToUser(switchOpts)
	exitsyscall()

	switch vector {
	case ring0.EcallFromUser:
		// Fast path: system call executed.
		return hostarch.NoAccess, nil
	case ring0.InstructionAccessFault:
		return c.fault(int32(unix.SIGSEGV), info)
	case ring0.IllegalInstruction:
		return c.fault(int32(unix.SIGILL), info)
	case ring0.LoadAccessFault:
		return c.fault(int32(unix.SIGSEGV), info)
	case ring0.StoreAccessFault:
		return c.fault(int32(unix.SIGSEGV), info)
	case ring0.InstPageFault:
		return c.fault(int32(unix.SIGSEGV), info)
	case ring0.LoadPageFault:
		return c.fault(int32(unix.SIGSEGV), info)
	case ring0.StorePageFault:
		return c.fault(int32(unix.SIGSEGV), info)
	case ring0.ExtDabt:
		return c.fault(int32(unix.SIGSEGV), info)
	case ring0.Sigbus:
		return c.fault(int32(unix.SIGBUS), info)
	case ring0.Vector(bounce): // ring0.VirtualizationException.
		return hostarch.NoAccess, platform.ErrContextInterrupt
	default:
		panic(fmt.Sprintf("unexpected vector: 0x%x", vector))
	}
}

//go:nosplit
func seccompMmapSyscall(context unsafe.Pointer) (uintptr, uintptr, uintptr, unix.Errno) {
	ctx := bluepillArchContext(context)

	length := uintptr(ctx.Regs[11])
	prot := uintptr(ctx.Regs[12])

	// MAP_DENYWRITE is deprecated and ignored by Linux. It is used only
	// to distinguish this mmap from mappings allowed directly by seccomp.
	addr, _, errno := unix.RawSyscall6(
		uintptr(ctx.Regs[17]),
		uintptr(ctx.Regs[10]),
		length,
		prot,
		uintptr(ctx.Regs[13])|unix.MAP_DENYWRITE,
		uintptr(ctx.Regs[14]),
		uintptr(ctx.Regs[15]),
	)
	ctx.Regs[10] = uint64(addr)

	return addr, length, prot, errno
}
