// Copyright 2022 PingCAP, Inc.
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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pingcap/diag/pkg/utils/toml"
	"github.com/pingcap/tiup/pkg/cluster/spec"
	logprinter "github.com/pingcap/tiup/pkg/logger/printer"
)

// ProfileDirectoryName is the sub-path name storing profile config files
const ProfileDirectoryName = "profiles"

// CollectProfile is a pre-defined configuration of collecting jobs
type CollectProfile struct {
	Name        string   `toml:"name"` // name of the profile
	Version     string   `toml:"version"`
	Maintainers []string `toml:"maintainers,omitempty"`
	Description string   `toml:"description,omitempty"`
	Collectors  []string `toml:"collectors,omitempty"`
	Roles       []string `toml:"roles,omitempty"`
}

// readProfile tries to load a CollectProfile from file
func readProfile(name string) (*CollectProfile, error) {
	// try to parse input as a file path first
	if strings.HasSuffix(name, ".toml") {
		if cp, err := readProfileFile(name); err == nil {
			return cp, nil
		}
	}

	// then try user defined profiles
	if cp, err := readProfileFromDataDir(name); err == nil {
		return cp, nil
	}

	// then try pre-installed profiles
	if cp, err := readProfileFromComponentDir(name); err == nil {
		return cp, nil
	}

	return nil, fmt.Errorf("no valid collect profile with filename %s.toml found", name)
}

// readProfileFromDataDir tries to load a pre-installed profile file
func readProfileFromDataDir(name string) (*CollectProfile, error) {
	// try ~/.tiup/storage/diag/profiles/<name>.toml
	fp := spec.ProfilePath(
		ProfileDirectoryName,
		fmt.Sprintf("%s.toml", name),
	)
	return readProfileFile(fp)
}

// readProfileFromComponentDir tries to load a pre-installed profile file
func readProfileFromComponentDir(name string) (*CollectProfile, error) {
	// try ~/.tiup/components/diag/<version>/profiles/<name>.toml
	fp := filepath.Join(
		ProfileDirectoryName,
		fmt.Sprintf("%s.toml", name),
	)
	return readProfileFile(fp)
}

func readProfileFile(fp string) (*CollectProfile, error) {
	f, err := os.Open(fp)
	if err != nil {
		logprinter.Infof("error reading %s: %s", fp, err)
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		logprinter.Infof("error reading %s: %s", fp, err)
		return nil, err
	}

	var cp CollectProfile
	err = toml.Unmarshal(data, &cp)
	if err != nil {
		return nil, err
	}
	return &cp, nil
}
