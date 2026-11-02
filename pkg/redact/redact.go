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

// Package redact creates a separate, fail-closed copy of collected data.
package redact

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/klauspost/compress/zstd"
)

const (
	maxDepth = 8
	// Bound decompression, including nested archives, across one operation.
	maxExpandedBytes int64 = 64 << 30
	maxEntries             = 1000000
	maxAddresses           = 65536
)

var ipLiteral = regexp.MustCompile(`(?i)(?:[0-9a-f]{0,4}:){2,}[0-9a-f:.]*(?:%[0-9a-z_.-]+)?|(?:[0-9]{1,3}\.){3}[0-9]{1,3}`)

// Report deliberately contains no source addresses or reversible mapping.
type Report struct {
	IPs                  int `json:"ips"`
	Files                int `json:"files"`
	Replacements         int `json:"replacements"`
	LogIPv4QuestionMarks int `json:"log_ipv4_question_marks"`
}

// Redactor holds a mapping shared by every file and archive entry in one run.
type Redactor struct {
	workers             int
	shared              *sharedState
	progress            *progressTracker
	mapping             map[netip.Addr]netip.Addr
	destinations        map[netip.Addr]bool
	report              Report
	remaining           int64
	entries             int
	aliases             []string
	automatic           bool
	scanOnly            bool
	verifying           bool
	discovered          map[netip.Addr]bool
	logIPv4QuestionMark bool
	inLogBody           bool
	keptIPv4Octets      int
}

// New validates addresses before collection starts. Sort order makes the same
// address list produce the same mapping, regardless of flags or file order.
func New(addresses []string) (*Redactor, error) {
	set := make(map[netip.Addr]bool)
	for _, raw := range addresses {
		addr, err := netip.ParseAddr(strings.TrimSpace(raw))
		if err != nil || addr.Zone() != "" {
			return nil, errors.New("redact-ips requires literal IPv4 or IPv6 addresses without ports, CIDRs or zones")
		}
		set[addr.Unmap()] = true
		if len(set) > maxAddresses {
			return nil, errors.New("redaction distinct IP limit exceeded")
		}
	}
	if len(set) == 0 {
		return nil, errors.New("redact-ips requires at least one address")
	}
	keys := make([]netip.Addr, 0, len(set))
	for addr := range set {
		keys = append(keys, addr)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].Less(keys[j]) })
	r := &Redactor{mapping: make(map[netip.Addr]netip.Addr), destinations: make(map[netip.Addr]bool)}
	v4 := netip.MustParseAddr("198.18.0.0")
	v6 := netip.MustParseAddr("2001:db8::")
	for _, addr := range keys {
		r.aliases = append(r.aliases, strings.ToLower(addr.String()), strings.ToLower(addr.StringExpanded()))
		candidate := &v6
		if addr.Is4() {
			candidate = &v4
		}
		for {
			*candidate = candidate.Next()
			if !candidate.IsValid() || (addr.Is4() && !netip.MustParsePrefix("198.18.0.0/15").Contains(*candidate)) {
				return nil, errors.New("too many addresses for the replacement range")
			}
			if !set[*candidate] {
				break
			}
		}
		r.mapping[addr] = *candidate
		r.destinations[*candidate] = true
	}
	return r, nil
}

func (r *Redactor) text(data []byte) ([]byte, error) {
	var collision, unexpected, logIPv4Remaining bool
	out := ipLiteral.ReplaceAllFunc(data, func(match []byte) []byte {
		literal := string(match)
		prefix, suffix := "", ""
		// A separator colon or filename suffix can border an IPv6 literal.
		if strings.HasPrefix(literal, ":") && !strings.HasPrefix(literal, "::") {
			prefix, literal = ":", literal[1:]
		}
		for strings.HasSuffix(literal, ".") {
			literal, suffix = strings.TrimSuffix(literal, "."), suffix+"."
		}
		addr, err := netip.ParseAddr(literal)
		if err != nil {
			return match
		}
		key := addr.WithZone("").Unmap()
		if r.discovered != nil {
			if r.shared != nil {
				r.shared.mu.Lock()
				defer r.shared.mu.Unlock()
			}
			r.discovered[key] = true
			return match
		}
		if r.verifying && r.inLogBody && key.Is4() {
			logIPv4Remaining = true
			return match
		}
		if replacement, ok := r.mapping[key]; ok {
			r.report.Replacements++
			if r.inLogBody && key.Is4() {
				r.report.LogIPv4QuestionMarks++
				return []byte(prefix + "?" + suffix)
			}
			value := replacement.String()
			if addr.Is4In6() {
				value = "::ffff:" + value
			}
			if addr.Zone() != "" {
				value += "%" + addr.Zone()
			}
			return []byte(prefix + value + suffix)
		}
		collision = collision || (!r.verifying && r.destinations[key])
		unexpected = unexpected || (r.automatic && !(r.verifying && r.destinations[key]))
		return match
	})
	if logIPv4Remaining {
		return nil, errors.New("an IPv4 literal remains in a question-mark log body")
	}
	if r.discoveryCount() > maxAddresses {
		return nil, errors.New("redaction distinct IP limit exceeded")
	}
	if unexpected {
		return nil, errors.New("an unrecognized IP appeared after discovery or remained during verification")
	}
	if collision {
		return nil, errors.New("a replacement IP already exists in the input; refusing to merge instance identities")
	}
	if err := r.checkAliases(out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Redactor) checkAliases(out []byte) error {
	// An independent literal check catches ambiguous regex contexts. Refuse
	// publication instead of silently leaving a known source address behind.
	lower := bytes.ToLower(out)
	// A short source literal such as ::1 may be a suffix of a different,
	// valid replacement IPv6 address. Recognize complete replacement spans.
	var allowed [][2]int
	if len(r.aliases) > 0 {
		for _, loc := range ipLiteral.FindAllIndex(out, -1) {
			literal := strings.TrimRight(string(out[loc[0]:loc[1]]), ".")
			if strings.HasPrefix(literal, ":") && !strings.HasPrefix(literal, "::") {
				literal = literal[1:]
			}
			addr, err := netip.ParseAddr(literal)
			if err == nil && r.destinations[addr.WithZone("").Unmap()] {
				allowed = append(allowed, [2]int{loc[0], loc[1]})
			}
		}
	}
	for _, alias := range r.aliases {
		offset := 0
		for rest := lower; ; {
			index := bytes.Index(rest, []byte(alias))
			if index < 0 {
				break
			}
			end := index + len(alias)
			// Do not mistake 10.1.1.10 for the selected 10.1.1.1.
			beforeDigit := index > 0 && rest[index-1] >= '0' && rest[index-1] <= '9'
			afterDigit := end < len(rest) && rest[end] >= '0' && rest[end] <= '9'
			afterIPv6 := strings.Contains(alias, ":") && end < len(rest) && (rest[end] == ':' || rest[end] >= 'a' && rest[end] <= 'f')
			inReplacement := false
			for _, span := range allowed {
				if span[0] <= offset+index && span[1] >= offset+end {
					inReplacement = true
					break
				}
			}
			if !beforeDigit && !afterDigit && !afterIPv6 && !inReplacement {
				return errors.New("a source IP literal remains in an ambiguous text context")
			}
			rest = rest[end:]
			offset += end
		}
	}
	return nil
}

func (r *Redactor) name(name string) (string, error) {
	value, err := r.text([]byte(name))
	return string(value), err
}

// Copy transforms input into a new sibling directory. Failed work is removed;
// the original input is never edited. Symlinks and special files are rejected.
func (r *Redactor) Copy(input, output string) (string, Report, error) {
	r.shared = nil
	r.report = Report{IPs: len(r.mapping)}
	r.remaining = maxExpandedBytes
	r.entries = 0
	input, err := filepath.Abs(input)
	if err != nil {
		return "", r.report, err
	}
	root, err := os.Lstat(input)
	if err != nil || !root.IsDir() {
		return "", r.report, errors.New("redaction input must be a real directory, not an archive or symlink")
	}
	if _, err := os.Lstat(filepath.Join(input, ".collect.lock")); !os.IsNotExist(err) {
		return "", r.report, errors.New("collection is incomplete or still running; refusing to redact")
	}
	if err := CheckInPlaceComplete(input); err != nil {
		return "", r.report, err
	}
	if output == "" {
		base, err := r.name(filepath.Base(input))
		if err != nil {
			return "", r.report, err
		}
		output = filepath.Join(filepath.Dir(input), base+"-redacted")
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return "", r.report, err
	}
	// Resolve parent symlinks before checking containment.
	resolvedInput, err := filepath.EvalSymlinks(input)
	if err != nil {
		return "", r.report, err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(output))
	if err != nil {
		return "", r.report, err
	}
	output = filepath.Join(parent, filepath.Base(output))
	rel, err := filepath.Rel(resolvedInput, output)
	if err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return "", r.report, errors.New("redaction output must be outside the input directory")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return "", r.report, errors.New("redaction output already exists or cannot be checked")
	}
	stage, err := os.MkdirTemp(parent, ".diag-redact-")
	if err != nil {
		return "", r.report, err
	}
	defer os.RemoveAll(stage)
	if err := r.startProgress(input, "Redacting"); err != nil {
		return "", r.report, err
	}
	seen := make(map[string]bool)
	pool := r.newPool()
	err = filepath.WalkDir(input, func(source string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if source == input {
			return nil
		}
		if err := r.entry(); err != nil {
			return err
		}
		relative, err := filepath.Rel(input, source)
		if err != nil {
			return err
		}
		name, err := r.name(filepath.ToSlash(relative))
		if err != nil {
			return err
		}
		if seen[name] {
			return errors.New("redaction caused a path collision")
		}
		seen[name] = true
		target := filepath.Join(stage, filepath.FromSlash(name))
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Mkdir(target, 0700)
		}
		if !info.Mode().IsRegular() {
			return errors.New("symlinks and special files cannot be safely redacted")
		}
		return pool.submit(func(worker *Redactor) error {
			in, err := os.Open(source)
			if err != nil {
				return err
			}
			defer in.Close()
			out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			transformErr := worker.transform(worker.progressReader(in), out, relative, 0)
			closeErr := out.Close()
			if err := errors.Join(transformErr, closeErr); err != nil {
				return fmt.Errorf("cannot safely redact file %q: %w", name, err)
			}
			worker.report.Files++
			worker.progressFileDone()
			return nil
		})
	})
	err = errors.Join(err, pool.finish(r))
	if err != nil {
		return "", r.report, err
	}
	r.finishProgress()
	if err := r.verify(stage); err != nil {
		return "", r.report, err
	}
	if err := os.Rename(stage, output); err != nil {
		return "", r.report, err
	}
	return output, r.report, nil
}

// Verify through the same container readers, including decompressed entry
// names and contents. Do not count the second pass in the user-facing report.
func (r *Redactor) verify(dir string) error {
	if err := r.startProgress(dir, "Verifying"); err != nil {
		return err
	}
	check := &Redactor{mapping: r.mapping, destinations: r.destinations, aliases: r.aliases, remaining: maxExpandedBytes, automatic: r.automatic, scanOnly: true, verifying: true, logIPv4QuestionMark: r.logIPv4QuestionMark, workers: r.workers, progress: r.progress}
	pool := check.newPool()
	err := filepath.WalkDir(dir, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		if _, err := check.name(filepath.ToSlash(relative)); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		return pool.submit(func(worker *Redactor) error {
			in, err := os.Open(file)
			if err != nil {
				return err
			}
			defer in.Close()
			if err := worker.transform(worker.progressReader(in), io.Discard, relative, 0); err != nil {
				return err
			}
			worker.progressFileDone()
			return nil
		})
	})
	err = errors.Join(err, pool.finish(check))
	if err != nil {
		return fmt.Errorf("redaction verification failed: %w", err)
	}
	if check.report.Replacements != 0 {
		return errors.New("source IP literals remain after redaction")
	}
	r.finishProgress()
	return nil
}

func (r *Redactor) entry() error {
	if r.shared != nil {
		r.shared.mu.Lock()
		defer r.shared.mu.Unlock()
		r.shared.entries++
		if r.shared.entries > maxEntries {
			return errors.New("redaction file count limit exceeded")
		}
		return nil
	}
	r.entries++
	if r.entries > maxEntries {
		return errors.New("redaction file count limit exceeded")
	}
	return nil
}

func (r *Redactor) transform(in io.Reader, out io.Writer, name string, depth int) error {
	if depth > maxDepth {
		return errors.New("redaction archive nesting limit exceeded")
	}
	reader := bufio.NewReader(in)
	header, _ := reader.Peek(512)
	switch {
	case bytes.HasPrefix(header, []byte{0x1f, 0x8b}):
		gz, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer gz.Close()
		if r.scanOnly {
			return r.transform(gz, io.Discard, strings.TrimSuffix(name, ".gz"), depth+1)
		}
		writer := gzip.NewWriter(out) // Drop source filename, comment and extra metadata.
		err = r.transform(gz, writer, strings.TrimSuffix(name, ".gz"), depth+1)
		return errors.Join(err, writer.Close())
	case bytes.HasPrefix(header, []byte{0x28, 0xb5, 0x2f, 0xfd}):
		decoder, err := zstd.NewReader(reader, zstd.WithDecoderMaxMemory(128<<20), zstd.WithDecoderConcurrency(1))
		if err != nil {
			return err
		}
		defer decoder.Close()
		name = strings.TrimSuffix(strings.TrimSuffix(name, ".zst"), ".zstd")
		if r.scanOnly {
			return r.transform(decoder, io.Discard, name, depth+1)
		}
		writer, err := zstd.NewWriter(out, zstd.WithEncoderConcurrency(1))
		if err != nil {
			return err
		}
		err = r.transform(decoder, writer, name, depth+1)
		return errors.Join(err, writer.Close())
	case bytes.HasPrefix(header, []byte("PK\x03\x04")), bytes.HasPrefix(header, []byte("PK\x05\x06")):
		return r.zip(reader, out, depth+1, name)
	case len(header) >= 262 && string(header[257:262]) == "ustar", strings.HasSuffix(strings.ToLower(name), ".tar"):
		return r.tar(reader, out, depth+1, name)
	default:
		for _, suffix := range []string{".gz", ".zst", ".zstd", ".zip", ".xz", ".bz2", ".7z", ".diag", ".tgz"} {
			if strings.HasSuffix(strings.ToLower(name), suffix) {
				return errors.New("unsupported or malformed compressed/encrypted file")
			}
		}
		return r.plainFile(reader, out, name)
	}
}

func ipByte(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || strings.ContainsRune(".:%_-", rune(b))
}

func (r *Redactor) plain(in io.Reader, out io.Writer) error {
	// Keep the last token across chunks so addresses split at 64 KiB survive.
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64<<10), 128<<10)
	scanner.Split(func(data []byte, eof bool) (int, []byte, error) {
		if eof && len(data) == 0 {
			return 0, nil, nil
		}
		if eof {
			return len(data), data, nil
		}
		if len(data) < 64<<10 {
			return 0, nil, nil
		}
		cut := len(data) - utf8.UTFMax
		for cut > 0 && (ipByte(data[cut-1]) || !utf8.RuneStart(data[cut])) {
			cut--
		}
		if cut == 0 {
			return 0, nil, errors.New("text token exceeds redaction buffer limit")
		}
		return cut, data[:cut], nil
	})
	for scanner.Scan() {
		data := scanner.Bytes()
		if !r.consume(int64(len(data))) {
			return errors.New("redaction expanded data limit exceeded")
		}
		if !utf8.Valid(data) {
			return errors.New("unsupported binary or non-UTF-8 content")
		}
		for _, b := range data {
			if b < 32 && b != '\n' && b != '\r' && b != '\t' && b != 27 {
				return errors.New("unsupported binary content")
			}
		}
		value, err := r.text(data)
		if err != nil {
			return err
		}
		if _, err := out.Write(value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func archiveName(name string) error {
	if strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.Clean(name) == ".." || strings.HasPrefix(path.Clean(name), "../") {
		return errors.New("unsafe archive path")
	}
	return nil
}

func (r *Redactor) tar(in io.Reader, out io.Writer, depth int, containerName string) error {
	reader, writer := tar.NewReader(in), tar.NewWriter(out)
	seen := make(map[string]bool)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return writer.Close()
		}
		if err != nil {
			return err
		}
		if err := r.entry(); err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeDir {
			return errors.New("archive links and special entries cannot be safely redacted")
		}
		if err := archiveName(header.Name); err != nil {
			return err
		}
		name, err := r.name(header.Name)
		if err != nil {
			return err
		}
		if seen[path.Clean(name)] {
			return errors.New("redaction caused an archive path collision")
		}
		seen[path.Clean(name)] = true
		originalName := archiveContext(containerName, header.Name)
		header.Name = name
		header.Linkname = ""
		header.PAXRecords, header.Xattrs = nil, nil
		header.Uname, header.Gname = "", ""
		if r.scanOnly {
			if header.Typeflag != tar.TypeDir {
				if err := r.transform(reader, io.Discard, originalName, depth); err != nil {
					return err
				}
			}
			continue
		}
		if header.Typeflag == tar.TypeDir {
			if err := writer.WriteHeader(header); err != nil {
				return err
			}
			continue
		}
		err = r.tarEntry(reader, writer, header, originalName, depth)
		if err != nil {
			return err
		}
	}
}

func (r *Redactor) tarEntry(in io.Reader, out *tar.Writer, header *tar.Header, name string, depth int) error {
	file, err := os.CreateTemp("", "diag-redact-entry-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := r.transform(in, file, name, depth); err != nil {
		return err
	}
	size, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	header.Size = size
	if err := out.WriteHeader(header); err != nil {
		return err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return err
	}
	_, err = io.Copy(out, file)
	return err
}

func (r *Redactor) zip(in io.Reader, out io.Writer, depth int, containerName string) error {
	file, err := os.CreateTemp("", "diag-redact-zip-")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	size, err := io.Copy(file, io.LimitReader(in, maxExpandedBytes+1))
	if err != nil {
		return err
	}
	if size > maxExpandedBytes {
		return errors.New("zip input exceeds redaction size limit")
	}
	reader, err := zip.NewReader(file, size)
	if err != nil {
		return err
	}
	writer := zip.NewWriter(out)
	seen := make(map[string]bool)
	for _, entry := range reader.File {
		if err := r.entry(); err != nil {
			return err
		}
		if err := archiveName(entry.Name); err != nil {
			return err
		}
		if !entry.Mode().IsRegular() && !entry.Mode().IsDir() {
			return errors.New("zip links and special entries cannot be safely redacted")
		}
		name, err := r.name(entry.Name)
		if err != nil {
			return err
		}
		if seen[path.Clean(name)] {
			return errors.New("redaction caused an archive path collision")
		}
		seen[path.Clean(name)] = true
		if r.scanOnly {
			if entry.FileInfo().IsDir() {
				continue
			}
			src, err := entry.Open()
			if err != nil {
				return err
			}
			err = r.transform(src, io.Discard, archiveContext(containerName, entry.Name), depth)
			closeErr := src.Close()
			if err := errors.Join(err, closeErr); err != nil {
				return err
			}
			continue
		}
		// Rebuild headers to omit original names, comments and arbitrary extras.
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(entry.Mode())
		dst, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}
		if entry.FileInfo().IsDir() {
			continue
		}
		src, err := entry.Open()
		if err != nil {
			return err
		}
		err = r.transform(src, dst, archiveContext(containerName, entry.Name), depth)
		closeErr := src.Close()
		if err := errors.Join(err, closeErr); err != nil {
			return err
		}
	}
	return writer.Close()
}
