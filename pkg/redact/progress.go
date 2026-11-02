// Copyright 2026 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redact

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"sync"
	"time"
)

// Progress describes one redaction pass. Byte counts refer to stored input,
// including compressed archives, rather than their expanded contents.
// It deliberately contains no file names or source addresses.
type Progress struct {
	Phase      string
	Files      int64
	TotalFiles int64
	Bytes      int64
	TotalBytes int64
	Done       bool
}

type progressTracker struct {
	mu sync.Mutex
	Progress
	callback func(Progress)
	last     time.Time
}

// SetProgress observes subsequent copy, rewrite and verification passes.
// Callbacks run synchronously; nil disables progress reporting.
func (r *Redactor) SetProgress(callback func(Progress)) {
	if callback == nil {
		r.progress = nil
		return
	}
	r.progress = &progressTracker{callback: callback}
}

func (r *Redactor) startProgress(dir, phase string) error {
	if r.progress == nil {
		return nil
	}
	p := r.progress
	p.Progress = Progress{Phase: phase}
	p.emit(true)
	// Count metadata only. Do not add another content/decompression pass.
	entries := 0
	err := filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		entries++
		if entries > maxEntries {
			return errors.New("redaction file count limit exceeded")
		}
		if entry.Type().IsRegular() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			p.TotalFiles++
			p.TotalBytes += info.Size()
		}
		p.emit(false)
		return nil
	})
	p.emit(true)
	return err
}

func (p *progressTracker) emit(force bool) {
	if force || time.Since(p.last) >= 200*time.Millisecond {
		p.callback(p.Progress)
		p.last = time.Now()
	}
}

func (r *Redactor) progressReader(in io.Reader) io.Reader {
	if r.progress == nil {
		return in
	}
	return &progressReader{Reader: in, tracker: r.progress}
}

type progressReader struct {
	io.Reader
	tracker *progressTracker
}

func (r *progressReader) Read(buf []byte) (int, error) {
	n, err := r.Reader.Read(buf)
	r.tracker.mu.Lock()
	defer r.tracker.mu.Unlock()
	r.tracker.Bytes += int64(n)
	r.tracker.emit(false)
	return n, err
}

func (r *Redactor) progressFileDone() {
	if r.progress != nil {
		r.progress.mu.Lock()
		defer r.progress.mu.Unlock()
		r.progress.Files++
		r.progress.emit(false)
	}
}

func (r *Redactor) finishProgress() {
	if r.progress != nil {
		r.progress.Done = true
		r.progress.emit(true)
	}
}
