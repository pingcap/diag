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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InPlaceMarker blocks packaging and further redaction after an interrupted
// destructive rewrite. It contains no original IP inventory.
const InPlaceMarker = ".redact-in-progress"

// CheckInPlaceComplete rejects directories that may contain mixed original and
// redacted data. A marker must never be removed merely to bypass this check.
func CheckInPlaceComplete(dir string) error {
	_, err := os.Lstat(filepath.Join(dir, InPlaceMarker))
	if err == nil {
		return errors.New("in-place redaction is incomplete; do not package or reuse this directory; recollect the data")
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}

type rewriteEntry struct {
	relative string
	info     os.FileInfo
}

// Rewrite replaces collected files one at a time without keeping a second
// full directory. One rewritten file and archive scratch space are still
// required. An interruption can leave mixed data; the marker remains then.
func (r *Redactor) Rewrite(input string) (string, Report, error) {
	r.report = Report{IPs: len(r.mapping)}
	input, err := filepath.Abs(input)
	if err != nil {
		return "", r.report, err
	}
	root, err := os.Lstat(input)
	if err != nil || !root.IsDir() {
		return "", r.report, errors.New("in-place redaction requires a real directory")
	}
	if err := CheckInPlaceComplete(input); err != nil {
		return "", r.report, err
	}
	if _, err := os.Lstat(filepath.Join(input, ".collect.lock")); !os.IsNotExist(err) {
		return "", r.report, errors.New("collection is incomplete or still running; refusing in-place redaction")
	}
	plan, err := r.prepareRewrite(input)
	if err != nil {
		return "", r.report, fmt.Errorf("in-place preflight failed; original data unchanged: %w", err)
	}
	if err := r.applyRewrite(input, plan); err != nil {
		return "", r.report, fmt.Errorf("in-place redaction failed; leave %s in place and do not package this directory: %w", InPlaceMarker, err)
	}
	return input, r.report, nil
}

func (r *Redactor) prepareRewrite(input string) ([]rewriteEntry, error) {
	if err := r.startProgress(input, "Checking in-place rewrite"); err != nil {
		return nil, err
	}
	check := *r
	check.shared = nil
	check.report = Report{}
	check.remaining = maxExpandedBytes
	check.entries = 0
	check.scanOnly = true
	check.verifying = false
	check.inLogBody = false
	var plan []rewriteEntry
	seen := make(map[string]bool)
	pool := check.newPool()
	err := filepath.WalkDir(input, func(file string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if file == input {
			return nil
		}
		if err := check.entry(); err != nil {
			return err
		}
		rel, err := filepath.Rel(input, file)
		if err != nil {
			return err
		}
		name, err := check.name(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		if seen[name] {
			return errors.New("redaction would cause a path collision")
		}
		seen[name] = true
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return errors.New("symlinks and special files cannot be rewritten")
		}
		plan = append(plan, rewriteEntry{relative: rel, info: info})
		if info.IsDir() {
			return nil
		}
		return pool.submit(func(worker *Redactor) error {
			in, err := os.Open(file)
			if err != nil {
				return err
			}
			defer in.Close()
			if err := worker.transform(worker.progressReader(in), io.Discard, rel, 0); err != nil {
				return err
			}
			worker.progressFileDone()
			return nil
		})
	})
	err = errors.Join(err, pool.finish(&check))
	if err == nil {
		r.finishProgress()
	}
	return plan, err
}

func (r *Redactor) applyRewrite(input string, plan []rewriteEntry) error {
	if err := r.startProgress(input, "Redacting in place"); err != nil {
		return err
	}
	marker, err := os.OpenFile(filepath.Join(input, InPlaceMarker), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(marker, "Incomplete local redaction. Do not package this directory.\n")
	syncErr := marker.Sync()
	if err := errors.Join(writeErr, syncErr, marker.Close()); err != nil {
		return err
	}
	r.report = Report{IPs: len(r.mapping)}
	r.remaining = maxExpandedBytes
	r.entries = 0
	r.shared = nil
	pool := r.newPool()
	var workErr error
	for _, entry := range plan {
		if err := r.entry(); err != nil {
			workErr = err
			break
		}
		if entry.info.IsDir() {
			continue
		}
		if err := pool.submit(func(worker *Redactor) error {
			if err := worker.rewriteFile(input, entry); err != nil {
				return err
			}
			worker.report.Files++
			worker.progressFileDone()
			return nil
		}); err != nil {
			workErr = err
			break
		}
	}
	if err := errors.Join(workErr, pool.finish(r)); err != nil {
		return err
	}
	// Children must be renamed before their parents. Source names remain
	// usable until their own turn, even for nested IP-bearing directories.
	sort.SliceStable(plan, func(i, j int) bool {
		return strings.Count(plan[i].relative, string(filepath.Separator)) > strings.Count(plan[j].relative, string(filepath.Separator))
	})
	for _, entry := range plan {
		base := filepath.Base(entry.relative)
		redacted, err := r.name(base)
		if err != nil {
			return err
		}
		if redacted == base {
			continue
		}
		source := filepath.Join(input, entry.relative)
		target := filepath.Join(filepath.Dir(source), redacted)
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			return errors.New("in-place rename target exists or cannot be checked")
		}
		if err := os.Rename(source, target); err != nil {
			return err
		}
	}
	r.finishProgress()
	if err := r.verify(input); err != nil {
		return err
	}
	return os.Remove(filepath.Join(input, InPlaceMarker))
}

func sameSource(expected, current os.FileInfo) bool {
	return current.Mode().IsRegular() && os.SameFile(expected, current) && expected.Size() == current.Size() && expected.ModTime().Equal(current.ModTime())
}

func (r *Redactor) rewriteFile(input string, entry rewriteEntry) error {
	source := filepath.Join(input, entry.relative)
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	if !sameSource(entry.info, info) {
		return errors.New("a collected file changed after preflight")
	}
	// Same-directory temporary files make replacement atomic on this volume.
	out, err := os.CreateTemp(filepath.Dir(source), ".diag-redact-file-")
	if err != nil {
		return err
	}
	defer os.Remove(out.Name())
	defer out.Close()
	if err := r.transform(r.progressReader(in), out, entry.relative, 0); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	current, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !sameSource(entry.info, current) {
		return errors.New("a collected file changed during redaction")
	}
	return os.Rename(out.Name(), source)
}
