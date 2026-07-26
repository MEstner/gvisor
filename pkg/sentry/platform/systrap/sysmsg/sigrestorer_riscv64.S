// Copyright 2020 The gVisor Authors.
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

#include <linux/unistd.h>

.global __export_restore_rt;
.type __export_restore_rt, @function;
__export_restore_rt:
  li a7, __NR_rt_sigreturn
  ecall
.size __export_restore_rt,.-__export_restore_rt

.global __export_restore_rt_from_frame;
.type __export_restore_rt_from_frame, @function;
// __export_restore_rt_from_frame performs rt_sigreturn after switching SP
// back to the kernel-created struct rt_sigframe.
//
// a0 is &rt_sigframe.info. info is the first field in rt_sigframe, so a0 is
// also the address of the complete frame.
__export_restore_rt_from_frame:
  mv sp, a0
  li a7, __NR_rt_sigreturn
  ecall
  // rt_sigreturn does not return. Trap if it unexpectedly does.
  ebreak
