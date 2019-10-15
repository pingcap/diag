// Copyright © 2016 Zlatko Čalušić
//
// Use of this source code is governed by an MIT-style license that can be found in the LICENSE file.

package sysinfo

import (
	"strings"
	"unsafe"
)

// https://en.wikipedia.org/wiki/CPUID#EAX.3D0:_Get_vendor_ID
var hvmap = map[string]string{
	"bhyve bhyve ": "bhyve",
	"KVMKVMKVM":    "kvm",
	"Microsoft Hv": "hyperv",
	"VMwareVMware": "vmware",
	"XenVMMXenVMM": "xenhvm",
}

// cpuid_amd64.s, cpuid_386.s, cpuid_default.s
func cpuid(info *[4]uint32, ax uint32)

func isHypervisorActive() bool {
	var info [4]uint32
	cpuid(&info, 0x1)
	return info[2]&(1<<31) != 0
}

func getHypervisorCpuid(ax uint32) string {
	var info [4]uint32
	cpuid(&info, ax)
	return hvmap[strings.TrimRight(string((*[12]byte)(unsafe.Pointer(&info[1]))[:]), "\000")]
}

func (si *SysInfo) getHypervisor() {
	if !isHypervisorActive() {
		if hypervisorType := slurpFile("/sys/hypervisor/type"); hypervisorType != "" {
			if hypervisorType == "xen" {
				si.Node.Hypervisor = "xenpv"
			}
		}
		return
	}

	// KVM has been caught to move its real signature to this leaf, and put something completely different in the
	// standard location. So this leaf must be checked first.
	if hv := getHypervisorCpuid(0x40000100); hv != "" {
		si.Node.Hypervisor = hv
		return
	}

	if hv := getHypervisorCpuid(0x40000000); hv != "" {
		si.Node.Hypervisor = hv
		return
	}

	si.Node.Hypervisor = "unknown"
}
