// Copyright 2026 The gVisor Authors.
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

package linux

// TASK_SIZE for riscv64 is 2^38 (256GB).
// TASK_SIZE depends on the paging mode selected by the host kernel.
// Sv48 provides 2^47 bytes of userspace; Sv39 provides 2^38 bytes.
var feasibleTaskSizes = []uintptr{
	uintptr(1) << 47,
	uintptr(1) << 38,
}