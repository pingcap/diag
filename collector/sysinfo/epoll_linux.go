// Copyright 2018 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

// Check if epoll exclusive available on the host
// Ported from https://github.com/pingcap/tidb-ansible/blob/v3.1.0/scripts/check/epoll_chk.cc

//go:build linux
// +build linux

package sysinfo

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// checkEpollExclusive checks if the host system support epoll exclusive mode
func checkEpollExclusive() bool {
	fd, err := syscall.EpollCreate1(syscall.EPOLL_CLOEXEC)
	if err != nil || fd < 0 {
		return false
	}
	defer syscall.Close(fd)

	evfd, _, evErr := syscall.Syscall(syscall.SYS_EVENTFD2, 0, uintptr(syscall.O_CLOEXEC), 0)
	if evErr != 0 || int(evfd) < 0 {
		return false
	}
	defer syscall.Close(int(evfd))

	/* choose events that should cause an error on
	   EPOLLEXCLUSIVE enabled kernels - specifically the combination of
	   EPOLLONESHOT and EPOLLEXCLUSIVE */
	ev := syscall.EpollEvent{
		Events: unix.EPOLLET |
			unix.EPOLLIN |
			unix.EPOLLEXCLUSIVE |
			unix.EPOLLONESHOT,
		//Fd: int32(fd),
	}
	if err := syscall.EpollCtl(fd, unix.EPOLL_CTL_ADD, int(evfd), &ev); err != nil {
		if err != syscall.EINVAL {
			return false
		} // else true
	} else {
		return false
	}

	return true
}
