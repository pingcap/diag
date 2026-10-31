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
	"path"
	"path/filepath"
	"strings"
)

// MaskLogIPv4 enables question marks in recognized log bodies only. All paths,
// configuration, topology and metrics still use the unique replacement IPs.
// Automatic discovery is required so unlisted IPv4 addresses cannot be skipped.
func (r *Redactor) MaskLogIPv4() error {
	if r.keptIPv4Octets != 0 {
		return errors.New("keep-ipv4-octets and hide-log-ipv4 are mutually exclusive")
	}
	if !r.automatic {
		return errors.New("log IPv4 question marks require automatic IP discovery")
	}
	r.logIPv4QuestionMark = true
	return nil
}

func (r *Redactor) plainFile(in io.Reader, out io.Writer, name string) error {
	previous := r.inLogBody
	r.inLogBody = r.logIPv4QuestionMark && isLogFile(name)
	defer func() { r.inLogBody = previous }()
	return r.plain(in, out)
}

// This is a logical classification label, never a filesystem extraction path.
// Entry names are validated with archiveName before this function is called.
func archiveContext(container, entry string) string {
	return strings.TrimSuffix(container, "/") + "/" + entry
}

// Use original names, including archive parents, to keep classification stable
// after IP path replacement. Known structural files always take precedence.
func isLogFile(name string) bool {
	name = strings.ToLower(filepath.ToSlash(name))
	base := path.Base(name)
	for _, suffix := range []string{".toml", ".yaml", ".yml"} {
		if strings.HasSuffix(base, suffix) {
			return false
		}
	}
	switch base {
	case "cluster.json", "tidbcluster.json", "tidbmonitor.json", "metrics.json", "alerts.json", "config.json":
		return false
	}
	parts := strings.Split(name, "/")
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "monitor", "metrics", "config", "configs", "db_vars", "sql_bind", "component_meta":
			return false
		}
	}
	if base == "log" || base == "dmesg" || base == "dmesg.txt" || strings.HasSuffix(base, ".log") || strings.Contains(base, ".log.") || strings.HasSuffix(base, ".stderr") || strings.HasSuffix(base, ".err") {
		return true
	}
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "log", "logs", "slowlog", "slowlogs", "cluster_audit", "dm_audit":
			return true
		}
	}
	return false
}
