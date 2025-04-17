// Copyright 2025 PingCAP, Inc.
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

package collector

import (
	"sync"
)

// WaitGroupWrapper is a wrapper for sync.WaitGroup
type WaitGroupWrapper struct {
	sync.WaitGroup
	PanicCnt int
}

// RunWithRecover wraps goroutine startup call with force recovery, add 1 to WaitGroup
// and call done when function return. it will dump current goroutine stack into log if catch any recover result.
// exec is that execute logic function. recoverFn is that handler will be called after recover and before dump stack,
// passing `nil` means noop.
func (w *WaitGroupWrapper) RunWithRecover(exec func(), recoverFn func(r any)) {
	w.Add(1)
	go func() {
		defer func() {
			r := recover()
			if recoverFn != nil {
				recoverFn(r)
			}
			if r != nil {
				w.PanicCnt++
			}
			w.Done()
		}()
		exec()
	}()
}
