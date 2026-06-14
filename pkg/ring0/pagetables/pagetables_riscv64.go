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

package pagetables

import (
	"sync/atomic"

	"gvisor.dev/gvisor/pkg/hostarch"
)

// archPageTables is architecture-specific data.
type archPageTables struct {
	// root is the pagetable root for kernel space.
	root *PTEs

	// rootPhysical is the cached physical address of the root.
	//
	// This is saved only to prevent constant translation.
	rootPhysical uintptr

	asid uint16
}

// SATP returns the translation table base register
//
//go:nosplit
func (p *PageTables) SATP(noFlush bool, asid uint16) uint64 {
	return ((uint64(p.rootPhysical) >> satpPPNOffset) & satpPPNMask) | (uint64(asid) << satpASIDOffset) | uint64(satpMode)
}

// MapOpts are riscv64 options.
type MapOpts struct {
	// AccessType defines permissions.
	AccessType hostarch.AccessType

	// Global indicates the page is globally accessible.
	Global bool

	// User indicates the page is a user page.
	User bool

	// MemoryType is unused on riscv64, but needed for interface compatibility.
	MemoryType hostarch.MemoryType
}

// PTE is a page table entry.
type PTE uintptr

// Bits in page table entries.
// Reference:
// riscv-privileged-v1.10.pdf
// arch/riscv/include/asm/pgtable-bits.h
const (
	// R/W access permission
	//typeTable   = 0x3 << 1
	typeSect = pteValid | readable | executable
	//typePage    = 0x3 << 1
	pteValid = 0x1 << 0
	present  = pteValid
	//pteTableBit = 0x1 << 1
	//pteTypeMask = 0x3 << 0
	//present     = pteValid | pteTableBit
	readable   = 0x1 << 1
	writable   = 0x1 << 2
	executable = 0x1 << 3
	user       = 0x1 << 4
	global     = 0x1 << 5
	accessed   = 0x1 << 6
	dirty      = 0x1 << 7

	typeTable  = 0x1 << 0
	pageOffset = 12
)

const (
	// include RSW
	optionOffset = 10
	optionMask   = 0x3ff
	ppnMask      = 0x3ffffffffffc00
	protDefault  = present | accessed | user | dirty
)

// Address extracts the address. This should only be used if Valid returns true.
//
//go:nosplit
func (p *PTE) Address() uintptr {
	return (atomic.LoadUintptr((*uintptr)(p)) &^ optionMask) << (pageOffset - optionOffset)
}

// Set sets this PTE value.
//
// This does not change the sect page property.
//
//go:nosplit
func (p *PTE) Set(addr uintptr, opts MapOpts) {
	v := ((addr >> (pageOffset - optionOffset)) & ppnMask) | readable | protDefault

	if !opts.AccessType.Any() {
		// Leave as non-valid if no access is available.
		v &^= pteValid
	}

	if opts.Global {
		v |= global
	}

	if opts.AccessType.Execute {
		v |= executable
	}

	if opts.AccessType.Write {
		v |= writable
	}

	if opts.User {
		v |= user
	} else {
		v = v &^ user
	}

	atomic.StoreUintptr((*uintptr)(p), v)
}

// Clear clears this PTE, including the super page information
//
//go:nosplit
func (p *PTE) Clear() {
	atomic.StoreUintptr((*uintptr)(p), 0)
}

// Valid returns true iff this entry is valid.
//
//go:nosplit
func (p *PTE) Valid() bool {
	return atomic.LoadUintptr((*uintptr)(p))&present != 0
}

// Opts returns the PTE options.
//
// These are all options except Valid and Sect.
//
//go:nosplit
func (p *PTE) Opts() MapOpts {
	v := atomic.LoadUintptr((*uintptr)(p))

	return MapOpts{
		AccessType: hostarch.AccessType{
			Read:    true,
			Write:   v&writable != 0,
			Execute: v&executable != 0,
		},
		Global: v&global != 0,
		User:   v&user != 0,
	}
}

// SetSect sets this page as a sect page.
//
// The page must not be valid or a panic will result.
// riscv-privileged-v1.10 4.3 Virtual Address Translation Process
//
//go:nosplit
func (p *PTE) SetSect() {
	if p.Valid() {
		// This is not allowed.
		panic("SetSect called on valid page!")
	}
	atomic.StoreUintptr((*uintptr)(p), protDefault|readable)
}

// IsSect returns true iff this page is a sect page.
//
//go:nosplit
func (p *PTE) IsSect() bool {
	return atomic.LoadUintptr((*uintptr)(p))&(executable|readable) != 0
}

// setPageTable sets this PTE value and forces the write bit and sect bit to
// be cleared. This is used explicitly for breaking sect pages.
//
//go:nosplit
func (p *PTE) setPageTable(pt *PageTables, ptes *PTEs) {
	addr := pt.Allocator.PhysicalFor(ptes)
	if addr&^optionMask != addr {
		// This should never happen.
		panic("unaligned physical address!")
	}
	v := ((addr >> (pageOffset - optionOffset)) & ppnMask) | typeTable
	atomic.StoreUintptr((*uintptr)(p), v)
}

// Address constraints.
//
// The lowerTop and upperBottom currently apply to four-level pagetables;
// additional refactoring would be necessary to support five-level pagetables.
const (
	lowerTop    = 0x00007fffffffffff
	upperBottom = 0xffff800000000000
	pteShift    = 12
	pmdShift    = 21
	pudShift    = 30
	pgdShift    = 39

	pteMask = 0x1ff << pteShift
	pmdMask = 0x1ff << pmdShift
	pudMask = 0x1ff << pudShift
	pgdMask = 0x1ff << pgdShift

	pteSize = 1 << pteShift
	pmdSize = 1 << pmdShift
	pudSize = 1 << pudShift
	pgdSize = 1 << pgdShift

	satpASIDOffset = 44
	satpPPNOffset  = 12
	satpPPNMask    = 0xfffffffffff
	// Sv48
	satpMode = 0x9000000000000000

	entriesPerPage = 512
)

// InitArch does some additional initialization related to the architecture.
//
// +checkescape:hard,stack
//
//go:nosplit
func (p *PageTables) InitArch(allocator Allocator) {
	if p.upperSharedPageTables != nil {
		p.cloneUpperShared()
	}
}

//go:nosplit
func pgdIndex(upperStart uintptr) uintptr {
	if upperStart&(pgdSize-1) != 0 {
		panic("upperStart should be pgd size aligned")
	}
	if upperStart >= upperBottom {
		return entriesPerPage/2 + (upperStart-upperBottom)/pgdSize
	}
	if upperStart < lowerTop {
		return upperStart / pgdSize
	}
	panic("upperStart should be in canonical range")
}

// cloneUpperShared clone the upper from the upper shared page tables.
//
//go:nosplit
func (p *PageTables) cloneUpperShared() {
	start := pgdIndex(p.upperStart)
	copy(p.root[start:entriesPerPage], p.upperSharedPageTables.root[start:entriesPerPage])
}

// PTEs is a collection of entries.
type PTEs [entriesPerPage]PTE

// ============================================
// H-Extension Guest Address Translation Support
// ============================================

// HGATP returns the Guest Address Translation and Protection register value
// for nested virtualization in H-Extension mode.
//
// HGATP structure:
// - Bits [63]: MODE (1 = SV39X, 0 = BARE)
// - Bits [62:44]: VMID (Virtual Machine ID)
// - Bits [43:0]: PPN (Physical Page Number of guest page tables)
//
// This enables VS-Mode guests to use their own page tables under HS-Mode
// hypervisor control through first-stage (HGATP) and second-stage page walks.
//
//go:nosplit
func (p *PageTables) HGATP(vmid uint16) uint64 {
	// Build HGATP value with SV39X mode and given VMID
	// Mode bits (63:60): 0x8 = SV39X (two-level guest address translation)
	// VMID bits (59:44): Virtual machine identifier
	// PPN bits (43:0): Physical page number of root guest page table

	const (
		hgatpModeShift = 60
		hgatpModeSV39X = 0x8
		hgatpVMIDShift = 44
		hgatpVMIDMask  = 0xFFFF
		hgatpPPNMask   = 0x0FFFFFFFFFFFFFFF
	)

	// Extract physical page number from root page table physical address
	ppn := (uint64(p.rootPhysical) >> satpPPNOffset) & hgatpPPNMask

	// Build HGATP: MODE (63:60) | VMID (59:44) | PPN (43:0)
	hgatp := (uint64(hgatpModeSV39X) << hgatpModeShift) |
		(uint64(vmid&hgatpVMIDMask) << hgatpVMIDShift) |
		ppn

	return hgatp
}

// GuestPageFaultHandler handles page faults from VS-Mode guests.
//
// This is called when a guest page translation fails. The fault may be:
// - First-stage fault (guest SATP translation fails)
// - Second-stage fault (hypervisor HGATP translation fails)
//
// Parameters:
// - gpa: Guest Physical Address that caused the fault
// - gva: Guest Virtual Address (if available)
// - faultType: Type of access that caused fault (instruction/load/store)
//
// Returns true if fault was handled, false if it should propagate to guest.
//
//go:nosplit
func (p *PageTables) GuestPageFaultHandler(gpa uintptr, gva uintptr, faultType uint32) bool {
	// TODO: Implement guest page fault handling
	// 1. Check if faultType is one of:
	//    - 0x14: Guest Instruction Page Fault
	//    - 0x15: Guest Load Page Fault
	//    - 0x16: Guest Store Page Fault
	// 2. Translate GPA to host physical address
	// 3. Allocate physical page if needed
	// 4. Update guest HGATP entries
	// 5. Return true if resolved

	return false
}

// IsGuestPageFault checks if the trap is a guest page fault
//
// Guest page faults have exception codes:
// - 0x14: Guest Instruction Page Fault
// - 0x15: Guest Load Page Fault
// - 0x16: Guest Store Page Fault
//
//go:nosplit
func IsGuestPageFault(trapValue uint64) bool {
	switch trapValue {
	case 0x14, 0x15, 0x16:
		return true
	default:
		return false
	}
}
